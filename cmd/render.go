package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/japorto100/specdag/dag"

	"github.com/spf13/cobra"
)

var viewFlag string

// escapeMermaidLabel maskiert Anführungszeichen und Zeilenumbrüche
func escapeMermaidLabel(s string) string {
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

// RenderMap renders a DependencyMap struct into Mermaid diagram syntax, filtered by view.
func RenderMap(depMap *dag.DependencyMap, view string) (string, error) {
	renderedNodes, renderedEdges := selectRenderedElements(depMap, view)
	return renderMermaid(depMap, renderedNodes, renderedEdges), nil
}

func selectRenderedElements(depMap *dag.DependencyMap, view string) (map[string]bool, map[int]bool) {
	renderedNodes := make(map[string]bool)
	renderedEdges := make(map[int]bool)

	switch view {
	case "orphans":
		renderOrphans(depMap, renderedNodes)
	case "approvals":
		renderApprovals(depMap, renderedNodes, renderedEdges)
	case "events":
		renderNodeTypes(depMap, renderedNodes, renderedEdges, "event", "contract")
	case "verification":
		renderNodeTypes(depMap, renderedNodes, renderedEdges, "expectation", "verifier", "artifact")
	case "critical-path":
		renderCriticalPath(depMap, renderedNodes, renderedEdges)
	default:
		renderFull(depMap, renderedNodes, renderedEdges)
	}
	return renderedNodes, renderedEdges
}

func renderOrphans(depMap *dag.DependencyMap, renderedNodes map[string]bool) {
	hasEdges := make(map[string]bool)
	for _, edge := range depMap.Edges {
		hasEdges[edge.From] = true
		hasEdges[edge.To] = true
	}
	for _, node := range depMap.Nodes {
		if !hasEdges[node.ID] {
			renderedNodes[node.ID] = true
		}
	}
}

func renderApprovals(depMap *dag.DependencyMap, renderedNodes map[string]bool, renderedEdges map[int]bool) {
	for i, edge := range depMap.Edges {
		fromNode := getNode(depMap.Nodes, edge.From)
		toNode := getNode(depMap.Nodes, edge.To)
		if (fromNode != nil && fromNode.Type == "approval") || (toNode != nil && toNode.Type == "approval") {
			renderedNodes[edge.From] = true
			renderedNodes[edge.To] = true
			renderedEdges[i] = true
		}
	}
	for _, node := range depMap.Nodes {
		if node.Type == "approval" {
			renderedNodes[node.ID] = true
		}
	}
}

func renderNodeTypes(depMap *dag.DependencyMap, renderedNodes map[string]bool, renderedEdges map[int]bool, types ...string) {
	allowed := make(map[string]bool, len(types))
	for _, nodeType := range types {
		allowed[nodeType] = true
	}
	for _, node := range depMap.Nodes {
		if allowed[node.Type] {
			renderedNodes[node.ID] = true
		}
	}
	for i, edge := range depMap.Edges {
		if renderedNodes[edge.From] && renderedNodes[edge.To] {
			renderedEdges[i] = true
		}
	}
}

func renderCriticalPath(depMap *dag.DependencyMap, renderedNodes map[string]bool, renderedEdges map[int]bool) {
	adj := make(map[string][]int)
	for i, edge := range depMap.Edges {
		adj[edge.From] = append(adj[edge.From], i)
	}

	visited := make(map[string]bool)
	var path []int
	var dfs func(u string) bool

	dfs = func(u string) bool {
		if isNodeType(depMap.Nodes, u, "expectation") {
			markPath(depMap, path, renderedNodes, renderedEdges)
			renderedNodes[u] = true
			return true
		}

		visited[u] = true
		reachedExpectation := false
		for _, edgeIdx := range adj[u] {
			next := depMap.Edges[edgeIdx].To
			if !visited[next] {
				path = append(path, edgeIdx)
				if dfs(next) {
					reachedExpectation = true
				}
				path = path[:len(path)-1]
			}
		}
		visited[u] = false
		return reachedExpectation
	}

	for _, node := range depMap.Nodes {
		if node.Type == "intent" {
			dfs(node.ID)
		}
	}
}

func markPath(depMap *dag.DependencyMap, path []int, renderedNodes map[string]bool, renderedEdges map[int]bool) {
	for _, edgeIdx := range path {
		renderedEdges[edgeIdx] = true
		renderedNodes[depMap.Edges[edgeIdx].From] = true
		renderedNodes[depMap.Edges[edgeIdx].To] = true
	}
}

func renderFull(depMap *dag.DependencyMap, renderedNodes map[string]bool, renderedEdges map[int]bool) {
	for _, node := range depMap.Nodes {
		renderedNodes[node.ID] = true
	}
	for i := range depMap.Edges {
		renderedEdges[i] = true
	}
}

func renderMermaid(depMap *dag.DependencyMap, renderedNodes map[string]bool, renderedEdges map[int]bool) string {
	var buf bytes.Buffer
	buf.WriteString("```mermaid\n")
	buf.WriteString("graph TD\n")

	// Rendern der gefilterten Knoten
	nodeRendered := false
	for _, node := range depMap.Nodes {
		if !renderedNodes[node.ID] {
			continue
		}
		nodeRendered = true
		id := dag.GenerateMermaidID(node.ID)
		title := escapeMermaidLabel(node.Title)

		label := fmt.Sprintf("%s (%s)", title, node.Type)
		if node.Owner != "" {
			label = fmt.Sprintf("%s [%s]", label, escapeMermaidLabel(node.Owner))
		}

		switch node.Type {
		case "intent":
			fmt.Fprintf(&buf, "    %s([\"%s\"])\n", id, label)
		case "expectation":
			fmt.Fprintf(&buf, "    %s{{\"%s\"}}\n", id, label)
		case "event":
			fmt.Fprintf(&buf, "    %s[/\"%s\"/]\n", id, label)
		case "command", "query", "job":
			fmt.Fprintf(&buf, "    %s[\"%s\"]\n", id, label)
		case "artifact":
			fmt.Fprintf(&buf, "    %s[(\"%s\")]\n", id, label)
		case "verifier":
			fmt.Fprintf(&buf, "    %s(\"%s\")\n", id, label)
		case "approval":
			fmt.Fprintf(&buf, "    %s{\"%s\"}\n", id, label)
		default:
			fmt.Fprintf(&buf, "    %s[\"%s\"]\n", id, label)
		}
	}

	if !nodeRendered {
		buf.WriteString("    empty[\"No nodes found for this view\"]\n")
	}

	buf.WriteString("\n")

	// Rendern der gefilterten Kanten
	for i, edge := range depMap.Edges {
		if !renderedEdges[i] {
			continue
		}
		from := dag.GenerateMermaidID(edge.From)
		to := dag.GenerateMermaidID(edge.To)
		edgeType := escapeMermaidLabel(edge.Type)

		label := edgeType
		if edge.Condition != "" {
			label = fmt.Sprintf("%s (if: %s)", label, escapeMermaidLabel(edge.Condition))
		}
		if edge.Required {
			label = fmt.Sprintf("%s [REQ]", label)
		}

		fmt.Fprintf(&buf, "    %s -->|\"%s\"| %s\n", from, label, to)
	}

	buf.WriteString("```")
	return buf.String()
}

// RenderFile liest die dependency-map.yaml und gibt den Mermaid-String zurück, gefiltert nach view.
func RenderFile(filePath string, view string) (string, error) {
	depMap, err := LoadDependencyMap(filePath)
	if err != nil {
		return "", err
	}

	return RenderMap(depMap, view)
}

func getNode(nodes []dag.Node, id string) *dag.Node {
	for _, n := range nodes {
		if n.ID == id {
			return &n
		}
	}
	return nil
}

func isNodeType(nodes []dag.Node, id string, nodeType string) bool {
	node := getNode(nodes, id)
	return node != nil && node.Type == nodeType
}

var renderCmd = &cobra.Command{
	Use:   "render [file.yaml]",
	Short: "Renders the dependency map into Mermaid markdown diagram syntax",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		output, err := RenderFile(filePath, viewFlag)
		if err != nil {
			fmt.Printf("FAIL: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(output)
	},
}

func init() {
	renderCmd.Flags().StringVarP(&viewFlag, "view", "v", "full", "Mermaid view filter (full, critical-path, approvals, events, verification, orphans)")
}
