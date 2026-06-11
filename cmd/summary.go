package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/japorto100/specdag/dag"
	"github.com/spf13/cobra"
)

// GetSummaryText liest die dependency-map.yaml und generiert einen Textreport.
func GetSummaryText(filePath string) (string, error) {
	depMap, err := LoadDependencyMap(filePath)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, "=== SpecDAG Summary: %s ===\n", depMap.Graph.ID)
	fmt.Fprintf(&buf, "Kind: %s | Status: %s | Topology: %s\n", depMap.Graph.Kind, depMap.Graph.Status, depMap.Graph.Topology)
	fmt.Fprintf(&buf, "Nodes: %d | Edges: %d\n\n", len(depMap.Nodes), len(depMap.Edges))

	orphans := getOrphanLines(depMap)
	buf.WriteString("--- Verwaiste Knoten (Orphans) ---\n")
	if len(orphans) > 0 {
		buf.WriteString(strings.Join(orphans, "\n") + "\n")
	} else {
		buf.WriteString("Keine verwaisten Knoten gefunden.\n")
	}
	buf.WriteString("\n")

	approvals := getApprovalLines(depMap)
	buf.WriteString("--- Freigabe-Schranken (Approvals) ---\n")
	if len(approvals) > 0 {
		buf.WriteString(strings.Join(approvals, "\n") + "\n")
	} else {
		buf.WriteString("Keine Freigabe-Schranken definiert.\n")
	}
	buf.WriteString("\n")

	missingVerifications := getMissingVerificationLines(depMap)
	buf.WriteString("--- Fehlende Verifikationen (Unverified Expectations) ---\n")
	if len(missingVerifications) > 0 {
		buf.WriteString(strings.Join(missingVerifications, "\n") + "\n")
	} else {
		buf.WriteString("Alle Erwartungen (Expectations) sind durch Verifier oder Events abgedeckt!\n")
	}
	buf.WriteString("\n")

	paths := getCriticalPaths(depMap, " -> ")
	buf.WriteString("--- Kritischer Kausalpfad (Critical Path) ---\n")
	if len(paths) > 0 {
		buf.WriteString(strings.Join(paths, "\n") + "\n")
	} else {
		buf.WriteString("Kein Kausalpfad von Intent zu Expectation gefunden.\n")
	}

	return buf.String(), nil
}

func getOrphanLines(depMap *dag.DependencyMap) []string {
	hasEdges := make(map[string]bool)
	for _, edge := range depMap.Edges {
		hasEdges[edge.From] = true
		hasEdges[edge.To] = true
	}

	var orphans []string
	for _, node := range depMap.Nodes {
		if !hasEdges[node.ID] {
			orphans = append(orphans, fmt.Sprintf("- %s (%s): \"%s\"", node.ID, node.Type, node.Title))
		}
	}
	return orphans
}

func getApprovalLines(depMap *dag.DependencyMap) []string {
	var approvals []string
	for _, edge := range depMap.Edges {
		if edge.Type == "requires_approval" {
			condition := "none"
			if edge.Condition != "" {
				condition = edge.Condition
			}
			approvals = append(approvals, fmt.Sprintf("- %s blockiert %s (Condition: %s)", edge.To, edge.From, condition))
		}
	}
	return approvals
}

func getMissingVerificationLines(depMap *dag.DependencyMap) []string {
	verifiedExpectations := getVerifiedExpectations(depMap)
	var missing []string
	for _, node := range depMap.Nodes {
		if node.Type == "expectation" && !verifiedExpectations[node.ID] {
			missing = append(missing, fmt.Sprintf("- %s: \"%s\"", node.ID, node.Title))
		}
	}
	return missing
}

func getVerifiedExpectations(depMap *dag.DependencyMap) map[string]bool {
	verifiedExpectations := make(map[string]bool)
	for _, edge := range depMap.Edges {
		if edge.Type != "verifies" {
			continue
		}
		fromNode := getNode(depMap.Nodes, edge.From)
		if fromNode != nil && (fromNode.Type == "verifier" || fromNode.Type == "event") {
			verifiedExpectations[edge.To] = true
		}
	}
	return verifiedExpectations
}

func getCriticalPaths(depMap *dag.DependencyMap, separator string) []string {
	adj := make(map[string][]string)
	for _, edge := range depMap.Edges {
		adj[edge.From] = append(adj[edge.From], edge.To)
	}

	var paths []string
	visited := make(map[string]bool)
	var currentPath []string

	var dfs func(u string)
	dfs = func(u string) {
		currentPath = append(currentPath, u)
		visited[u] = true

		if isNodeType(depMap.Nodes, u, "expectation") {
			paths = append(paths, strings.Join(currentPath, separator))
		} else {
			for _, v := range adj[u] {
				if !visited[v] {
					dfs(v)
				}
			}
		}

		currentPath = currentPath[:len(currentPath)-1]
		visited[u] = false
	}

	for _, node := range depMap.Nodes {
		if node.Type == "intent" {
			dfs(node.ID)
		}
	}
	return paths
}

var summaryCmd = &cobra.Command{
	Use:   "summary [file.yaml]",
	Short: "Prints a human-readable text report containing blockers, orphans, approvals, and coverage",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		output, err := GetSummaryText(filePath)
		if err != nil {
			fmt.Printf("FAIL: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(output)
	},
}
