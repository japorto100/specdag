package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type mcpServerEntry struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Type    string            `json:"type,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

type mcpConfig struct {
	MCPservers map[string]mcpServerEntry `json:"mcpServers"`
}

type agentInfo struct {
	Name       string
	ConfigPath string
}

var knownAgents = []agentInfo{
	{"claude-code", "~/.claude/mcp.json"},
	{"cursor", "~/.cursor/mcp.json"},
	{"codex", "~/.codex/mcp.json"},
	{"opencode", "~/.opencode/mcp.json"},
	{"aider", "~/.aider/mcp.json"},
	{"goose", "~/.config/goose/mcp.json"},
	{"vscode", "~/.vscode/mcp.json"},
	{"project-local", ".mcp.json"},
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Register specdag MCP server — interactive agent selector",
	Long: `Registers specdag as an MCP server.
You choose which CLI agents to install for.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		agentFlag, _ := cmd.Flags().GetString("agent")
		allFlag, _ := cmd.Flags().GetBool("all")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		var selected []agentInfo

		if dryRun && agentFlag == "" && !allFlag {
			// Dry-run with no agent selection: show all
			selected = knownAgents
		} else if allFlag {
			selected = knownAgents
		} else if agentFlag != "" {
			for _, a := range knownAgents {
				if a.Name == agentFlag {
					selected = append(selected, a)
					break
				}
			}
			if len(selected) == 0 {
				return fmt.Errorf("unknown agent: %s\nAvailable: %s", agentFlag, agentNames())
			}
		} else {
			selected = interactiveSelect()
		}

		if len(selected) == 0 {
			fmt.Fprintln(os.Stderr, "No agents selected. Nothing to do.")
			return nil
		}

		binPath, _ := os.Executable()
		entry := mcpServerEntry{
			Command: binPath,
			Args:    []string{"mcp"},
			Type:    "stdio",
		}

		registered := 0
		for _, a := range selected {
			path := expandHome(a.ConfigPath)
			if dryRun {
				fmt.Fprintf(os.Stderr, "[dry-run] would write to %s\n", path)
				continue
			}
			if err := registerInConfig(path, entry); err != nil {
				fmt.Fprintf(os.Stderr, "⚠  %s: %v\n", a.Name, err)
				continue
			}
			fmt.Fprintf(os.Stderr, "✅ %s → %s\n", a.Name, path)
			registered++
		}

		if !dryRun && registered > 0 {
			fmt.Fprintf(os.Stderr, "\nDone. Restart your agent(s) to pick up specdag.\n")
		}
		return nil
	},
}

func interactiveSelect() []agentInfo {
	reader := bufio.NewReader(os.Stdin)

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  specdag — MCP Agent Selector")
	fmt.Fprintln(os.Stderr, "  ─────────────────────────────")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  Select agents to install specdag MCP server for:")
	fmt.Fprintln(os.Stderr, "  (enter numbers separated by spaces, or 'all')")
	fmt.Fprintln(os.Stderr, "")

	for i, a := range knownAgents {
		marker := " "
		if _, err := os.Stat(expandHome(a.ConfigPath)); err == nil {
			marker = "✓"
		}
		fmt.Fprintf(os.Stderr, "  [%s] %d. %-16s %s\n", marker, i+1, a.Name, a.ConfigPath)
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprint(os.Stderr, "  > ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if strings.ToLower(input) == "all" {
		return knownAgents
	}

	var selected []agentInfo
	for _, part := range strings.Fields(input) {
		for i, a := range knownAgents {
			if part == fmt.Sprintf("%d", i+1) {
				selected = append(selected, a)
			}
		}
	}

	return selected
}

func agentNames() string {
	var names []string
	for _, a := range knownAgents {
		names = append(names, a.Name)
	}
	return strings.Join(names, ", ")
}

func registerInConfig(path string, entry mcpServerEntry) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	cfg := mcpConfig{MCPservers: make(map[string]mcpServerEntry)}

	data, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		if cfg.MCPservers == nil {
			cfg.MCPservers = make(map[string]mcpServerEntry)
		}
	}

	if existing, ok := cfg.MCPservers["specdag"]; ok {
		if existing.Command == entry.Command {
			return fmt.Errorf("already registered")
		}
	}

	cfg.MCPservers["specdag"] = entry

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, append(out, '\n'), 0o600)
}

func init() {
	setupCmd.Flags().StringP("agent", "a", "", "Install for specific agent only")
	setupCmd.Flags().BoolP("all", "A", false, "Install for all detected agents")
	setupCmd.Flags().Bool("dry-run", false, "Show what would be written without writing")
	mcpCmd.AddCommand(setupCmd)
}
