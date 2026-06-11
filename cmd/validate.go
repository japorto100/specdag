package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
		"draft":       true,
		"accepted":    true,
		"implemented": true,
		"partial":     true,
		"blocked":     true,
		"deferred":    true,
		"superseded":  true,
		"archived":    true,
		"failed":      true,
	}
	return allowed[s]
}

func validStatusList() string {
	return "draft, accepted, implemented, partial, blocked, deferred, superseded, archived, or failed"
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
func ValidateDependencyMap(depMap *dag.DependencyMap, strict bool) error {
	if err := validateGraphMetadata(depMap); err != nil {
		return err
	}

	nodeMap, err := validateNodes(depMap.Nodes)
	if err != nil {
		return err
	}
	if err := validateEdges(depMap.Edges, nodeMap, strict); err != nil {
		return err
	}
	return validateTopology(depMap)
}

func validateGraphMetadata(depMap *dag.DependencyMap) error {
	if depMap.Graph.ID == "" {
		return fmt.Errorf("graph.id is required")
	}

	validKinds := map[string]bool{
		"spec_dependency":    true,
		"event_flow":         true,
		"agent_run_trace":    true,
		"execution_workflow": true,
	}
	if !validKinds[depMap.Graph.Kind] {
		return fmt.Errorf("invalid graph.kind: '%s' (must be spec_dependency, event_flow, agent_run_trace, or execution_workflow)", depMap.Graph.Kind)
	}

	if depMap.Graph.Topology != "" && depMap.Graph.Topology != "dag" && depMap.Graph.Topology != "graph" {
		return fmt.Errorf("invalid graph.topology: '%s' (must be 'dag' or 'graph')", depMap.Graph.Topology)
	}

	if !isValidStatus(depMap.Graph.Status) {
		return fmt.Errorf("invalid graph.status: '%s' (must be %s)", depMap.Graph.Status, validStatusList())
	}
	return nil
}

func validateNodes(nodes []dag.Node) (map[string]dag.Node, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("nodes list cannot be empty")
	}

	nodeMap := make(map[string]dag.Node)
	for i, node := range nodes {
		if node.ID == "" {
			return nil, fmt.Errorf("node[%d].id is empty", i)
		}
		if _, found := nodeMap[node.ID]; found {
			return nil, fmt.Errorf("duplicate node ID detected: %s", node.ID)
		}
		if !isValidNodeType(node.Type) {
			return nil, fmt.Errorf("invalid node type for '%s': '%s'", node.ID, node.Type)
		}
		if node.Title == "" {
			return nil, fmt.Errorf("node '%s' has an empty title", node.ID)
		}
		if node.Status != "" && !isValidStatus(node.Status) {
			return nil, fmt.Errorf("invalid node status for '%s': '%s' (must be %s)", node.ID, node.Status, validStatusList())
		}
		nodeMap[node.ID] = node
	}
	return nodeMap, nil
}

func validateEdges(edges []dag.Edge, nodeMap map[string]dag.Node, strict bool) error {
	for i, edge := range edges {
		fromNode, toNode, err := validateEdge(edge, nodeMap, i)
		if err != nil {
			return err
		}
		if strict {
			if err := validateStrictEdge(edge, fromNode, toNode); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateEdge(edge dag.Edge, nodeMap map[string]dag.Node, index int) (dag.Node, dag.Node, error) {
	if edge.From == "" || edge.To == "" {
		return dag.Node{}, dag.Node{}, fmt.Errorf("edge[%d] has empty from or to", index)
	}
	fromNode, fromFound := nodeMap[edge.From]
	if !fromFound {
		return dag.Node{}, dag.Node{}, fmt.Errorf("edge references undeclared node ID: %s (in from)", edge.From)
	}
	toNode, toFound := nodeMap[edge.To]
	if !toFound {
		return dag.Node{}, dag.Node{}, fmt.Errorf("edge references undeclared node ID: %s (in to)", edge.To)
	}
	if !isValidEdgeType(edge.Type) {
		return dag.Node{}, dag.Node{}, fmt.Errorf("invalid edge type: '%s' (between %s and %s)", edge.Type, edge.From, edge.To)
	}
	return fromNode, toNode, nil
}

func validateStrictEdge(edge dag.Edge, fromNode dag.Node, toNode dag.Node) error {
	valid := true
	switch edge.Type {
	case "defines_success_for":
		valid = isIntentToExpectation(fromNode, toNode)
	case "triggers":
		valid = isTriggerEdge(fromNode, toNode)
	case "produces":
		valid = isProducesEdge(fromNode, toNode)
	case "consumes":
		valid = isConsumesEdge(fromNode, toNode)
	case "verified_by":
		valid = isVerifiedByEdge(fromNode, toNode)
	case "verifies":
		valid = isVerifiesEdge(fromNode, toNode)
	case "requires_approval":
		valid = isRequiresApprovalEdge(fromNode, toNode)
	}
	if !valid {
		return strictEdgeMismatch(edge, fromNode, toNode)
	}
	return nil
}

func isIntentToExpectation(fromNode dag.Node, toNode dag.Node) bool {
	return fromNode.Type == "intent" && toNode.Type == "expectation"
}

func isTriggerEdge(fromNode dag.Node, toNode dag.Node) bool {
	return isAnyNodeType(fromNode, "event", "approval") && isAnyNodeType(toNode, "job", "command")
}

func isProducesEdge(fromNode dag.Node, toNode dag.Node) bool {
	return isAnyNodeType(fromNode, "job", "command", "contract") && isAnyNodeType(toNode, "artifact", "event")
}

func isConsumesEdge(fromNode dag.Node, toNode dag.Node) bool {
	return isAnyNodeType(fromNode, "job", "command") && isAnyNodeType(toNode, "artifact", "event", "contract")
}

func isVerifiedByEdge(fromNode dag.Node, toNode dag.Node) bool {
	return isAnyNodeType(fromNode, "artifact", "event", "job") && toNode.Type == "verifier"
}

func isVerifiesEdge(fromNode dag.Node, toNode dag.Node) bool {
	return isAnyNodeType(fromNode, "verifier", "event") && toNode.Type == "expectation"
}

func isRequiresApprovalEdge(fromNode dag.Node, toNode dag.Node) bool {
	return isAnyNodeType(fromNode, "job", "event", "command") && toNode.Type == "approval"
}

func isAnyNodeType(node dag.Node, nodeTypes ...string) bool {
	return slices.Contains(nodeTypes, node.Type)
}

func strictEdgeMismatch(edge dag.Edge, fromNode dag.Node, toNode dag.Node) error {
	expected := map[string]string{
		"defines_success_for": "intent -> expectation",
		"triggers":            "event/approval -> job/command",
		"produces":            "job/command/contract -> artifact/event",
		"consumes":            "job/command -> artifact/event/contract",
		"verified_by":         "artifact/event/job -> verifier",
		"verifies":            "verifier/event -> expectation",
		"requires_approval":   "job/event/command -> approval",
	}
	return fmt.Errorf("strict edge mismatch: %s must link %s, got %s (%s) -> %s (%s)",
		edge.Type, expected[edge.Type], edge.From, fromNode.Type, edge.To, toNode.Type)
}

func validateTopology(depMap *dag.DependencyMap) error {
	if depMap.Graph.Topology == "graph" {
		return nil
	}

	graph := dag.NewGraph()
	for _, node := range depMap.Nodes {
		graph.AddNode(node)
	}
	for _, edge := range depMap.Edges {
		graph.AddEdge(edge.From, edge.To)
	}

	cycle, err := graph.FindCycles()
	if err != nil {
		return fmt.Errorf("graph is cyclic! Cycle path: %v", cycle)
	}
	return nil
}

// ValidateDependencyMapNonStrict wrapper for backward compatibility
func ValidateDependencyMapNonStrict(depMap *dag.DependencyMap) error {
	return ValidateDependencyMap(depMap, false)
}

// ValidateFile lädt und validiert eine Map-Datei (nicht-strikt).
func ValidateFile(filePath string) error {
	return ValidateFileWithStrict(filePath, false)
}

// ValidateFileWithStrict lädt und validiert eine Map-Datei mit strict option.
func ValidateFileWithStrict(filePath string, strict bool) error {
	depMap, err := LoadDependencyMap(filePath)
	if err != nil {
		return err
	}
	return ValidateDependencyMap(depMap, strict)
}

var strictFlag bool

var validateCmd = &cobra.Command{
	Use:   "validate [file.yaml|file.json]",
	Short: "Validates structural schema and logic constraints of a dependency map",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		if err := ValidateFileWithStrict(filePath, strictFlag); err != nil {
			fmt.Printf("FAIL: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("PASS: %s is valid!\n", filePath)
	},
}

func init() {
	validateCmd.Flags().BoolVar(&strictFlag, "strict", false, "Enforce strict ESDD node and edge relationship rules")
}
