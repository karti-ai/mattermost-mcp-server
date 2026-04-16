package webhook

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
	CreateIncomingWebhookToolName = "mattermost_create_incoming_webhook"
	ListIncomingWebhooksToolName  = "mattermost_list_incoming_webhooks"
	DeleteIncomingWebhookToolName = "mattermost_delete_incoming_webhook"
)

var (
	CreateIncomingWebhookTool = mcp.NewTool(
		CreateIncomingWebhookToolName,
		mcp.WithDescription("Create an incoming webhook for a channel"),
		mcp.WithString("channel_id", mcp.Required(), mcp.Description("Channel ID to create webhook for")),
		mcp.WithString("display_name", mcp.Required(), mcp.Description("Display name for the webhook")),
	)

	ListIncomingWebhooksTool = mcp.NewTool(
		ListIncomingWebhooksToolName,
		mcp.WithDescription("List incoming webhooks for a team"),
		mcp.WithString("team_id", mcp.Required(), mcp.Description("Team ID to list webhooks for")),
		mcp.WithNumber("page", mcp.Description("Page number (default 0)")),
		mcp.WithNumber("per_page", mcp.Description("Items per page (default 20, max 100)")),
	)

	DeleteIncomingWebhookTool = mcp.NewTool(
		DeleteIncomingWebhookToolName,
		mcp.WithDescription("Delete an incoming webhook"),
		mcp.WithString("webhook_id", mcp.Required(), mcp.Description("Webhook ID to delete")),
	)
)

func init() {
	registerTools()
}

func registerTools() {
	tools := []server.ServerTool{
		{Tool: CreateIncomingWebhookTool, Handler: CreateIncomingWebhookFn},
		{Tool: ListIncomingWebhooksTool, Handler: ListIncomingWebhooksFn},
		{Tool: DeleteIncomingWebhookTool, Handler: DeleteIncomingWebhookFn},
	}
	for _, t := range tools {
		if t.Tool.Name == ListIncomingWebhooksToolName {
			Tool.RegisterRead(t)
		} else {
			Tool.RegisterWrite(t)
		}
	}
}

func CreateIncomingWebhookFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Webhook] Called CreateIncomingWebhookFn")

	args := req.GetArguments()

	channelID, err := params.GetString(args, "channel_id")
	if err != nil {
		return to.Error(fmt.Errorf("[channel_id] %v", err)), nil
	}

	displayName, err := params.GetString(args, "display_name")
	if err != nil {
		return to.Error(fmt.Errorf("[display_name] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	hook, err := client.CreateIncomingWebhook(ctx, channelID, displayName)
	if err != nil {
		return to.Error(fmt.Errorf("[webhook] failed to create: %v", err)), nil
	}

	return to.Result(map[string]interface{}{
		"id":           hook.Id,
		"channel_id":   hook.ChannelId,
		"display_name": hook.DisplayName,
		"message":      "Webhook created successfully - retrieve URL from Mattermost UI",
	}), nil
}

func ListIncomingWebhooksFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Webhook] Called ListIncomingWebhooksFn")

	args := req.GetArguments()

	teamID, err := params.GetString(args, "team_id")
	if err != nil {
		return to.Error(fmt.Errorf("[team_id] %v", err)), nil
	}

	page := params.GetOptionalInt(args, "page", 0)
	perPage := params.GetOptionalInt(args, "per_page", 20)
	if perPage > 100 {
		perPage = 100
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	hooks, err := client.ListIncomingWebhooks(ctx, teamID, int(page), int(perPage))
	if err != nil {
		return to.Error(fmt.Errorf("[webhook] failed to list: %v", err)), nil
	}

	results := make([]map[string]interface{}, 0, len(hooks))
	for _, hook := range hooks {
		results = append(results, map[string]interface{}{
			"id":           hook.Id,
			"channel_id":   hook.ChannelId,
			"display_name": hook.DisplayName,
			"create_at":    hook.CreateAt,
		})
	}

	return to.Result(map[string]interface{}{
		"webhooks": results,
		"count":    len(results),
		"page":     page,
		"per_page": perPage,
	}), nil
}

func DeleteIncomingWebhookFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Webhook] Called DeleteIncomingWebhookFn")

	args := req.GetArguments()

	webhookID, err := params.GetString(args, "webhook_id")
	if err != nil {
		return to.Error(fmt.Errorf("[webhook_id] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	err = client.DeleteIncomingWebhook(ctx, webhookID)
	if err != nil {
		return to.Error(fmt.Errorf("[webhook] failed to delete: %v", err)), nil
	}

	return to.Result(map[string]interface{}{
		"success":    true,
		"webhook_id": webhookID,
		"message":    "Webhook deleted successfully",
	}), nil
}
