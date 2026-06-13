package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove specdag MCP server from agent configs",
	RunE: func(cmd *cobra.Command, args []string) error {
		agentFlag, _ := cmd.Flags().GetString("agent")
		allFlag, _ := cmd.Flags().GetBool("all")

		var targets []agentInfo
		if allFlag {
			targets = knownAgents
		} else if agentFlag != "" {
			for _, a := range knownAgents {
				if a.Name == agentFlag {
					targets = append(targets, a)
					break
				}
			}
			if len(targets) == 0 {
				return fmt.Errorf("unknown agent: %s", agentFlag)
			}
		} else {
			targets = interactiveSelect()
		}

		removed := 0
		for _, a := range targets {
			path := expandHome(a.ConfigPath)
			if err := removeFromConfig(path, "specdag"); err != nil {
				continue
			}
			fmt.Fprintf(os.Stderr, "✅ removed from %s\n", a.Name)
			removed++
		}

		if removed == 0 {
			fmt.Fprintln(os.Stderr, "specdag not found in any selected config.")
		} else {
			fmt.Fprintf(os.Stderr, "\nRemoved from %d config(s). Restart your agent(s).\n", removed)
		}
		return nil
	},
}

func removeFromConfig(path string, name string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	cfg := mcpConfig{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	if cfg.MCPservers == nil {
		return fmt.Errorf("not found")
	}

	if _, ok := cfg.MCPservers[name]; !ok {
		return fmt.Errorf("not found")
	}

	delete(cfg.MCPservers, name)

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, append(out, '\n'), 0o600)
}

func init() {
	uninstallCmd.Flags().StringP("agent", "a", "", "Remove from specific agent only")
	uninstallCmd.Flags().BoolP("all", "A", false, "Remove from all agents")
	mcpCmd.AddCommand(uninstallCmd)
}
