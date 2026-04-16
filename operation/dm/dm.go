package dm

import (
	"context"
	"fmt"
	"strings"

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
	CreateDMToolName      = "mattermost_create_dm"
	CreateGroupDMToolName = "mattermost_create_group_dm"
)

var (
	CreateDMTool = mcp.NewTool(
		CreateDMToolName,
		mcp.WithDescription("Create direct message channel with user"),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User ID to DM with")),
	)

	CreateGroupDMTool = mcp.NewTool(
		CreateGroupDMToolName,
		mcp.WithDescription("Create group DM channel with multiple users"),
		mcp.WithString("user_ids", mcp.Required(), mcp.Description("Comma-separated list of user IDs to include in group DM")),
	)
)

func init() {
	registerTools()
}

func registerTools() {
	tools := []server.ServerTool{
		{Tool: CreateDMTool, Handler: CreateDMFn},
		{Tool: CreateGroupDMTool, Handler: CreateGroupDMFn},
	}
	for _, t := range tools {
		Tool.RegisterWrite(t)
	}
}

func CreateDMFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[DM] Called CreateDMFn")

	args := req.GetArguments()

	userId, err := params.GetString(args, "user_id")
	if err != nil {
		return to.Error(fmt.Errorf("[user_id] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	// Get current user to get our own user ID
	me, err := client.GetMe(ctx)
	if err != nil {
		return to.Error(fmt.Errorf("[me] failed to get current user: %v", err)), nil
	}

	// Create direct message channel between current user and target user
	channel, err := client.CreateDirectChannel(ctx, me.Id, userId)
	if err != nil {
		return to.Error(fmt.Errorf("[channel] failed to create DM: %v", err)), nil
	}

	return to.Result(SlimChannel(channel)), nil
}

func CreateGroupDMFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[DM] Called CreateGroupDMFn")

	args := req.GetArguments()

	userIDsStr, err := params.GetString(args, "user_ids")
	if err != nil {
		return to.Error(fmt.Errorf("[user_ids] %v", err)), nil
	}

	// Parse comma-separated user IDs
	userIDs := strings.Split(userIDsStr, ",")
	if len(userIDs) < 2 {
		return to.Error(fmt.Errorf("[user_ids] at least 2 user IDs required for group DM (got %d)", len(userIDs))), nil
	}

	// Trim whitespace from each ID
	for i, id := range userIDs {
		userIDs[i] = strings.TrimSpace(id)
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	// Create group DM channel
	channel, err := client.CreateGroupChannel(ctx, userIDs)
	if err != nil {
		return to.Error(fmt.Errorf("[channel] failed to create group DM: %v", err)), nil
	}

	return to.Result(SlimChannel(channel)), nil
}
