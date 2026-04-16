package messaging

import (
	"context"
	"fmt"

	"github.com/karti-ai/mattermost-mcp-server/pkg/log"
	"github.com/karti-ai/mattermost-mcp-server/pkg/mattermost"
	"github.com/karti-ai/mattermost-mcp-server/pkg/params"
	"github.com/karti-ai/mattermost-mcp-server/pkg/to"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	GetChannelMessagesToolName = "mattermost_get_channel_messages"
)

var (
	GetChannelMessagesTool = mcp.NewTool(
		GetChannelMessagesToolName,
		mcp.WithDescription("Read message history from a channel"),
		mcp.WithString("channel_id", mcp.Required(), mcp.Description("Channel ID to read from")),
		mcp.WithNumber("limit", mcp.Description("Number of messages to return (default 60, max 200)")),
		mcp.WithString("before", mcp.Description("Get messages before this post ID (for pagination)")),
		mcp.WithString("after", mcp.Description("Get messages after this post ID (for pagination)")),
	)
)

func init() {
	registerGetChannelMessagesTool()
}

func registerGetChannelMessagesTool() {
	tools := []server.ServerTool{
		{Tool: GetChannelMessagesTool, Handler: GetChannelMessagesFn},
	}
	for _, t := range tools {
		Tool.RegisterRead(t)
	}
}

func GetChannelMessagesFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Messaging] Called GetChannelMessagesFn")

	args := req.GetArguments()

	channelID, err := params.GetString(args, "channel_id")
	if err != nil {
		return to.Error(fmt.Errorf("[channel_id] %v", err)), nil
	}

	limit := params.GetOptionalInt(args, "limit", 60)
	if limit > 200 {
		limit = 200
	}
	if limit < 1 {
		limit = 60
	}

	before := params.GetOptionalString(args, "before", "")
	after := params.GetOptionalString(args, "after", "")

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	posts, err := client.GetChannelPosts(ctx, channelID, int(limit), before, after)
	if err != nil {
		return to.Error(fmt.Errorf("[posts] failed to get channel messages: %v", err)), nil
	}

	results := make([]map[string]interface{}, 0, len(posts.Posts))
	for _, post := range posts.Posts {
		results = append(results, SlimPost(post))
	}

	return to.Result(map[string]interface{}{
		"posts":      results,
		"count":      len(results),
		"channel_id": channelID,
		"has_more":   len(results) == int(limit),
	}), nil
}
