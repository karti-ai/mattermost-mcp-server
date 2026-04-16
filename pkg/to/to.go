package to

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func Result(data interface{}) *mcp.CallToolResult {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return Error(fmt.Errorf("failed to marshal result: %w", err))
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(content),
			},
		},
	}
}

func Text(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: text,
			},
		},
	}
}

func Error(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: err.Error(),
			},
		},
		IsError: true,
	}
}
