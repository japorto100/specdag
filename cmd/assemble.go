package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/japorto100/specdag/dag"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var outputFlag string
var formatFlag string

func AssembleDirectory(rootDir string, format string, includeGraphs bool) (string, error) {
	globalGraph := newGlobalDependencyMap()
	nodeMap := make(map[string]dag.Node)

	edges, hasGraphTopology, err := collectAssemblyInputs(rootDir, nodeMap, includeGraphs)
	if err != nil {
		return "", err
	}

	if hasGraphTopology {
		globalGraph.Graph.Topology = "graph"
	}
	globalGraph.Nodes = sortedNodes(nodeMap)
	globalGraph.Edges = edges

	if err := validateGlobalEdges(nodeMap, globalGraph.Edges); err != nil {
		return "", err
	}
	if err := validateGlobalCycles(&globalGraph); err != nil {
		return "", err
	}

	return formatDependencyMap(globalGraph, format)
}

func newGlobalDependencyMap() dag.DependencyMap {
	globalGraph := dag.DependencyMap{}
	globalGraph.Graph.ID = "global-system-map"
	globalGraph.Graph.Kind = "spec_dependency"
	globalGraph.Graph.Topology = "dag"
	globalGraph.Graph.Status = "accepted"
	globalGraph.Graph.Scope = "global"
	globalGraph.Graph.Generated = true
	return globalGraph
}

func collectAssemblyInputs(rootDir string, nodeMap map[string]dag.Node, includeGraphs bool) ([]dag.Edge, bool, error) {
	var edges []dag.Edge
	hasGraphTopology := false

	walkErr := filepath.Walk(rootDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk %s: %w", path, walkErr)
		}
		if info.IsDir() || !isDependencyMapFilename(info.Name()) {
			return nil
		}

		mapEdges, graphTopology, err := loadAssemblyMap(path, nodeMap, includeGraphs)
		if err != nil {
			return err
		}
		hasGraphTopology = hasGraphTopology || graphTopology
		edges = append(edges, mapEdges...)
		return nil
	})
	if walkErr != nil {
		return nil, false, fmt.Errorf("walk dependency maps in %s: %w", rootDir, walkErr)
	}

	sortEdges(edges)
	return edges, hasGraphTopology, nil
}

func loadAssemblyMap(path string, nodeMap map[string]dag.Node, includeGraphs bool) ([]dag.Edge, bool, error) {
	if err := ValidateFile(path); err != nil {
		return nil, false, fmt.Errorf("invalid dependency map at %s: %w", path, err)
	}

	depMap, err := LoadDependencyMap(path)
	if err != nil {
		return nil, false, fmt.Errorf("error reading %s: %w", path, err)
	}

	if depMap.Graph.Topology == "graph" && !includeGraphs {
		fmt.Fprintf(os.Stderr, "WARN: Skipping dependency map at %s because topology is 'graph' (use --include-graphs to include)\n", path)
		return nil, false, nil
	}
	if err := mergeNodes(nodeMap, depMap.Nodes); err != nil {
		return nil, false, err
	}

	return depMap.Edges, depMap.Graph.Topology == "graph", nil
}

func mergeNodes(nodeMap map[string]dag.Node, nodes []dag.Node) error {
	for _, node := range nodes {
		existing, found := nodeMap[node.ID]
		if !found {
			nodeMap[node.ID] = node
			continue
		}
		if existing.Type != node.Type || existing.Title != node.Title {
			return fmt.Errorf("node ID conflict: %s has conflicting definitions in different specs (type: '%s' vs '%s', title: '%s' vs '%s')",
				node.ID, existing.Type, node.Type, existing.Title, node.Title)
		}
	}
	return nil
}

func validateGlobalEdges(nodeMap map[string]dag.Node, edges []dag.Edge) error {
	for _, edge := range edges {
		if _, found := nodeMap[edge.From]; !found {
			return fmt.Errorf("global edge references undeclared node ID: %s (in from)", edge.From)
		}
		if _, found := nodeMap[edge.To]; !found {
			return fmt.Errorf("global edge references undeclared node ID: %s (in to)", edge.To)
		}
	}
	return nil
}

func validateGlobalCycles(globalGraph *dag.DependencyMap) error {
	if globalGraph.Graph.Topology == "graph" {
		return nil
	}

	graph := dag.NewGraph()
	for _, node := range globalGraph.Nodes {
		graph.AddNode(node)
	}
	for _, edge := range globalGraph.Edges {
		graph.AddEdge(edge.From, edge.To)
	}

	cycle, err := graph.FindCycles()
	if err != nil {
		return fmt.Errorf("global cycle detected! Cycle path: %v", cycle)
	}
	return nil
}

func formatDependencyMap(depMap dag.DependencyMap, format string) (string, error) {
	var outputBytes []byte
	var err error
	if strings.ToLower(format) == "yaml" {
		outputBytes, err = yaml.Marshal(depMap)
		if err != nil {
			return "", fmt.Errorf("failed to format global YAML: %w", err)
		}
	} else {
		outputBytes, err = json.MarshalIndent(depMap, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to format global JSON: %w", err)
		}
	}

	return string(outputBytes), nil
}

func isDependencyMapFilename(name string) bool {
	return name == "dependency-map.yaml" || name == "dependency-map.yml" || name == "dependency-map.json"
}

func sortedNodes(nodeMap map[string]dag.Node) []dag.Node {
	nodes := make([]dag.Node, 0, len(nodeMap))
	for _, node := range nodeMap {
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})
	return nodes
}

func sortEdges(edges []dag.Edge) {
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}
		if edges[i].Type != edges[j].Type {
			return edges[i].Type < edges[j].Type
		}
		if edges[i].Condition != edges[j].Condition {
			return edges[i].Condition < edges[j].Condition
		}
		return !edges[i].Required && edges[j].Required
	})
}

var includeGraphsFlag bool

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

		output, err := AssembleDirectory(rootDir, format, includeGraphsFlag)
		if err != nil {
			fmt.Printf("FAIL: %v\n", err)
			os.Exit(1)
		}

		if outputFlag != "" {
			if err := os.WriteFile(outputFlag, []byte(output), 0600); err != nil {
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
	assembleCmd.Flags().BoolVar(&includeGraphsFlag, "include-graphs", false, "Include dependency maps with topology 'graph' in the assembly")
}
