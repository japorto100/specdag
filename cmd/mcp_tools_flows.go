package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerFlowsTools(s *server.MCPServer) {
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
}
