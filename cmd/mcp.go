package cmd

import (
	"context"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Starts the specdag tool as an MCP Server over Stdio",
	Run: func(cmd *cobra.Command, args []string) {
		// Neuen MCP Server erstellen
		s := server.NewMCPServer("specdag-mcp", "1.0.0")

		// 1. validate_map Tool registrieren
		validateTool := mcp.NewTool("validate_map",
			mcp.WithDescription("Validates a local dependency-map.yaml file for schema correctness and acyclic (cycle-free) constraints."),
			mcp.WithString("filePath", mcp.Required(), mcp.Description("Absolute or relative path to the dependency-map.yaml file")),
		)
		s.AddTool(validateTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			filePath, _ := req.RequireString("filePath")
			err := ValidateFile(filePath)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("FAIL: %v", err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("PASS: %s is valid and acyclic!", filePath)), nil
		})

		// 2. assemble_maps Tool registrieren
		assembleTool := mcp.NewTool("assemble_maps",
			mcp.WithDescription("Walks a specs directory, merges all feature dependency maps and checks for global cycles and node conflicts."),
			mcp.WithString("dirPath", mcp.Required(), mcp.Description("Path to the directory containing features (e.g. specs/features)")),
		)
		s.AddTool(assembleTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			dirPath, _ := req.RequireString("dirPath")
			output, err := AssembleDirectory(dirPath, "json", false)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("FAIL: %v", err)), nil
			}
			return mcp.NewToolResultText(output), nil
		})

		// 3. render_mermaid Tool registrieren
		renderTool := mcp.NewTool("render_mermaid",
			mcp.WithDescription("Renders a local dependency-map.yaml into a Mermaid markdown diagram string."),
			mcp.WithString("filePath", mcp.Required(), mcp.Description("Path to the dependency-map.yaml file")),
			mcp.WithString("view", mcp.Description("View filter: full (default), critical-path, approvals, events, verification, orphans")),
		)
		s.AddTool(renderTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			filePath, _ := req.RequireString("filePath")
			view := req.GetString("view", "full")
			output, err := RenderFile(filePath, view)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("FAIL: %v", err)), nil
			}
			return mcp.NewToolResultText(output), nil
		})

		// 4. get_rules Tool registrieren
		rulesTool := mcp.NewTool("get_rules",
			mcp.WithDescription("Returns the full ESDD (Event-Spec-Driven Development) skill rules, templates, and layouts."),
		)
		s.AddTool(rulesTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			fullRules, err := GetSkillRules()
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("FAIL: %v", err)), nil
			}
			return mcp.NewToolResultText(fullRules), nil
		})

		// 5. summary_map Tool registrieren
		summaryTool := mcp.NewTool("summary_map",
			mcp.WithDescription("Provides a textual summary of metrics, blocked approvals, orphans, unverified expectations, and the critical path of a map."),
			mcp.WithString("filePath", mcp.Required(), mcp.Description("Path to the dependency-map.yaml file")),
		)
		s.AddTool(summaryTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			filePath, _ := req.RequireString("filePath")
			summary, err := GetSummaryText(filePath)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("FAIL: %v", err)), nil
			}
			return mcp.NewToolResultText(summary), nil
		})

		// 6. analyze_impact Tool registrieren
		impactTool := mcp.NewTool("analyze_impact",
			mcp.WithDescription("Calculates all downstream nodes affected by changing a specific node ID in a map."),
			mcp.WithString("filePath", mcp.Required(), mcp.Description("Path to the dependency-map.yaml file")),
			mcp.WithString("nodeId", mcp.Required(), mcp.Description("Node ID to analyze downstream impact for")),
		)
		s.AddTool(impactTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			filePath, _ := req.RequireString("filePath")
			nodeID, _ := req.RequireString("nodeId")
			list, err := GetImpactList(filePath, nodeID)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("FAIL: %v", err)), nil
			}
			var builder strings.Builder
			builder.WriteString(fmt.Sprintf("=== Downstream Impact of changing %s ===\n", nodeID))
			if len(list) > 0 {
				for _, item := range list {
					builder.WriteString(item + "\n")
				}
			} else {
				builder.WriteString("No downstream dependencies (0 affected nodes).\n")
			}
			return mcp.NewToolResultText(builder.String()), nil
		})

		// 7. generate_report Tool registrieren
		reportTool := mcp.NewTool("generate_report",
			mcp.WithDescription("Generates a static HTML review report for a dependency map file or specs directory."),
			mcp.WithString("targetPath", mcp.Required(), mcp.Description("Path to the file or directory of maps")),
			mcp.WithString("outputPath", mcp.Required(), mcp.Description("Destination path for the generated HTML file")),
		)
		s.AddTool(reportTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			targetPath, _ := req.RequireString("targetPath")
			outputPath, _ := req.RequireString("outputPath")

			data, err := GenerateReportData(targetPath)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("FAIL: %v", err)), nil
			}

			tmpl, err := template.New("report").Parse(htmlTemplate)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("FAIL: template parse error: %v", err)), nil
			}

			out, err := os.Create(outputPath)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("FAIL: cannot create output file: %v", err)), nil
			}
			defer out.Close()

			if err := tmpl.Execute(out, data); err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("FAIL: template execute error: %v", err)), nil
			}

			return mcp.NewToolResultText(fmt.Sprintf("PASS: HTML report successfully generated at %s", outputPath)), nil
		})

		// 8. start_feature_flow Tool registrieren
		startFeatureTool := mcp.NewTool("start_feature_flow",
			mcp.WithDescription("Provides the structured ESDD checklist and question flow for starting a single-feature implementation."),
			mcp.WithString("featurePath", mcp.Required(), mcp.Description("Path to the feature folder (e.g. specs/features/012-agent-run)")),
		)
		s.AddTool(startFeatureTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			featurePath, _ := req.RequireString("featurePath")
			flow := fmt.Sprintf(`# ESDD Single Feature Flow: %s

Please answer these questions before implementing:
1. **Human Intent:** What value does this feature deliver to the user?
2. **Expectations:** What must happen? What must *never* happen? (Safety boundaries)
3. **Evidence:** Are there existing specs, code behaviors, or DB schemas that anchor this?
4. **Events/Contracts:** What events (past tense), commands, and contracts belong here?
5. **Jobs/Approvals:** What jobs, verifiers, and human-in-the-loop approvals are needed?
6. **No-Guessing:** What parameters or decisions must the agent NOT guess?
7. **Local Map:** Create or update 'dependency-map.yaml' (if Severity Level is 2+).
8. **Verification:** What checks or tests prove the feature is correct?

Suggested files in %s:
- spec.md (Goal, Requirements, Success criteria)
- tasks.md (Small checkable tasks)
- dependency-map.yaml (Local Spec-DAG)
- event-flow.md (Sequence diagram or event prose)
`, featurePath, featurePath)
			return mcp.NewToolResultText(flow), nil
		})

		// 9. start_integration_flow Tool registrieren
		startIntegrationTool := mcp.NewTool("start_integration_flow",
			mcp.WithDescription("Provides the ESDD checklist for integrating multiple features via shared events and contracts."),
			mcp.WithString("dirPath", mcp.Required(), mcp.Description("Path to the directory containing features (e.g. specs/features)")),
		)
		s.AddTool(startIntegrationTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			dirPath, _ := req.RequireString("dirPath")
			flow := fmt.Sprintf(`# ESDD Multi-Feature Integration Flow: %s

Please follow these integration steps:
1. **Identify Features:** Which local maps are involved in this integration?
2. **Shared Messages:** Which events or contracts connect them? Who produces, who consumes?
3. **Run Assembly:** Run 'specdag assemble %s' to merge feature maps and detect naming conflicts or global cycles.
4. **Impact Check:** If an event schema changes, use 'specdag impact' to find affected downstream nodes.
5. **Durable Truth:** Ensure shared events/contracts are moved to global catalogs under 'specs/events/' or 'specs/contracts/'.
6. **Cross-Check:** Run 'specdag check-catalogs %s' to verify metadata consistency.
7. **Report:** Generate a static HTML report ('specdag report %s') to review validation status and gaps.
`, dirPath, dirPath, dirPath, dirPath)
			return mcp.NewToolResultText(flow), nil
		})

		// 10. start_reconciliation_flow Tool registrieren
		startReconciliationTool := mcp.NewTool("start_reconciliation_flow",
			mcp.WithDescription("Provides the ESDD checklist for reconciling discrepancies between the intended Spec-DAG and observed code/run traces."),
			mcp.WithString("featurePath", mcp.Required(), mcp.Description("Path to the feature folder or map file")),
		)
		s.AddTool(startReconciliationTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			featurePath, _ := req.RequireString("featurePath")
			flow := fmt.Sprintf(`# ESDD Reconciliation Flow: %s

Use this when existing code, legacy specs, or runtime traces disagree with the intended Spec-DAG.

Please execute these steps:
1. **Spec-DAG Intent:** Review the normative 'dependency-map.yaml'. What is the desired behavior?
2. **Observed Implementation:** Use code intelligence graphs (e.g., GitNexus) to analyze the actual codebase and call chains.
3. **Observed Runtime:** Check execution logs and runtime traces (e.g. 'agent_run_trace').
4. **Identify Mismatches:** Where does code diverge from specs? (Missing implementation, undeclared dependency, missing verifier).
5. **Analyze Cause:** Is the code wrong, the spec wrong, or is a decision pending?
6. **Document Decisions:** Do NOT silently change the map. Record the mismatch as 'Evidence -> Implication -> Open Gap' and write the resolution to 'decisions.md' first.
7. **Implement & Verify:** Update code/spec and run verification tests.
`, featurePath)
			return mcp.NewToolResultText(flow), nil
		})

		// 11. review_dependency_map Tool registrieren
		reviewMapTool := mcp.NewTool("review_dependency_map",
			mcp.WithDescription("Reviews a local dependency map from a methodic ESDD perspective, checking nodes, edges, and obligations."),
			mcp.WithString("filePath", mcp.Required(), mcp.Description("Path to the dependency-map.yaml/json file")),
		)
		s.AddTool(reviewMapTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			filePath, _ := req.RequireString("filePath")
			
			// Laden und grobe checks
			depMap, err := LoadDependencyMap(filePath)
			if err != nil {
				return mcp.NewToolResultText(fmt.Sprintf("FAIL: Map cannot be parsed: %v", err)), nil
			}
			
			var suggestions []string
			hasIntent := false
			hasExpectation := false
			hasVerifier := false
			hasApproval := false
			
			for _, n := range depMap.Nodes {
				switch n.Type {
				case "intent":
					hasIntent = true
				case "expectation":
					hasExpectation = true
				case "verifier":
					hasVerifier = true
				case "approval":
					hasApproval = true
				}
			}
			
			if !hasIntent {
				suggestions = append(suggestions, "- Add an 'intent' node to define the human goal of the feature.")
			}
			if !hasExpectation {
				suggestions = append(suggestions, "- Add an 'expectation' node to define success/safety invariants.")
			}
			if !hasVerifier && hasExpectation {
				suggestions = append(suggestions, "- Add a 'verifier' node and link it to your 'expectation' via a 'verifies' edge.")
			}
			if !hasApproval {
				suggestions = append(suggestions, "- Check if a human-in-the-loop 'approval' node is required for this feature.")
			}
			
			// Kanten-Checks
			requiresApprovalCount := 0
			for _, e := range depMap.Edges {
				if e.Type == "requires_approval" {
					requiresApprovalCount++
				}
			}
			if requiresApprovalCount == 0 && hasApproval {
				suggestions = append(suggestions, "- You defined an approval node but no 'requires_approval' edge targets it.")
			}

			review := fmt.Sprintf(`# SpecDAG Methodic Review: %s

Status: %s | Nodes: %d | Edges: %d

Methodic Recommendations:
%s

Review Checklist:
- [ ] Does every event represent a true past-tense system occurrence?
- [ ] Are all verifiers linked to expectations (no isolated check-boxes)?
- [ ] Are contracts linked to events representing their schema propagation?
- [ ] Is there any evidence link that is currently undocumented?
`, filePath, depMap.Graph.Status, len(depMap.Nodes), len(depMap.Edges), strings.Join(suggestions, "\n"))
			
			return mcp.NewToolResultText(review), nil
		})

		// 12. migrate_feature_to_dag Tool registrieren
		migrateTool := mcp.NewTool("migrate_feature_to_dag",
			mcp.WithDescription("Guide for migrating an existing feature description or legacy specifications into an ESDD Spec-DAG."),
			mcp.WithString("featurePath", mcp.Required(), mcp.Description("Path to the feature directory")),
		)
		s.AddTool(migrateTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			featurePath, _ := req.RequireString("featurePath")
			guide := fmt.Sprintf(`# ESDD Migration Guide: %s

Please execute these steps to migrate legacy feature files:
1. **Identify legacy artifacts:** Locate the main readme, spec, or code files for this feature.
2. **Extract the Intent:** What was the user-facing outcome? Define a node:
   - type: intent
   - title: "User can <action>"
3. **Extract Success Criteria:** What invariants/rules were expected? Define nodes:
   - type: expectation
   - title: "<Condition> holds"
4. **Trace the Actions:** What APIs (contracts), messages (events), and processes (jobs) run here? Define their nodes.
5. **Find Approvals:** Does the legacy spec mention manual reviews or risk rules? Define 'approval' nodes.
6. **Identify Tests:** Which tests represent the verification of the expectations? Define 'verifier' nodes.
7. **Map the Edges:** Link them using ESDD edge types.
8. **Bootstrap:** Create 'dependency-map.yaml' in %s and run 'specdag validate'.
`, featurePath, featurePath)
			return mcp.NewToolResultText(guide), nil
		})

		// Starten über Stdio
		if err := server.ServeStdio(s); err != nil {
			fmt.Fprintf(os.Stderr, "Error running MCP server: %v\n", err)
			os.Exit(1)
		}
	},
}
