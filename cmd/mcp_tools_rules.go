package cmd

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerRulesTools(s *server.MCPServer) {
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
}
