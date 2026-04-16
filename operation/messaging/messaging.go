package messaging

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
	"github.com/mattermost/mattermost-server/v6/model"
)

var Tool = tool.New()

const (
	SendMessageToolName        = "mattermost_send_message"
	EditMessageToolName        = "mattermost_edit_message"
	DeleteMessageToolName      = "mattermost_delete_message"
	BulkDeleteMessagesToolName = "mattermost_bulk_delete_messages"
	PinPostToolName            = "mattermost_pin_post"
	UnpinPostToolName          = "mattermost_unpin_post"
	GetPinnedPostsToolName     = "mattermost_get_pinned_posts"
	GetPostToolName            = "mattermost_get_post"
)

var (
	SendMessageTool = mcp.NewTool(
		SendMessageToolName,
		mcp.WithDescription("Send a message to a Mattermost channel"),
		mcp.WithString("channel_id", mcp.Required(), mcp.Description("Channel ID to send the message to")),
		mcp.WithString("message", mcp.Required(), mcp.Description("Message content to send")),
		mcp.WithString("thread_id", mcp.Description("Thread ID for replies (optional). If provided, message will be posted as a reply in the thread")),
	)

	EditMessageTool = mcp.NewTool(
		EditMessageToolName,
		mcp.WithDescription("Edit an existing message in Mattermost"),
		mcp.WithString("post_id", mcp.Required(), mcp.Description("Post ID of the message to edit")),
		mcp.WithString("message", mcp.Required(), mcp.Description("New message content")),
	)

	DeleteMessageTool = mcp.NewTool(
		DeleteMessageToolName,
		mcp.WithDescription("Delete a message from Mattermost"),
		mcp.WithString("post_id", mcp.Required(), mcp.Description("Post ID of the message to delete")),
	)

	BulkDeleteMessagesTool = mcp.NewTool(
		BulkDeleteMessagesToolName,
		mcp.WithDescription("Delete multiple messages at once (up to 100)"),
		mcp.WithString("post_ids", mcp.Required(), mcp.Description("Comma-separated list of post IDs to delete (max 100)")),
	)

	PinPostTool = mcp.NewTool(
		PinPostToolName,
		mcp.WithDescription("Pin a post to a channel"),
		mcp.WithString("post_id", mcp.Required(), mcp.Description("Post ID to pin")),
	)

	UnpinPostTool = mcp.NewTool(
		UnpinPostToolName,
		mcp.WithDescription("Unpin a post from a channel"),
		mcp.WithString("post_id", mcp.Required(), mcp.Description("Post ID to unpin")),
	)

	GetPinnedPostsTool = mcp.NewTool(
		GetPinnedPostsToolName,
		mcp.WithDescription("Get all pinned posts in a channel"),
		mcp.WithString("channel_id", mcp.Required(), mcp.Description("Channel ID to get pinned posts from")),
	)

	GetPostTool = mcp.NewTool(
		GetPostToolName,
		mcp.WithDescription("Get a single post by its ID"),
		mcp.WithString("post_id", mcp.Required(), mcp.Description("Post ID to retrieve")),
	)
)

func init() {
	registerTools()
}

func registerTools() {
	tools := []server.ServerTool{
		{Tool: SendMessageTool, Handler: SendMessageFn},
		{Tool: EditMessageTool, Handler: EditMessageFn},
		{Tool: DeleteMessageTool, Handler: DeleteMessageFn},
		{Tool: BulkDeleteMessagesTool, Handler: BulkDeleteMessagesFn},
		{Tool: PinPostTool, Handler: PinPostFn},
		{Tool: UnpinPostTool, Handler: UnpinPostFn},
		{Tool: GetPinnedPostsTool, Handler: GetPinnedPostsFn},
		{Tool: GetPostTool, Handler: GetPostFn},
	}
	for _, t := range tools {
		if t.Tool.Name == GetPinnedPostsToolName || t.Tool.Name == GetPostToolName {
			Tool.RegisterRead(t)
		} else {
			Tool.RegisterWrite(t)
		}
	}
}

func SendMessageFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Messaging] Called SendMessageFn")

	args := req.GetArguments()

	channelId, err := params.GetString(args, "channel_id")
	if err != nil {
		return to.Error(fmt.Errorf("[channel_id] %v", err)), nil
	}

	message, err := params.GetString(args, "message")
	if err != nil {
		return to.Error(fmt.Errorf("[message] %v", err)), nil
	}

	threadId := params.GetOptionalString(args, "thread_id", "")

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	post := &model.Post{
		ChannelId: channelId,
		Message:   message,
	}

	if threadId != "" {
		post.RootId = threadId
	}

	result, err := client.CreatePost(ctx, post)
	if err != nil {
		return to.Error(fmt.Errorf("[post] failed to send message: %v", err)), nil
	}

	return to.Result(SlimPost(result)), nil
}

func EditMessageFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Messaging] Called EditMessageFn")

	args := req.GetArguments()

	postId, err := params.GetString(args, "post_id")
	if err != nil {
		return to.Error(fmt.Errorf("[post_id] %v", err)), nil
	}

	message, err := params.GetString(args, "message")
	if err != nil {
		return to.Error(fmt.Errorf("[message] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	post := &model.Post{
		Message: message,
	}

	result, err := client.UpdatePost(ctx, postId, post)
	if err != nil {
		return to.Error(fmt.Errorf("[post] failed to edit message: %v", err)), nil
	}

	return to.Result(SlimPost(result)), nil
}

func DeleteMessageFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Messaging] Called DeleteMessageFn")

	args := req.GetArguments()

	postId, err := params.GetString(args, "post_id")
	if err != nil {
		return to.Error(fmt.Errorf("[post_id] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	err = client.DeletePost(ctx, postId)
	if err != nil {
		return to.Error(fmt.Errorf("[post] failed to delete message: %v", err)), nil
	}

	return to.Result(map[string]interface{}{
		"success": true,
		"post_id": postId,
		"message": "Message deleted successfully",
	}), nil
}

func BulkDeleteMessagesFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Messaging] Called BulkDeleteMessagesFn")

	args := req.GetArguments()

	postIDsStr, err := params.GetString(args, "post_ids")
	if err != nil {
		return to.Error(fmt.Errorf("[post_ids] %v", err)), nil
	}

	postIDs := strings.Split(postIDsStr, ",")
	if len(postIDs) > 100 {
		return to.Error(fmt.Errorf("[post_ids] too many post IDs (max 100, got %d)", len(postIDs))), nil
	}

	for i, id := range postIDs {
		postIDs[i] = strings.TrimSpace(id)
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	deleted := 0
	failed := 0
	var failedIDs []string
	var lastError string

	for _, postID := range postIDs {
		err := client.DeletePost(ctx, postID)
		if err != nil {
			failed++
			failedIDs = append(failedIDs, postID)
			lastError = err.Error()
		} else {
			deleted++
		}
	}

	result := map[string]interface{}{
		"total":      len(postIDs),
		"deleted":    deleted,
		"failed":     failed,
		"successful": deleted == len(postIDs),
	}

	if failed > 0 {
		result["failed_ids"] = failedIDs
		result["last_error"] = lastError
		return to.Result(result), nil
	}

	return to.Result(result), nil
}

func PinPostFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Messaging] Called PinPostFn")

	args := req.GetArguments()

	postId, err := params.GetString(args, "post_id")
	if err != nil {
		return to.Error(fmt.Errorf("[post_id] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	err = client.PinPost(ctx, postId)
	if err != nil {
		return to.Error(fmt.Errorf("[post] failed to pin: %v", err)), nil
	}

	return to.Result(map[string]interface{}{
		"success": true,
		"post_id": postId,
		"message": "Post pinned successfully",
	}), nil
}

func UnpinPostFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Messaging] Called UnpinPostFn")

	args := req.GetArguments()

	postId, err := params.GetString(args, "post_id")
	if err != nil {
		return to.Error(fmt.Errorf("[post_id] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	err = client.UnpinPost(ctx, postId)
	if err != nil {
		return to.Error(fmt.Errorf("[post] failed to unpin: %v", err)), nil
	}

	return to.Result(map[string]interface{}{
		"success": true,
		"post_id": postId,
		"message": "Post unpinned successfully",
	}), nil
}

func GetPinnedPostsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Messaging] Called GetPinnedPostsFn")

	args := req.GetArguments()

	channelId, err := params.GetString(args, "channel_id")
	if err != nil {
		return to.Error(fmt.Errorf("[channel_id] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	posts, err := client.GetPinnedPosts(ctx, channelId)
	if err != nil {
		return to.Error(fmt.Errorf("[channel] failed to get pinned posts: %v", err)), nil
	}

	results := make([]map[string]interface{}, 0, len(posts.Posts))
	for _, post := range posts.Posts {
		results = append(results, SlimPost(post))
	}

	return to.Result(map[string]interface{}{
		"posts":      results,
		"count":      len(results),
		"channel_id": channelId,
	}), nil
}

func GetPostFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Messaging] Called GetPostFn")

	args := req.GetArguments()

	postId, err := params.GetString(args, "post_id")
	if err != nil {
		return to.Error(fmt.Errorf("[post_id] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	post, err := client.GetPost(ctx, postId)
	if err != nil {
		return to.Error(fmt.Errorf("[post] failed to get: %v", err)), nil
	}

	return to.Result(SlimPost(post)), nil
}
