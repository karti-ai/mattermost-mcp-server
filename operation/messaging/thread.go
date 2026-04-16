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
	GetThreadToolName = "mattermost_get_thread"
)

var (
	GetThreadTool = mcp.NewTool(
		GetThreadToolName,
		mcp.WithDescription("Read all messages in a thread conversation"),
		mcp.WithString("post_id", mcp.Required(), mcp.Description("The root post ID of the thread")),
	)
)

func init() {
	registerGetThreadTool()
}

func registerGetThreadTool() {
	tools := []server.ServerTool{
		{Tool: GetThreadTool, Handler: GetThreadFn},
	}
	for _, t := range tools {
		Tool.RegisterRead(t)
	}
}

func GetThreadFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Messaging] Called GetThreadFn")

	args := req.GetArguments()

	postID, err := params.GetString(args, "post_id")
	if err != nil {
		return to.Error(fmt.Errorf("[post_id] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	thread, err := client.GetPostThread(ctx, postID)
	if err != nil {
		return to.Error(fmt.Errorf("[thread] failed to get thread: %v", err)), nil
	}

	results := make([]map[string]interface{}, 0, len(thread.Posts))
	for _, post := range thread.Posts {
		results = append(results, SlimPost(post))
	}

	return to.Result(map[string]interface{}{
		"posts":       results,
		"count":       len(results),
		"root_id":     postID,
		"has_replies": len(results) > 1,
	}), nil
}
