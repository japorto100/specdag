package cmd

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Starts the specdag tool as an MCP Server over Stdio",
	Run: func(cmd *cobra.Command, args []string) {
		// Neuen MCP Server erstellen
		s := server.NewMCPServer("specdag-mcp", Version)

		// Ausgelagerte Registrierungs-Funktionen aufrufen
		registerCoreTools(s)
		registerRulesTools(s)
		registerFlowsTools(s)

		// Starten über Stdio
		if err := server.ServeStdio(s); err != nil {
			fmt.Fprintf(os.Stderr, "Error running MCP server: %v\n", err)
			os.Exit(1)
		}
	},
}
