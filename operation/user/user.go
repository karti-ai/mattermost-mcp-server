package user

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
	SearchUsersToolName = "mattermost_search_users"
	GetUserToolName     = "mattermost_get_user"
)

var (
	SearchUsersTool = mcp.NewTool(
		SearchUsersToolName,
		mcp.WithDescription("Search users by term"),
		mcp.WithString("term", mcp.Required(), mcp.Description("Search term (username, email, name)")),
		mcp.WithString("team_id", mcp.Description("Limit to team (optional)")),
		mcp.WithNumber("limit", mcp.Description("Max results (default 30)")),
	)

	GetUserTool = mcp.NewTool(
		GetUserToolName,
		mcp.WithDescription("Get a specific user by ID or username"),
		mcp.WithString("user_id", mcp.Description("User ID to look up (optional if username provided)")),
		mcp.WithString("username", mcp.Description("Username to look up (optional if user_id provided)")),
	)
)

func init() {
	registerTools()
}

func registerTools() {
	tools := []server.ServerTool{
		{Tool: SearchUsersTool, Handler: SearchUsersFn},
		{Tool: GetUserTool, Handler: GetUserFn},
	}
	for _, t := range tools {
		Tool.RegisterRead(t)
	}
}

func SearchUsersFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[User] Called SearchUsersFn")

	args := req.GetArguments()

	term, err := params.GetString(args, "term")
	if err != nil {
		return to.Error(fmt.Errorf("[term] %v", err)), nil
	}

	teamID := params.GetOptionalString(args, "team_id", "")
	limit := params.GetOptionalInt(args, "limit", 30)

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	search := &model.UserSearch{
		Term:  term,
		Limit: int(limit),
	}

	if teamID != "" {
		search.TeamId = teamID
	}

	users, err := client.SearchUsers(ctx, search)
	if err != nil {
		return to.Error(fmt.Errorf("[users] failed to search users: %v", err)), nil
	}

	results := make([]map[string]interface{}, 0, len(users))
	for _, u := range users {
		results = append(results, SlimUser(u))
	}

	return to.Result(map[string]interface{}{
		"users": results,
		"count": len(results),
	}), nil
}

func GetUserFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[User] Called GetUserFn")

	args := req.GetArguments()

	userID := params.GetOptionalString(args, "user_id", "")
	username := params.GetOptionalString(args, "username", "")

	if userID == "" && username == "" {
		return to.Error(fmt.Errorf("[user] either user_id or username must be provided")), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	var user *model.User
	var err error

	if userID != "" {
		user, err = client.GetUser(ctx, userID)
	} else {
		user, err = client.GetUserByUsername(ctx, username)
	}

	if err != nil {
		return to.Error(fmt.Errorf("[user] failed to get user: %v", err)), nil
	}

	return to.Result(SlimUser(user)), nil
}
