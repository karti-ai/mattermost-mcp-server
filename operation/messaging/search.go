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
	SearchPostsToolName = "mattermost_search_posts"
)

var (
	SearchPostsTool = mcp.NewTool(
		SearchPostsToolName,
		mcp.WithDescription("Search for posts/messages in a team using search terms"),
		mcp.WithString("team_id", mcp.Required(), mcp.Description("Team ID to search in")),
		mcp.WithString("terms", mcp.Required(), mcp.Description("Search terms (e.g., \"BTCUSD\", \"error\", \"meeting\")")),
		mcp.WithBoolean("is_or_search", mcp.Description("Use OR logic instead of AND (default: false)")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results to return (default 30)")),
	)
)

func init() {
	registerSearchTools()
}

func registerSearchTools() {
	tools := []server.ServerTool{
		{Tool: SearchPostsTool, Handler: SearchPostsFn},
	}
	for _, t := range tools {
		Tool.RegisterRead(t)
	}
}

func SearchPostsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Messaging] Called SearchPostsFn")

	args := req.GetArguments()

	teamID, err := params.GetString(args, "team_id")
	if err != nil {
		return to.Error(fmt.Errorf("[team_id] %v", err)), nil
	}

	terms, err := params.GetString(args, "terms")
	if err != nil {
		return to.Error(fmt.Errorf("[terms] %v", err)), nil
	}

	isOrSearch := params.GetOptionalBool(args, "is_or_search", false)
	limit := params.GetOptionalInt(args, "limit", 30)

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	results, err := client.SearchPosts(ctx, teamID, terms, isOrSearch)
	if err != nil {
		return to.Error(fmt.Errorf("[search] failed to search posts: %v", err)), nil
	}

	posts := results.ToSlice()
	if len(posts) > int(limit) {
		posts = posts[:limit]
	}

	postResults := make([]map[string]interface{}, 0, len(posts))
	for _, post := range posts {
		postResults = append(postResults, SlimPost(post))
	}

	return to.Result(map[string]interface{}{
		"posts": postResults,
		"count": len(postResults),
	}), nil
}
