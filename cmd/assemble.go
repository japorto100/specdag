package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/japorto100/specdag/dag"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var outputFlag string

// AssembleDirectory sucht alle dependency-map.yaml Dateien im rootDir,
// merged diese zu einer globalen Map und prüft auf globale Zyklen.
// Gibt das JSON-Ergebnis als String zurück.
func AssembleDirectory(rootDir string) (string, error) {
	globalGraph := struct {
		Graph struct {
			ID       string `json:"id"`
			Kind     string `json:"kind"`
			Topology string `json:"topology,omitempty"`
			Status   string `json:"status"`
		} `json:"graph"`
		Nodes []dag.Node `json:"nodes"`
		Edges []dag.Edge `json:"edges"`
	}{}
	globalGraph.Graph.ID = "global-system-map"
	globalGraph.Graph.Kind = "global_dependency"
	globalGraph.Graph.Topology = "dag"
	globalGraph.Graph.Status = "accepted"

	nodeMap := make(map[string]dag.Node)
	var edges []dag.Edge

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (info.Name() == "dependency-map.yaml" || info.Name() == "dependency-map.yml") {
			// 1. Lokale Datei hart validieren
			if err := ValidateFile(path); err != nil {
				return fmt.Errorf("invalid dependency map at %s: %w", path, err)
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("error reading %s: %w", path, err)
			}

			var depMap dag.DependencyMap
			if err := yaml.Unmarshal(data, &depMap); err != nil {
				return fmt.Errorf("error parsing %s: %w", path, err)
			}

			// 2. Nodes mergen und semantische Konflikte prüfen (ID, Typ und Titel)
			for _, node := range depMap.Nodes {
				if existing, found := nodeMap[node.ID]; found {
					if existing.Type != node.Type || existing.Title != node.Title {
						return fmt.Errorf("node ID conflict: %s has conflicting definitions in different specs (type: '%s' vs '%s', title: '%s' vs '%s')",
							node.ID, existing.Type, node.Type, existing.Title, node.Title)
					}
				} else {
					nodeMap[node.ID] = node
				}
			}

			edges = append(edges, depMap.Edges...)
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	for _, node := range nodeMap {
		globalGraph.Nodes = append(globalGraph.Nodes, node)
	}
	globalGraph.Edges = edges

	// 3. Globale Kantenprüfung: Existieren alle Kanten-Endpunkte in der globalen Node-Map?
	for _, e := range globalGraph.Edges {
		if _, found := nodeMap[e.From]; !found {
			return "", fmt.Errorf("global edge references undeclared node ID: %s (in from)", e.From)
		}
		if _, found := nodeMap[e.To]; !found {
			return "", fmt.Errorf("global edge references undeclared node ID: %s (in to)", e.To)
		}
	}

	// 4. Globalen Graphen auf Zyklen checken
	g := dag.NewGraph()
	for _, n := range globalGraph.Nodes {
		g.AddNode(n)
	}
	for _, e := range globalGraph.Edges {
		g.AddEdge(e.From, e.To)
	}

	cycle, err := g.FindCycles()
	if err != nil {
		return "", fmt.Errorf("global cycle detected! Cycle path: %v", cycle)
	}

	outputBytes, err := json.MarshalIndent(globalGraph, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format global JSON: %w", err)
	}

	return string(outputBytes), nil
}

var assembleCmd = &cobra.Command{
	Use:   "assemble [dir]",
	Short: "Assembles multiple feature dependency-map.yaml files into a single global map",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rootDir := args[0]
		output, err := AssembleDirectory(rootDir)
		if err != nil {
			fmt.Printf("FAIL: %v\n", err)
			os.Exit(1)
		}

		if outputFlag != "" {
			if err := os.WriteFile(outputFlag, []byte(output), 0644); err != nil {
				fmt.Printf("FAIL: Failed to write output to %s: %v\n", outputFlag, err)
				os.Exit(1)
			}
			fmt.Printf("PASS: Successfully assembled global map into %s\n", outputFlag)
		} else {
			fmt.Println(output)
		}
	},
}

func init() {
	assembleCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Output JSON file path")
}
