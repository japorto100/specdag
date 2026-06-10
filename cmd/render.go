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
	// Filter aufbauen
	renderedNodes := make(map[string]bool)
	renderedEdges := make(map[int]bool)

	// Adjazenzliste für Pfadsuchen
	adj := make(map[string][]int) // Map von nodeID zu Indices der ausgehenden Edges
	for i, e := range depMap.Edges {
		adj[e.From] = append(adj[e.From], i)
	}

	switch view {
	case "orphans":
		// Orphans haben keine ein- oder ausgehenden Kanten
		hasEdges := make(map[string]bool)
		for _, e := range depMap.Edges {
			hasEdges[e.From] = true
			hasEdges[e.To] = true
		}
		for _, n := range depMap.Nodes {
			if !hasEdges[n.ID] {
				renderedNodes[n.ID] = true
			}
		}

	case "approvals":
		// Nur Approvals und deren direkte Nachbarn/Kanten
		for i, e := range depMap.Edges {
			fromNode := getNode(depMap.Nodes, e.From)
			toNode := getNode(depMap.Nodes, e.To)
			if (fromNode != nil && fromNode.Type == "approval") || (toNode != nil && toNode.Type == "approval") {
				renderedNodes[e.From] = true
				renderedNodes[e.To] = true
				renderedEdges[i] = true
			}
		}
		// Auch isolierte Approvals rendern
		for _, n := range depMap.Nodes {
			if n.Type == "approval" {
				renderedNodes[n.ID] = true
			}
		}

	case "events":
		// Nur Events, Contracts und deren direkte Verbindungen
		for _, n := range depMap.Nodes {
			if n.Type == "event" || n.Type == "contract" {
				renderedNodes[n.ID] = true
			}
		}
		for i, e := range depMap.Edges {
			if renderedNodes[e.From] && renderedNodes[e.To] {
				renderedEdges[i] = true
			}
		}

	case "verification":
		// Nur Expectation, Verifier, Artifact und deren Verbindungen
		for _, n := range depMap.Nodes {
			if n.Type == "expectation" || n.Type == "verifier" || n.Type == "artifact" {
				renderedNodes[n.ID] = true
			}
		}
		for i, e := range depMap.Edges {
			if renderedNodes[e.From] && renderedNodes[e.To] {
				renderedEdges[i] = true
			}
		}

	case "critical-path":
		// Alle Pfade von Intent zu Expectation
		visited := make(map[string]bool)
		var path []int
		var dfs func(u string) bool

		dfs = func(u string) bool {
			uNode := getNode(depMap.Nodes, u)
			if uNode != nil && uNode.Type == "expectation" {
				// Kanten auf dem gefundenen Pfad markieren
				for _, edgeIdx := range path {
					renderedEdges[edgeIdx] = true
					renderedNodes[depMap.Edges[edgeIdx].From] = true
					renderedNodes[depMap.Edges[edgeIdx].To] = true
				}
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
					path = path[:len(path)-1] // Backtrack
				}
			}
			visited[u] = false
			return reachedExpectation
		}

		for _, n := range depMap.Nodes {
			if n.Type == "intent" {
				dfs(n.ID)
			}
		}

	default: // "full"
		for _, n := range depMap.Nodes {
			renderedNodes[n.ID] = true
		}
		for i := range depMap.Edges {
			renderedEdges[i] = true
		}
	}

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
			buf.WriteString(fmt.Sprintf("    %s([\"%s\"])\n", id, label))
		case "expectation":
			buf.WriteString(fmt.Sprintf("    %s{{\"%s\"}}\n", id, label))
		case "event":
			buf.WriteString(fmt.Sprintf("    %s[/\"%s\"/]\n", id, label))
		case "command", "query", "job":
			buf.WriteString(fmt.Sprintf("    %s[\"%s\"]\n", id, label))
		case "artifact":
			buf.WriteString(fmt.Sprintf("    %s[(\"%s\")]\n", id, label))
		case "verifier":
			buf.WriteString(fmt.Sprintf("    %s(\"%s\")\n", id, label))
		case "approval":
			buf.WriteString(fmt.Sprintf("    %s{\"%s\"}\n", id, label))
		default:
			buf.WriteString(fmt.Sprintf("    %s[\"%s\"]\n", id, label))
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

		buf.WriteString(fmt.Sprintf("    %s -->|\"%s\"| %s\n", from, label, to))
	}

	buf.WriteString("```")
	return buf.String(), nil
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
