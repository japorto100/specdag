package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/japorto100/specdag/dag"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Hilfsfunktionen für erlaubte Knotentypen
func isValidNodeType(t string) bool {
	allowed := map[string]bool{
		"intent":      true,
		"expectation": true,
		"event":       true,
		"command":     true,
		"query":       true,
		"contract":    true,
		"job":         true,
		"artifact":    true,
		"verifier":    true,
		"approval":    true,
	}
	return allowed[t]
}

// Hilfsfunktionen für erlaubte Kantenbeziehungen
func isValidEdgeType(t string) bool {
	allowed := map[string]bool{
		"defines_success_for": true,
		"triggers":            true,
		"produces":            true,
		"consumes":            true,
		"verified_by":         true,
		"verifies":            true,
		"requires_approval":   true,
	}
	return allowed[t]
}

// Hilfsfunktion für erlaubte Status
func isValidStatus(s string) bool {
	allowed := map[string]bool{
		"draft":      true,
		"accepted":   true,
		"superseded": true,
	}
	return allowed[s]
}

// LoadDependencyMap lädt eine Map aus einer YAML- oder JSON-Datei.
func LoadDependencyMap(filePath string) (*dag.DependencyMap, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read file %s: %w", filePath, err)
	}

	var depMap dag.DependencyMap
	ext := strings.ToLower(filepath.Ext(filePath))
	trimmed := bytes.TrimSpace(data)
	isJSON := ext == ".json" || (len(trimmed) > 0 && trimmed[0] == '{')

	if isJSON {
		if err := json.Unmarshal(data, &depMap); err != nil {
			return nil, fmt.Errorf("invalid JSON syntax or structure in %s: %w", filePath, err)
		}
	} else {
		if err := yaml.Unmarshal(data, &depMap); err != nil {
			return nil, fmt.Errorf("invalid YAML syntax or structure in %s: %w", filePath, err)
		}
	}
	return &depMap, nil
}

// ValidateDependencyMap validiert die Struktur einer geladenen Dependency Map.
func ValidateDependencyMap(depMap *dag.DependencyMap) error {
	if depMap.Graph.ID == "" {
		return fmt.Errorf("graph.id is required")
	}

	// Valide Graph-Arten
	validKinds := map[string]bool{
		"spec_dependency":    true,
		"event_flow":         true,
		"agent_run_trace":    true,
		"execution_workflow": true,
	}
	if !validKinds[depMap.Graph.Kind] {
		return fmt.Errorf("invalid graph.kind: '%s' (must be spec_dependency, event_flow, agent_run_trace, or execution_workflow)", depMap.Graph.Kind)
	}

	// Topologie-Validierung
	if depMap.Graph.Topology != "" && depMap.Graph.Topology != "dag" && depMap.Graph.Topology != "graph" {
		return fmt.Errorf("invalid graph.topology: '%s' (must be 'dag' or 'graph')", depMap.Graph.Topology)
	}

	// Status-Validierung
	if !isValidStatus(depMap.Graph.Status) {
		return fmt.Errorf("invalid graph.status: '%s' (must be draft, accepted, or superseded)", depMap.Graph.Status)
	}

	// Nodes validieren
	if len(depMap.Nodes) == 0 {
		return fmt.Errorf("nodes list cannot be empty")
	}
	nodeSet := make(map[string]bool)
	for i, n := range depMap.Nodes {
		if n.ID == "" {
			return fmt.Errorf("node[%d].id is empty", i)
		}
		if nodeSet[n.ID] {
			return fmt.Errorf("duplicate node ID detected: %s", n.ID)
		}
		nodeSet[n.ID] = true

		if !isValidNodeType(n.Type) {
			return fmt.Errorf("invalid node type for '%s': '%s'", n.ID, n.Type)
		}
		if n.Title == "" {
			return fmt.Errorf("node '%s' has an empty title", n.ID)
		}
		if n.Status != "" && !isValidStatus(n.Status) {
			return fmt.Errorf("invalid node status for '%s': '%s'", n.ID, n.Status)
		}
	}

	// Edges validieren
	for i, e := range depMap.Edges {
		if e.From == "" || e.To == "" {
			return fmt.Errorf("edge[%d] has empty from or to", i)
		}
		if !nodeSet[e.From] {
			return fmt.Errorf("edge references undeclared node ID: %s (in from)", e.From)
		}
		if !nodeSet[e.To] {
			return fmt.Errorf("edge references undeclared node ID: %s (in to)", e.To)
		}
		if !isValidEdgeType(e.Type) {
			return fmt.Errorf("invalid edge type: '%s' (between %s and %s)", e.Type, e.From, e.To)
		}
	}

	// Zyklenerkennung (wenn topology nicht explizit auf "graph" gesetzt ist)
	isDAG := depMap.Graph.Topology != "graph"
	if isDAG {
		g := dag.NewGraph()
		for _, n := range depMap.Nodes {
			g.AddNode(n)
		}
		for _, e := range depMap.Edges {
			g.AddEdge(e.From, e.To)
		}

		cycle, err := g.FindCycles()
		if err != nil {
			return fmt.Errorf("graph is cyclic! Cycle path: %v", cycle)
		}
	}

	return nil
}

// ValidateFile lädt und validiert eine Map-Datei.
func ValidateFile(filePath string) error {
	depMap, err := LoadDependencyMap(filePath)
	if err != nil {
		return err
	}
	return ValidateDependencyMap(depMap)
}

var validateCmd = &cobra.Command{
	Use:   "validate [file.yaml|file.json]",
	Short: "Validates structural schema and logic constraints of a dependency map",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		if err := ValidateFile(filePath); err != nil {
			fmt.Printf("FAIL: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("PASS: %s is valid!\n", filePath)
	},
}
