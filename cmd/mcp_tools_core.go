package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerCoreTools(s *server.MCPServer) {
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

	hashTool := mcp.NewTool("hash_map",
		mcp.WithDescription("Computes a deterministic SHA-256 Merkle-DAG attestation for a dependency map, including referenced evidence files."),
		mcp.WithString("filePath", mcp.Required(), mcp.Description("Path to the dependency-map.yaml or dependency-map.json file")),
	)
	s.AddTool(hashTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filePath, _ := req.RequireString("filePath")
		attestation, err := ComputeMerkleAttestation(filePath)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("FAIL: %v", err)), nil
		}
		output, err := json.MarshalIndent(attestation, "", "  ")
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("FAIL: %v", err)), nil
		}
		return mcp.NewToolResultText(string(output)), nil
	})

	verifyTool := mcp.NewTool("verify_attestation",
		mcp.WithDescription("Validates a dependency map and verifies that its Merkle root matches an expected hash."),
		mcp.WithString("filePath", mcp.Required(), mcp.Description("Path to the dependency-map.yaml or dependency-map.json file")),
		mcp.WithString("expectedRoot", mcp.Required(), mcp.Description("Expected Merkle root hash, with optional sha256: prefix")),
	)
	s.AddTool(verifyTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filePath, _ := req.RequireString("filePath")
		expectedRoot, _ := req.RequireString("expectedRoot")
		attestation, verified, err := VerifyMerkleRoot(filePath, expectedRoot)
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("FAIL: %v", err)), nil
		}
		output, err := json.MarshalIndent(buildVerificationResult(attestation, expectedRoot, verified), "", "  ")
		if err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("FAIL: %v", err)), nil
		}
		if !verified {
			return mcp.NewToolResultText(fmt.Sprintf("FAIL: %s", output)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("PASS: %s", output)), nil
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
		fmt.Fprintf(&builder, "=== Downstream Impact of changing %s ===\n", nodeID)
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

		if err := tmpl.Execute(out, data); err != nil {
			if closeErr := out.Close(); closeErr != nil {
				return mcp.NewToolResultText(fmt.Sprintf("FAIL: template execute error: %v; close error: %v", err, closeErr)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("FAIL: template execute error: %v", err)), nil
		}
		if err := out.Close(); err != nil {
			return mcp.NewToolResultText(fmt.Sprintf("FAIL: cannot close output file: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("PASS: HTML report successfully generated at %s", outputPath)), nil
	})
}
