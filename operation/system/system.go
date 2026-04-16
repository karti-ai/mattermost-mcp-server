package system

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
	GetSystemLogsToolName   = "mattermost_get_system_logs"
	GetServerConfigToolName = "mattermost_get_server_config"
)

var (
	GetSystemLogsTool = mcp.NewTool(
		GetSystemLogsToolName,
		mcp.WithDescription("Get system logs (requires admin privileges)"),
		mcp.WithNumber("page", mcp.Description("Page number (default 0)")),
		mcp.WithNumber("per_page", mcp.Description("Log lines per page (default 100, max 500)")),
	)

	GetServerConfigTool = mcp.NewTool(
		GetServerConfigToolName,
		mcp.WithDescription("Get server configuration (requires admin privileges)"),
	)
)

func init() {
	registerTools()
}

func registerTools() {
	tools := []server.ServerTool{
		{Tool: GetSystemLogsTool, Handler: GetSystemLogsFn},
		{Tool: GetServerConfigTool, Handler: GetServerConfigFn},
	}
	for _, t := range tools {
		Tool.RegisterRead(t)
	}
}

func GetSystemLogsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[System] Called GetSystemLogsFn")

	args := req.GetArguments()

	page := params.GetOptionalInt(args, "page", 0)
	perPage := params.GetOptionalInt(args, "per_page", 100)
	if perPage > 500 {
		perPage = 500
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	logs, _, err := client.GetSystemLogs(ctx, int(page), int(perPage))
	if err != nil {
		return to.Error(fmt.Errorf("[system] failed to get logs: %v", err)), nil
	}

	return to.Result(map[string]interface{}{
		"logs":     logs,
		"count":    len(logs),
		"page":     page,
		"per_page": perPage,
	}), nil
}

func GetServerConfigFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[System] Called GetServerConfigFn")

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	config, err := client.GetConfig(ctx)
	if err != nil {
		return to.Error(fmt.Errorf("[system] failed to get config: %v", err)), nil
	}

	return to.Result(map[string]interface{}{
		"site_name":                     config.TeamSettings.SiteName,
		"max_notifications_per_channel": config.TeamSettings.MaxNotificationsPerChannel,
		"enable_custom_emoji":           config.ServiceSettings.EnableCustomEmoji,
		"enable_link_previews":          config.ServiceSettings.EnableLinkPreviews,
		"enable_public_channels":        config.TeamSettings.EnableOpenServer,
	}), nil
}
