package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// mcpCommand is ADR-009 decision item 3: an MCP server on stdin and stdout for
// agents that launch MCP servers as subprocesses. It is a proxy. It connects to
// the arena's /mcp endpoint as the CONNECT4_API_KEY agent, offers exactly the
// tools the arena lists, and forwards each call unchanged. The arena stays the
// one authority (MCP-009), and results and error objects are the arena's own.
func mcpCommand(o *options) *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Serve the arena's MCP tools over stdio, for MCP clients that launch a subprocess",
		Long: "mcp speaks MCP on stdin and stdout and forwards every tool call to the arena's /mcp endpoint\n" +
			"($CONNECT4_SERVER), authenticated with $CONNECT4_API_KEY. It offers the same tools and returns\n" +
			"the same results and errors as /mcp. It exits when stdin closes.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			return o.serveMCP(ctx, cmd.InOrStdin(), cmd.OutOrStdout())
		},
	}
}

// bearer adds the API key to each request to the arena and notes a 401, so a
// rejected key can be reported as UNAUTHENTICATED.
type bearer struct {
	key          string
	next         http.RoundTripper
	unauthorized *atomic.Bool
}

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.key)
	response, err := b.next.RoundTrip(r)
	if err == nil && response.StatusCode == http.StatusUnauthorized {
		b.unauthorized.Store(true)
	}
	return response, err
}

func (o *options) serveMCP(ctx context.Context, in io.Reader, out io.Writer) error {
	c := o.client()
	if c.key == "" {
		return &apiError{Code: "UNAUTHENTICATED",
			Message: "connect4 mcp needs an API key: set CONNECT4_API_KEY (get one with `connect4 register --name <name>`)"}
	}
	endpoint := strings.TrimSuffix(c.base, "/api/v1") + "/mcp"
	unauthorized := &atomic.Bool{}
	transport := &mcp.StreamableClientTransport{
		Endpoint:             endpoint,
		HTTPClient:           &http.Client{Transport: bearer{key: c.key, next: http.DefaultTransport, unauthorized: unauthorized}},
		MaxRetries:           -1,
		DisableStandaloneSSE: true,
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "connect4-cli", Version: "1.0.0"}, nil)
	remote, err := client.Connect(ctx, transport, nil)
	if err != nil {
		if unauthorized.Load() {
			return &apiError{Code: "UNAUTHENTICATED", Message: "the arena server rejected CONNECT4_API_KEY"}
		}
		return &apiError{Code: "INTERNAL", Retryable: true,
			Message: fmt.Sprintf("cannot reach the arena server at %s: %v", endpoint, err)}
	}
	defer func() { _ = remote.Close() }()

	server := mcp.NewServer(&mcp.Implementation{Name: "connect4-arena", Version: "1.0.0"}, nil)
	for tool, err := range remote.Tools(ctx, nil) {
		if err != nil {
			return &apiError{Code: "INTERNAL", Retryable: true, Message: "listing the arena's MCP tools: " + err.Error()}
		}
		server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return remote.CallTool(ctx, &mcp.CallToolParams{Name: req.Params.Name, Arguments: req.Params.Arguments})
		})
	}

	err = server.Run(ctx, &mcp.IOTransport{Reader: io.NopCloser(in), Writer: nopWriteCloser{out}})
	if err == nil || errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
		return nil // stdin closed: the client is done
	}
	return &apiError{Code: "INTERNAL", Message: "MCP over stdio: " + err.Error()}
}

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }
