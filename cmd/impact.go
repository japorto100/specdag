package cmd

import (
	"fmt"
	"os"

	"github.com/japorto100/specdag/dag"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// GetImpactList berechnet alle Knoten, die transitiv vom targetNodeID abhängen.
func GetImpactList(filePath string, targetNodeID string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read file %s: %w", filePath, err)
	}

	var depMap dag.DependencyMap
	// yaml.Unmarshal kann sowohl YAML als auch JSON parsen, da JSON eine Untermenge von YAML ist.
	if err := yaml.Unmarshal(data, &depMap); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}

	// Prüfen, ob der Zielknoten existiert
	targetExists := false
	for _, n := range depMap.Nodes {
		if n.ID == targetNodeID {
			targetExists = true
			break
		}
	}
	if !targetExists {
		return nil, fmt.Errorf("node ID '%s' not found in dependency map", targetNodeID)
	}

	// Adjazenzliste aufbauen
	adj := make(map[string][]string)
	for _, e := range depMap.Edges {
		adj[e.From] = append(adj[e.From], e.To)
	}

	// DFS zur Bestimmung aller transitiv erreichbaren Nachfolger
	visited := make(map[string]bool)
	var impactList []string

	var dfs func(u string)
	dfs = func(u string) {
		visited[u] = true
		for _, v := range adj[u] {
			if !visited[v] {
				vNode := getNode(depMap.Nodes, v)
				typeStr := "unknown"
				titleStr := ""
				if vNode != nil {
					typeStr = vNode.Type
					titleStr = vNode.Title
				}
				impactList = append(impactList, fmt.Sprintf("- %s (%s): \"%s\"", v, typeStr, titleStr))
				dfs(v)
			}
		}
	}

	dfs(targetNodeID)
	return impactList, nil
}

var impactCmd = &cobra.Command{
	Use:   "impact [file.yaml] [node-id]",
	Short: "Analyzes the downstream impact of changing a specific node",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		targetNodeID := args[1]

		list, err := GetImpactList(filePath, targetNodeID)
		if err != nil {
			fmt.Printf("FAIL: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("=== Impact Analysis for: %s ===\n", targetNodeID)
		if len(list) > 0 {
			fmt.Printf("Changing '%s' will affect the following %d downstream nodes:\n", targetNodeID, len(list))
			for _, item := range list {
				fmt.Println(item)
			}
		} else {
			fmt.Printf("Changing '%s' has no downstream dependencies (0 affected nodes).\n", targetNodeID)
		}
	},
}
