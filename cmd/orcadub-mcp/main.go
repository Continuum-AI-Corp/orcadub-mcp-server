// Command orcadub-mcp is an MCP stdio server exposing the OrcaDub video
// dubbing service (OrcaRouter model orca/dub) as MCP tools.
package main

import (
	"context"
	"log"

	"github.com/Continuum-AI-Corp/orcadub-mcp/internal/orcadub"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// serverVersion is stamped by goreleaser at release time (-X main.serverVersion).
var serverVersion = "dev"

func main() {
	// A missing ORCADUB_API_KEY does not block startup: tools register and
	// each call returns the OrcaRouter sign-up redirect until the key is set.
	cfg := orcadub.LoadConfig()
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "orcadub",
		Title:   "OrcaDub video dubbing",
		Version: serverVersion,
	}, nil)
	orcadub.RegisterTools(server, orcadub.NewClient(cfg))
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("orcadub-mcp: %v", err)
	}
}
