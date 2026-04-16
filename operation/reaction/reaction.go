package reaction

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
	"github.com/mattermost/mattermost-server/v6/model"
)

var Tool = tool.New()

const (
	AddReactionToolName    = "mattermost_add_reaction"
	RemoveReactionToolName = "mattermost_remove_reaction"
	ListReactionsToolName  = "mattermost_list_reactions"
)

var (
	AddReactionTool = mcp.NewTool(
		AddReactionToolName,
		mcp.WithDescription("Add emoji reaction to post"),
		mcp.WithString("post_id", mcp.Required(), mcp.Description("Post ID to add reaction to")),
		mcp.WithString("emoji_name", mcp.Required(), mcp.Description("Emoji name without colons (e.g., thumbsup, not :thumbsup:)")),
	)

	RemoveReactionTool = mcp.NewTool(
		RemoveReactionToolName,
		mcp.WithDescription("Remove emoji reaction from post"),
		mcp.WithString("post_id", mcp.Required(), mcp.Description("Post ID to remove reaction from")),
		mcp.WithString("emoji_name", mcp.Required(), mcp.Description("Emoji name without colons")),
	)

	ListReactionsTool = mcp.NewTool(
		ListReactionsToolName,
		mcp.WithDescription("List all emoji reactions on a post"),
		mcp.WithString("post_id", mcp.Required(), mcp.Description("Post ID to get reactions for")),
	)
)

func init() {
	registerTools()
}

func registerTools() {
	tools := []server.ServerTool{
		{Tool: AddReactionTool, Handler: AddReactionFn},
		{Tool: RemoveReactionTool, Handler: RemoveReactionFn},
		{Tool: ListReactionsTool, Handler: ListReactionsFn},
	}
	for _, t := range tools {
		if t.Tool.Name == ListReactionsToolName {
			Tool.RegisterRead(t)
		} else {
			Tool.RegisterWrite(t)
		}
	}
}

func AddReactionFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Reaction] Called AddReactionFn")

	args := req.GetArguments()

	postId, err := params.GetString(args, "post_id")
	if err != nil {
		return to.Error(fmt.Errorf("[post_id] %v", err)), nil
	}

	emojiName, err := params.GetString(args, "emoji_name")
	if err != nil {
		return to.Error(fmt.Errorf("[emoji_name] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	// Get current user ID
	me, err := client.GetMe(ctx)
	if err != nil {
		return to.Error(fmt.Errorf("[user] failed to get current user: %v", err)), nil
	}

	reaction := &model.Reaction{
		UserId:    me.Id,
		PostId:    postId,
		EmojiName: emojiName,
	}

	result, err := client.SaveReaction(ctx, reaction)
	if err != nil {
		return to.Error(fmt.Errorf("[reaction] failed to add reaction: %v", err)), nil
	}

	return to.Result(SlimReaction(result)), nil
}

func RemoveReactionFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Reaction] Called RemoveReactionFn")

	args := req.GetArguments()

	postId, err := params.GetString(args, "post_id")
	if err != nil {
		return to.Error(fmt.Errorf("[post_id] %v", err)), nil
	}

	emojiName, err := params.GetString(args, "emoji_name")
	if err != nil {
		return to.Error(fmt.Errorf("[emoji_name] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	// Get current user ID
	me, err := client.GetMe(ctx)
	if err != nil {
		return to.Error(fmt.Errorf("[user] failed to get current user: %v", err)), nil
	}

	reaction := &model.Reaction{
		UserId:    me.Id,
		PostId:    postId,
		EmojiName: emojiName,
	}

	err = client.DeleteReaction(ctx, reaction)
	if err != nil {
		return to.Error(fmt.Errorf("[reaction] failed to remove reaction: %v", err)), nil
	}

	return to.Result(map[string]interface{}{
		"success":    true,
		"post_id":    postId,
		"emoji_name": emojiName,
		"message":    "Reaction removed successfully",
	}), nil
}

func ListReactionsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Reaction] Called ListReactionsFn")

	args := req.GetArguments()

	postId, err := params.GetString(args, "post_id")
	if err != nil {
		return to.Error(fmt.Errorf("[post_id] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	reactions, err := client.GetReactions(ctx, postId)
	if err != nil {
		return to.Error(fmt.Errorf("[reaction] failed to list reactions: %v", err)), nil
	}

	results := make([]map[string]interface{}, 0, len(reactions))
	for _, r := range reactions {
		results = append(results, SlimReaction(r))
	}

	return to.Result(map[string]interface{}{
		"reactions": results,
		"count":     len(results),
		"post_id":   postId,
	}), nil
}
