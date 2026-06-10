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
			output, err := AssembleDirectory(dirPath)
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

		// Starten über Stdio
		if err := server.ServeStdio(s); err != nil {
			fmt.Fprintf(os.Stderr, "Error running MCP server: %v\n", err)
			os.Exit(1)
		}
	},
}
