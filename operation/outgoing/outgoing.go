package outgoing

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
	CreateOutgoingWebhookToolName = "mattermost_create_outgoing_webhook"
	ListOutgoingWebhooksToolName  = "mattermost_list_outgoing_webhooks"
	DeleteOutgoingWebhookToolName = "mattermost_delete_outgoing_webhook"
)

var (
	CreateOutgoingWebhookTool = mcp.NewTool(
		CreateOutgoingWebhookToolName,
		mcp.WithDescription("Create an outgoing webhook that triggers on specific words"),
		mcp.WithString("team_id", mcp.Required(), mcp.Description("Team ID for the webhook")),
		mcp.WithString("display_name", mcp.Required(), mcp.Description("Display name for the webhook")),
		mcp.WithString("trigger_words", mcp.Required(), mcp.Description("Comma-separated list of words that trigger the webhook")),
		mcp.WithString("callback_url", mcp.Required(), mcp.Description("URL to POST to when triggered")),
	)

	ListOutgoingWebhooksTool = mcp.NewTool(
		ListOutgoingWebhooksToolName,
		mcp.WithDescription("List outgoing webhooks for a team"),
		mcp.WithString("team_id", mcp.Required(), mcp.Description("Team ID to list webhooks for")),
		mcp.WithNumber("page", mcp.Description("Page number (default 0)")),
		mcp.WithNumber("per_page", mcp.Description("Items per page (default 20, max 100)")),
	)

	DeleteOutgoingWebhookTool = mcp.NewTool(
		DeleteOutgoingWebhookToolName,
		mcp.WithDescription("Delete an outgoing webhook"),
		mcp.WithString("webhook_id", mcp.Required(), mcp.Description("Webhook ID to delete")),
	)
)

func init() {
	registerTools()
}

func registerTools() {
	tools := []server.ServerTool{
		{Tool: CreateOutgoingWebhookTool, Handler: CreateOutgoingWebhookFn},
		{Tool: ListOutgoingWebhooksTool, Handler: ListOutgoingWebhooksFn},
		{Tool: DeleteOutgoingWebhookTool, Handler: DeleteOutgoingWebhookFn},
	}
	for _, t := range tools {
		if t.Tool.Name == ListOutgoingWebhooksToolName {
			Tool.RegisterRead(t)
		} else {
			Tool.RegisterWrite(t)
		}
	}
}

func CreateOutgoingWebhookFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Outgoing] Called CreateOutgoingWebhookFn")

	args := req.GetArguments()

	teamID, err := params.GetString(args, "team_id")
	if err != nil {
		return to.Error(fmt.Errorf("[team_id] %v", err)), nil
	}

	displayName, err := params.GetString(args, "display_name")
	if err != nil {
		return to.Error(fmt.Errorf("[display_name] %v", err)), nil
	}

	triggerWordsStr, err := params.GetString(args, "trigger_words")
	if err != nil {
		return to.Error(fmt.Errorf("[trigger_words] %v", err)), nil
	}
	triggerWords := strings.Split(triggerWordsStr, ",")
	for i, word := range triggerWords {
		triggerWords[i] = strings.TrimSpace(word)
	}

	callbackURL, err := params.GetString(args, "callback_url")
	if err != nil {
		return to.Error(fmt.Errorf("[callback_url] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	hook, err := client.CreateOutgoingWebhook(ctx, teamID, displayName, triggerWords, callbackURL)
	if err != nil {
		return to.Error(fmt.Errorf("[webhook] failed to create: %v", err)), nil
	}

	return to.Result(map[string]interface{}{
		"id":            hook.Id,
		"team_id":       hook.TeamId,
		"display_name":  hook.DisplayName,
		"trigger_words": hook.TriggerWords,
		"callback_url":  callbackURL,
		"message":       "Outgoing webhook created successfully",
	}), nil
}

func ListOutgoingWebhooksFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Outgoing] Called ListOutgoingWebhooksFn")

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

	hooks, err := client.ListOutgoingWebhooks(ctx, teamID, int(page), int(perPage))
	if err != nil {
		return to.Error(fmt.Errorf("[webhook] failed to list: %v", err)), nil
	}

	results := make([]map[string]interface{}, 0, len(hooks))
	for _, hook := range hooks {
		results = append(results, map[string]interface{}{
			"id":            hook.Id,
			"team_id":       hook.TeamId,
			"display_name":  hook.DisplayName,
			"trigger_words": hook.TriggerWords,
		})
	}

	return to.Result(map[string]interface{}{
		"webhooks": results,
		"count":    len(results),
		"page":     page,
		"per_page": perPage,
	}), nil
}

func DeleteOutgoingWebhookFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Outgoing] Called DeleteOutgoingWebhookFn")

	args := req.GetArguments()

	webhookID, err := params.GetString(args, "webhook_id")
	if err != nil {
		return to.Error(fmt.Errorf("[webhook_id] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	err = client.DeleteOutgoingWebhook(ctx, webhookID)
	if err != nil {
		return to.Error(fmt.Errorf("[webhook] failed to delete: %v", err)), nil
	}

	return to.Result(map[string]interface{}{
		"success":    true,
		"webhook_id": webhookID,
		"message":    "Outgoing webhook deleted successfully",
	}), nil
}
