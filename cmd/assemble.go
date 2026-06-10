package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/japorto100/specdag/dag"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var outputFlag string
var formatFlag string

// AssembleDirectory sucht alle dependency-map.* Dateien im rootDir,
// merged diese zu einer globalen Map und prüft auf globale Zyklen.
// Gibt das Ergebnis als String im gewünschten Format (json/yaml) zurück.
func AssembleDirectory(rootDir string, format string) (string, error) {
	globalGraph := struct {
		Graph struct {
			ID        string `yaml:"id" json:"id"`
			Kind      string `yaml:"kind" json:"kind"`
			Topology  string `yaml:"topology,omitempty" json:"topology,omitempty"`
			Status    string `yaml:"status" json:"status"`
			Scope     string `yaml:"scope,omitempty" json:"scope,omitempty"`
			Generated bool   `yaml:"generated" json:"generated"`
		} `yaml:"graph" json:"graph"`
		Nodes []dag.Node `yaml:"nodes" json:"nodes"`
		Edges []dag.Edge `yaml:"edges" json:"edges"`
	}{}
	globalGraph.Graph.ID = "global-system-map"
	globalGraph.Graph.Kind = "spec_dependency"
	globalGraph.Graph.Topology = "dag"
	globalGraph.Graph.Status = "accepted"
	globalGraph.Graph.Scope = "global"
	globalGraph.Graph.Generated = true

	nodeMap := make(map[string]dag.Node)
	var edges []dag.Edge

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (info.Name() == "dependency-map.yaml" || info.Name() == "dependency-map.yml" || info.Name() == "dependency-map.json") {
			// 1. Lokale Datei hart validieren
			if err := ValidateFile(path); err != nil {
				return fmt.Errorf("invalid dependency map at %s: %w", path, err)
			}

			// 2. Map laden mit zentraler Funktion
			depMap, err := LoadDependencyMap(path)
			if err != nil {
				return fmt.Errorf("error reading %s: %w", path, err)
			}

			// 3. Nodes mergen und semantische Konflikte prüfen (ID, Typ und Titel)
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

	// 4. Globale Kantenprüfung: Existieren alle Kanten-Endpunkte in der globalen Node-Map?
	for _, e := range globalGraph.Edges {
		if _, found := nodeMap[e.From]; !found {
			return "", fmt.Errorf("global edge references undeclared node ID: %s (in from)", e.From)
		}
		if _, found := nodeMap[e.To]; !found {
			return "", fmt.Errorf("global edge references undeclared node ID: %s (in to)", e.To)
		}
	}

	// 5. Globalen Graphen auf Zyklen checken
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

	// Formatierung
	var outputBytes []byte
	if strings.ToLower(format) == "yaml" {
		outputBytes, err = yaml.Marshal(globalGraph)
		if err != nil {
			return "", fmt.Errorf("failed to format global YAML: %w", err)
		}
	} else {
		outputBytes, err = json.MarshalIndent(globalGraph, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to format global JSON: %w", err)
		}
	}

	return string(outputBytes), nil
}

var assembleCmd = &cobra.Command{
	Use:   "assemble [dir]",
	Short: "Assembles multiple feature dependency maps (dependency-map.yaml/json) into a single global map",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rootDir := args[0]

		// Falls das Format nicht explizit gesetzt ist, aber der Output eine yaml-Endung hat,
		// wählen wir automatisch yaml als Format.
		format := formatFlag
		if format == "json" && outputFlag != "" {
			ext := strings.ToLower(filepath.Ext(outputFlag))
			if ext == ".yaml" || ext == ".yml" {
				format = "yaml"
			}
		}

		output, err := AssembleDirectory(rootDir, format)
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
	assembleCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Output file path")
	assembleCmd.Flags().StringVar(&formatFlag, "format", "json", "Output format (json or yaml)")
}
