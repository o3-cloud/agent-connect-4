// Package identity issues API keys to agents and recognises them (ADR-013,
// SEC-001..005). Keys are returned once, when issued; the registry keeps only
// their SHA-256 hashes, so a key it did not issue is never recognised.
package identity

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
	"unicode/utf8"
)

// KeyPrefix starts every issued API key.
const KeyPrefix = "c4_"

const maxDisplayName = 64

// Agent is a registered identity. It never holds the key itself.
type Agent struct {
	ID          string
	DisplayName string
	Type        string
}

// ErrInvalidAgent reports a registration with a bad display name or type.
var ErrInvalidAgent = errors.New("displayName must be 1 to 64 characters and type must be human or agent")

// Store persists registrations (ADR-015). Append returns once the event is
// durable.
type Store interface {
	Append(eventType string, at time.Time, data any) error
}

// Journaled event types.
const (
	EventAgentRegistered = "agent_registered"
	// EventKeyReissued replaces an agent's key hash; only the house agent's
	// key is reissued, on start (ADR-014, ADR-015 step 6).
	EventKeyReissued = "agent_key_reissued"
)

// registeredEvent is journaled per registration. It holds the key's SHA-256
// hash, never the key (SEC-005).
type registeredEvent struct {
	AgentID     string `json:"agentId"`
	DisplayName string `json:"displayName"`
	Type        string `json:"type"`
	KeyHash     string `json:"keyHash"`
	House       bool   `json:"house,omitempty"`
}

type reissuedEvent struct {
	AgentID string `json:"agentId"`
	KeyHash string `json:"keyHash"`
}

// Registry holds registered agents, indexed by the hash of their key.
type Registry struct {
	mu     sync.RWMutex
	byHash map[[sha256.Size]byte]Agent
	hashOf map[string][sha256.Size]byte // agent ID -> current key hash
	house  *Agent
	// houseMu serialises HouseKey, so the house agent is registered once.
	houseMu sync.Mutex
	store   Store
	now     func() time.Time
}

// NewRegistry returns an empty in-memory registry.
func NewRegistry() *Registry {
	return &Registry{
		byHash: map[[sha256.Size]byte]Agent{},
		hashOf: map[string][sha256.Size]byte{},
		now:    func() time.Time { return time.Now().UTC() },
	}
}

// WithStore makes the registry journal every registration to store before it
// takes effect, and returns the registry.
func (r *Registry) WithStore(store Store) *Registry {
	r.store = store
	return r
}

// Register creates an agent and returns it with its API key. The key is not
// stored and cannot be retrieved again; only its hash is kept and journaled.
func (r *Registry) Register(displayName, agentType string) (Agent, string, error) {
	return r.register(displayName, agentType, false)
}

func (r *Registry) register(displayName, agentType string, house bool) (Agent, string, error) {
	if n := utf8.RuneCountInString(displayName); n < 1 || n > maxDisplayName || (agentType != "human" && agentType != "agent") {
		return Agent{}, "", ErrInvalidAgent
	}
	idBytes := make([]byte, 8)
	_, _ = rand.Read(idBytes) // crypto/rand.Read never returns an error (Go 1.24+)
	key, hash := newKey()
	agent := Agent{ID: "agt_" + hex.EncodeToString(idBytes), DisplayName: displayName, Type: agentType}

	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.persist(EventAgentRegistered, registeredEvent{
		AgentID: agent.ID, DisplayName: displayName, Type: agentType, KeyHash: hex.EncodeToString(hash[:]), House: house,
	}); err != nil {
		return Agent{}, "", err
	}
	r.add(agent, hash, house)
	return agent, key, nil
}

// HouseKey returns the house agent and a key for it (ADR-014). The first call
// ever registers the agent as displayName; later calls, including after a
// restart, keep the same agent and replace its key, so the house agent keeps
// one identity and its games while its key never leaves the process.
func (r *Registry) HouseKey(displayName string) (Agent, string, error) {
	r.houseMu.Lock()
	defer r.houseMu.Unlock()
	r.mu.Lock()
	house := r.house
	r.mu.Unlock()
	if house == nil {
		return r.register(displayName, "agent", true)
	}
	key, hash := newKey()
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.persist(EventKeyReissued, reissuedEvent{AgentID: house.ID, KeyHash: hex.EncodeToString(hash[:])}); err != nil {
		return Agent{}, "", err
	}
	r.rekey(house.ID, hash)
	return *house, key, nil
}

// House returns the house agent, if one has been registered.
func (r *Registry) House() (Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.house == nil {
		return Agent{}, false
	}
	return *r.house, true
}

// Restore applies one journaled event. It reports false for an event type
// that is not the registry's. Call it before the registry is used.
func (r *Registry) Restore(eventType string, data json.RawMessage) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch eventType {
	case EventAgentRegistered:
		var e registeredEvent
		hash, err := decode(data, &e, &e.KeyHash)
		if err != nil {
			return true, err
		}
		if _, ok := r.hashOf[e.AgentID]; ok || e.AgentID == "" {
			return true, fmt.Errorf("replay %s: agent %q registered twice or has no id", eventType, e.AgentID)
		}
		r.add(Agent{ID: e.AgentID, DisplayName: e.DisplayName, Type: e.Type}, hash, e.House)
	case EventKeyReissued:
		var e reissuedEvent
		hash, err := decode(data, &e, &e.KeyHash)
		if err != nil {
			return true, err
		}
		if _, ok := r.hashOf[e.AgentID]; !ok {
			return true, fmt.Errorf("replay %s: unknown agent %q", eventType, e.AgentID)
		}
		r.rekey(e.AgentID, hash)
	default:
		return false, nil
	}
	return true, nil
}

func decode(data json.RawMessage, into any, keyHash *string) ([sha256.Size]byte, error) {
	var hash [sha256.Size]byte
	if err := json.Unmarshal(data, into); err != nil {
		return hash, fmt.Errorf("replay agent event: %w", err)
	}
	raw, err := hex.DecodeString(*keyHash)
	if err != nil || len(raw) != sha256.Size {
		return hash, fmt.Errorf("replay agent event: keyHash is not a SHA-256 hash")
	}
	copy(hash[:], raw)
	return hash, nil
}

// add and rekey change the indexes; the caller holds r.mu.
func (r *Registry) add(agent Agent, hash [sha256.Size]byte, house bool) {
	r.byHash[hash] = agent
	r.hashOf[agent.ID] = hash
	if house {
		r.house = &agent
	}
}

func (r *Registry) rekey(agentID string, hash [sha256.Size]byte) {
	old := r.hashOf[agentID]
	agent := r.byHash[old]
	delete(r.byHash, old)
	r.byHash[hash] = agent
	r.hashOf[agentID] = hash
}

func (r *Registry) persist(eventType string, data any) error {
	if r.store == nil {
		return nil
	}
	if err := r.store.Append(eventType, r.now(), data); err != nil {
		return fmt.Errorf("save registration: %w", err)
	}
	return nil
}

func newKey() (string, [sha256.Size]byte) {
	secret := make([]byte, 32)
	_, _ = rand.Read(secret)
	key := KeyPrefix + base64.RawURLEncoding.EncodeToString(secret)
	return key, sha256.Sum256([]byte(key))
}

// Authenticate returns the agent that owns key, or false if the registry did
// not issue it.
func (r *Registry) Authenticate(key string) (Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agent, ok := r.byHash[sha256.Sum256([]byte(key))]
	return agent, ok
}
