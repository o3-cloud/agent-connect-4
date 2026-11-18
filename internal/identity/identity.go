// Package identity issues API keys to agents and recognises them (ADR-013,
// SEC-001..005). Keys are returned once, when issued; the registry keeps only
// their SHA-256 hashes, so a key it did not issue is never recognised.
package identity

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"sync"
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

// Registry holds registered agents, indexed by the hash of their key.
type Registry struct {
	mu     sync.RWMutex
	byHash map[[sha256.Size]byte]Agent
}

// NewRegistry returns an empty in-memory registry.
func NewRegistry() *Registry {
	return &Registry{byHash: map[[sha256.Size]byte]Agent{}}
}

// Register creates an agent and returns it with its API key. The key is not
// stored and cannot be retrieved again.
func (r *Registry) Register(displayName, agentType string) (Agent, string, error) {
	if n := utf8.RuneCountInString(displayName); n < 1 || n > maxDisplayName || (agentType != "human" && agentType != "agent") {
		return Agent{}, "", ErrInvalidAgent
	}
	secret := make([]byte, 32)
	idBytes := make([]byte, 8)
	_, _ = rand.Read(secret) // crypto/rand.Read never returns an error (Go 1.24+)
	_, _ = rand.Read(idBytes)
	key := KeyPrefix + base64.RawURLEncoding.EncodeToString(secret)
	agent := Agent{ID: "agt_" + hex.EncodeToString(idBytes), DisplayName: displayName, Type: agentType}

	r.mu.Lock()
	r.byHash[sha256.Sum256([]byte(key))] = agent
	r.mu.Unlock()
	return agent, key, nil
}

// Authenticate returns the agent that owns key, or false if the registry did
// not issue it.
func (r *Registry) Authenticate(key string) (Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agent, ok := r.byHash[sha256.Sum256([]byte(key))]
	return agent, ok
}
