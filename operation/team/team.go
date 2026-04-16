package team

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
	ListTeamsToolName          = "mattermost_list_teams"
	ListTeamMembersToolName    = "mattermost_list_team_members"
	InviteUserToTeamToolName   = "mattermost_invite_user_to_team"
	RemoveUserFromTeamToolName = "mattermost_remove_user_from_team"
	GetTeamStatsToolName       = "mattermost_get_team_stats"
)

var (
	ListTeamsTool = mcp.NewTool(
		ListTeamsToolName,
		mcp.WithDescription("List all teams the bot has access to"),
	)

	ListTeamMembersTool = mcp.NewTool(
		ListTeamMembersToolName,
		mcp.WithDescription("List all members of a team"),
		mcp.WithString("team_id", mcp.Required(), mcp.Description("Team ID to list members for")),
		mcp.WithNumber("page", mcp.Description("Page number (default 0)")),
		mcp.WithNumber("per_page", mcp.Description("Members per page (default 60, max 200)")),
	)

	InviteUserToTeamTool = mcp.NewTool(
		InviteUserToTeamToolName,
		mcp.WithDescription("Invite/add a user to a team"),
		mcp.WithString("team_id", mcp.Required(), mcp.Description("Team ID to invite user to")),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User ID to invite")),
	)

	RemoveUserFromTeamTool = mcp.NewTool(
		RemoveUserFromTeamToolName,
		mcp.WithDescription("Remove a user from a team"),
		mcp.WithString("team_id", mcp.Required(), mcp.Description("Team ID to remove user from")),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User ID to remove")),
	)

	GetTeamStatsTool = mcp.NewTool(
		GetTeamStatsToolName,
		mcp.WithDescription("Get statistics for a team (member count, etc.)"),
		mcp.WithString("team_id", mcp.Required(), mcp.Description("Team ID to get stats for")),
	)
)

func init() {
	registerTools()
}

func registerTools() {
	tools := []server.ServerTool{
		{Tool: ListTeamsTool, Handler: ListTeamsFn},
		{Tool: ListTeamMembersTool, Handler: ListTeamMembersFn},
		{Tool: GetTeamStatsTool, Handler: GetTeamStatsFn},
		{Tool: InviteUserToTeamTool, Handler: InviteUserToTeamFn},
		{Tool: RemoveUserFromTeamTool, Handler: RemoveUserFromTeamFn},
	}
	for _, t := range tools {
		if t.Tool.Name == InviteUserToTeamToolName || t.Tool.Name == RemoveUserFromTeamToolName {
			Tool.RegisterWrite(t)
		} else {
			Tool.RegisterRead(t)
		}
	}
}

func ListTeamsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Team] Called ListTeamsFn")

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	user, err := client.GetMe(ctx)
	if err != nil {
		return to.Error(fmt.Errorf("[user] failed to get current user: %v", err)), nil
	}

	teams, err := client.GetTeamsForUser(ctx, user.Id)
	if err != nil {
		return to.Error(fmt.Errorf("[teams] failed to list teams: %v", err)), nil
	}

	results := make([]map[string]interface{}, 0, len(teams))
	for _, t := range teams {
		results = append(results, SlimTeam(t))
	}

	return to.Result(map[string]interface{}{
		"teams": results,
		"count": len(results),
	}), nil
}

func ListTeamMembersFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Team] Called ListTeamMembersFn")

	args := req.GetArguments()

	teamID, err := params.GetString(args, "team_id")
	if err != nil {
		return to.Error(fmt.Errorf("[team_id] %v", err)), nil
	}

	page := params.GetOptionalInt(args, "page", 0)
	perPage := params.GetOptionalInt(args, "per_page", 60)
	if perPage > 200 {
		perPage = 200
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	members, err := client.ListTeamMembers(ctx, teamID, int(page), int(perPage))
	if err != nil {
		return to.Error(fmt.Errorf("[team] failed to list members: %v", err)), nil
	}

	results := make([]map[string]interface{}, 0, len(members))
	for _, m := range members {
		results = append(results, map[string]interface{}{
			"user_id": m.UserId,
			"roles":   m.Roles,
		})
	}

	return to.Result(map[string]interface{}{
		"members":  results,
		"count":    len(results),
		"team_id":  teamID,
		"page":     page,
		"per_page": perPage,
	}), nil
}

func InviteUserToTeamFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Team] Called InviteUserToTeamFn")

	args := req.GetArguments()

	teamID, err := params.GetString(args, "team_id")
	if err != nil {
		return to.Error(fmt.Errorf("[team_id] %v", err)), nil
	}

	userID, err := params.GetString(args, "user_id")
	if err != nil {
		return to.Error(fmt.Errorf("[user_id] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	_, err = client.InviteUserToTeam(ctx, teamID, userID)
	if err != nil {
		return to.Error(fmt.Errorf("[team] failed to invite user: %v", err)), nil
	}

	return to.Result(map[string]interface{}{
		"success": true,
		"team_id": teamID,
		"user_id": userID,
		"message": "User invited to team successfully",
	}), nil
}

func RemoveUserFromTeamFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Team] Called RemoveUserFromTeamFn")

	args := req.GetArguments()

	teamID, err := params.GetString(args, "team_id")
	if err != nil {
		return to.Error(fmt.Errorf("[team_id] %v", err)), nil
	}

	userID, err := params.GetString(args, "user_id")
	if err != nil {
		return to.Error(fmt.Errorf("[user_id] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	err = client.RemoveUserFromTeam(ctx, teamID, userID)
	if err != nil {
		return to.Error(fmt.Errorf("[team] failed to remove user: %v", err)), nil
	}

	return to.Result(map[string]interface{}{
		"success": true,
		"team_id": teamID,
		"user_id": userID,
		"message": "User removed from team successfully",
	}), nil
}

func GetTeamStatsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Team] Called GetTeamStatsFn")

	args := req.GetArguments()

	teamID, err := params.GetString(args, "team_id")
	if err != nil {
		return to.Error(fmt.Errorf("[team_id] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	stats, err := client.GetTeamStats(ctx, teamID)
	if err != nil {
		return to.Error(fmt.Errorf("[team] failed to get stats: %v", err)), nil
	}

	return to.Result(map[string]interface{}{
		"team_id":        teamID,
		"total_members":  stats.TotalMemberCount,
		"active_members": stats.ActiveMemberCount,
	}), nil
}
