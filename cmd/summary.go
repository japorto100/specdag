package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// GetSummaryText liest die dependency-map.yaml und generiert einen Textreport.
func GetSummaryText(filePath string) (string, error) {
	depMap, err := LoadDependencyMap(filePath)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("=== SpecDAG Summary: %s ===\n", depMap.Graph.ID))
	buf.WriteString(fmt.Sprintf("Kind: %s | Status: %s | Topology: %s\n", depMap.Graph.Kind, depMap.Graph.Status, depMap.Graph.Topology))
	buf.WriteString(fmt.Sprintf("Nodes: %d | Edges: %d\n\n", len(depMap.Nodes), len(depMap.Edges)))

	// 1. Verwaiste Knoten (Orphans) finden
	hasEdges := make(map[string]bool)
	for _, e := range depMap.Edges {
		hasEdges[e.From] = true
		hasEdges[e.To] = true
	}
	var orphans []string
	for _, n := range depMap.Nodes {
		if !hasEdges[n.ID] {
			orphans = append(orphans, fmt.Sprintf("- %s (%s): \"%s\"", n.ID, n.Type, n.Title))
		}
	}
	buf.WriteString("--- Verwaiste Knoten (Orphans) ---\n")
	if len(orphans) > 0 {
		buf.WriteString(strings.Join(orphans, "\n") + "\n")
	} else {
		buf.WriteString("Keine verwaisten Knoten gefunden.\n")
	}
	buf.WriteString("\n")

	// 2. Approval Gates
	var approvals []string
	for _, e := range depMap.Edges {
		if e.Type == "requires_approval" {
			condStr := "none"
			if e.Condition != "" {
				condStr = e.Condition
			}
			approvals = append(approvals, fmt.Sprintf("- %s blockiert %s (Condition: %s)", e.To, e.From, condStr))
		}
	}
	buf.WriteString("--- Freigabe-Schranken (Approvals) ---\n")
	if len(approvals) > 0 {
		buf.WriteString(strings.Join(approvals, "\n") + "\n")
	} else {
		buf.WriteString("Keine Freigabe-Schranken definiert.\n")
	}
	buf.WriteString("\n")

	// 3. Verification Coverage (Welche Expectations sind ungedeckt?)
	verifiedExpectations := make(map[string]bool)
	for _, e := range depMap.Edges {
		if e.Type == "verifies" {
			fromNode := getNode(depMap.Nodes, e.From)
			if fromNode != nil && (fromNode.Type == "verifier" || fromNode.Type == "event") {
				verifiedExpectations[e.To] = true
			}
		}
	}
	var missingVerifications []string
	for _, n := range depMap.Nodes {
		if n.Type == "expectation" {
			if !verifiedExpectations[n.ID] {
				missingVerifications = append(missingVerifications, fmt.Sprintf("- %s: \"%s\"", n.ID, n.Title))
			}
		}
	}
	buf.WriteString("--- Fehlende Verifikationen (Unverified Expectations) ---\n")
	if len(missingVerifications) > 0 {
		buf.WriteString(strings.Join(missingVerifications, "\n") + "\n")
	} else {
		buf.WriteString("Alle Erwartungen (Expectations) sind durch Verifier oder Events abgedeckt!\n")
	}
	buf.WriteString("\n")

	// 4. Critical Path
	adj := make(map[string][]string)
	for _, e := range depMap.Edges {
		adj[e.From] = append(adj[e.From], e.To)
	}

	var paths []string
	visited := make(map[string]bool)
	var currentPath []string

	var dfs func(u string)
	dfs = func(u string) {
		currentPath = append(currentPath, u)
		visited[u] = true

		uNode := getNode(depMap.Nodes, u)
		if uNode != nil && uNode.Type == "expectation" {
			paths = append(paths, strings.Join(currentPath, " -> "))
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

	for _, n := range depMap.Nodes {
		if n.Type == "intent" {
			dfs(n.ID)
		}
	}

	buf.WriteString("--- Kritischer Kausalpfad (Critical Path) ---\n")
	if len(paths) > 0 {
		buf.WriteString(strings.Join(paths, "\n") + "\n")
	} else {
		buf.WriteString("Kein Kausalpfad von Intent zu Expectation gefunden.\n")
	}

	return buf.String(), nil
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
