package command

import (
	"context"
	"fmt"

	"github.com/karti-ai/mattermost-mcp-server/pkg/log"
	"github.com/karti-ai/mattermost-mcp-server/pkg/mattermost"
	"github.com/karti-ai/mattermost-mcp-server/pkg/params"
	"github.com/karti-ai/mattermost-mcp-server/pkg/to"
	"github.com/karti-ai/mattermost-mcp-server/pkg/tool"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var Tool = tool.New()

const (
	ExecuteSlashCommandToolName = "mattermost_execute_slash_command"
)

var (
	ExecuteSlashCommandTool = mcp.NewTool(
		ExecuteSlashCommandToolName,
		mcp.WithDescription("Execute a slash command in a channel (e.g., /remind, /poll)"),
		mcp.WithString("channel_id", mcp.Required(), mcp.Description("Channel ID to execute command in")),
		mcp.WithString("command", mcp.Required(), mcp.Description("Slash command to execute (e.g., /remind @channel meeting in 10 minutes)")),
	)
)

func init() {
	registerTools()
}

func registerTools() {
	tools := []server.ServerTool{
		{Tool: ExecuteSlashCommandTool, Handler: ExecuteSlashCommandFn},
	}
	for _, t := range tools {
		Tool.RegisterWrite(t)
	}
}

func ExecuteSlashCommandFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Command] Called ExecuteSlashCommandFn")

	args := req.GetArguments()

	channelID, err := params.GetString(args, "channel_id")
	if err != nil {
		return to.Error(fmt.Errorf("[channel_id] %v", err)), nil
	}

	command, err := params.GetString(args, "command")
	if err != nil {
		return to.Error(fmt.Errorf("[command] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	resp, err := client.ExecuteSlashCommand(ctx, channelID, command)
	if err != nil {
		return to.Error(fmt.Errorf("[command] failed to execute: %v", err)), nil
	}

	return to.Result(map[string]interface{}{
		"success":       true,
		"response":      resp.ResponseType,
		"text":          resp.Text,
		"goto_location": resp.GotoLocation,
	}), nil
}
