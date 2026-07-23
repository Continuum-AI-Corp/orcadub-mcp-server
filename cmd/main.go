// Command orcadub-mcp-server is an MCP stdio server exposing the OrcaDub video
// dubbing service (OrcaRouter model orca/dub) as MCP tools.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	dub "github.com/Continuum-AI-Corp/orcadub-mcp-server/internal"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// serverVersion is stamped by goreleaser at release time (-X main.serverVersion).
var serverVersion = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Println("orcadub-mcp-server " + serverVersion)
			return
		case "--help", "-h", "help":
			fmt.Println("orcadub — OrcaDub video dubbing.")
			fmt.Println("With no subcommand: runs as an MCP stdio server.")
			fmt.Println("CLI subcommands: health | upload | create | get | download (see `orcadub <cmd> -h`).")
			fmt.Println("Configuration: ORCADUB_API_KEY environment variable (https://www.orcarouter.ai/console).")
			fmt.Println("Docs: https://github.com/Continuum-AI-Corp/orcadub-mcp-server")
			return
		case "health", "upload", "create", "get", "download":
			os.Exit(dub.RunCLI(os.Args[1:]))
		}
	}
	// A missing ORCADUB_API_KEY does not block startup: tools register and
	// each call returns the OrcaRouter sign-up redirect until the key is set.
	cfg := dub.LoadConfig()
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "orcadub",
		Title:   "OrcaDub video dubbing",
		Version: serverVersion,
	}, nil)
	dub.RegisterTools(server, dub.NewClient(cfg))
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("orcadub-mcp-server: %v", err)
	}
}
