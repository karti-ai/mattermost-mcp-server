package channel

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
	ListChannelsToolName       = "mattermost_list_channels"
	GetChannelByNameToolName   = "mattermost_get_channel_by_name"
	GetChannelInfoToolName     = "mattermost_get_channel_info"
	ListChannelMembersToolName = "mattermost_list_channel_members"
)

var (
	ListChannelsTool = mcp.NewTool(
		ListChannelsToolName,
		mcp.WithDescription("List accessible channels for the authenticated user in a team"),
		mcp.WithString("team_id", mcp.Required(), mcp.Description("Team ID to list channels from")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results to return (default 30)")),
	)

	GetChannelByNameTool = mcp.NewTool(
		GetChannelByNameToolName,
		mcp.WithDescription("Get a channel by name in a team"),
		mcp.WithString("team_id", mcp.Required(), mcp.Description("Team ID to search in")),
		mcp.WithString("channel_name", mcp.Required(), mcp.Description("Channel name to find (e.g., \"general\", \"social\", \"trading-desk\")")),
	)

	GetChannelInfoTool = mcp.NewTool(
		GetChannelInfoToolName,
		mcp.WithDescription("Get detailed channel information including member count, purpose, etc."),
		mcp.WithString("channel_id", mcp.Required(), mcp.Description("Channel ID to get info for")),
	)

	ListChannelMembersTool = mcp.NewTool(
		ListChannelMembersToolName,
		mcp.WithDescription("List all members of a channel"),
		mcp.WithString("channel_id", mcp.Required(), mcp.Description("Channel ID to list members for")),
		mcp.WithNumber("page", mcp.Description("Page number for pagination (default 0)")),
		mcp.WithNumber("per_page", mcp.Description("Members per page (default 60, max 200)")),
	)
)

func init() {
	registerTools()
}

func registerTools() {
	tools := []server.ServerTool{
		{Tool: ListChannelsTool, Handler: ListChannelsFn},
		{Tool: GetChannelByNameTool, Handler: GetChannelByNameFn},
		{Tool: GetChannelInfoTool, Handler: GetChannelInfoFn},
		{Tool: ListChannelMembersTool, Handler: ListChannelMembersFn},
	}
	for _, t := range tools {
		Tool.RegisterRead(t)
	}
}

func ListChannelsFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Channel] Called ListChannelsFn")

	args := req.GetArguments()

	teamID, err := params.GetString(args, "team_id")
	if err != nil {
		return to.Error(fmt.Errorf("[team_id] %v", err)), nil
	}

	limit := params.GetOptionalInt(args, "limit", 30)

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	user, err := client.GetMe(ctx)
	if err != nil {
		return to.Error(fmt.Errorf("[user] failed to get current user: %v", err)), nil
	}

	channels, err := client.GetChannelsForTeamForUser(ctx, teamID, user.Id, false)
	if err != nil {
		return to.Error(fmt.Errorf("[channels] failed to list channels: %v", err)), nil
	}

	if len(channels) > int(limit) {
		channels = channels[:limit]
	}

	results := make([]map[string]interface{}, 0, len(channels))
	for _, ch := range channels {
		results = append(results, SlimChannel(ch))
	}

	return to.Result(map[string]interface{}{
		"channels": results,
		"count":    len(results),
	}), nil
}

func GetChannelByNameFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Channel] Called GetChannelByNameFn")

	args := req.GetArguments()

	teamID, err := params.GetString(args, "team_id")
	if err != nil {
		return to.Error(fmt.Errorf("[team_id] %v", err)), nil
	}

	channelName, err := params.GetString(args, "channel_name")
	if err != nil {
		return to.Error(fmt.Errorf("[channel_name] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	channel, err := client.GetChannelByName(ctx, teamID, channelName)
	if err != nil {
		return to.Error(fmt.Errorf("[channel] failed to get channel: %v", err)), nil
	}

	return to.Result(SlimChannel(channel)), nil
}

func GetChannelInfoFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Channel] Called GetChannelInfoFn")

	args := req.GetArguments()

	channelID, err := params.GetString(args, "channel_id")
	if err != nil {
		return to.Error(fmt.Errorf("[channel_id] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	channel, err := client.GetChannel(ctx, channelID)
	if err != nil {
		return to.Error(fmt.Errorf("[channel] failed to get channel: %v", err)), nil
	}

	stats, err := client.GetChannelStats(ctx, channelID)
	if err != nil {
		return to.Error(fmt.Errorf("[channel] failed to get channel stats: %v", err)), nil
	}

	return to.Result(DetailedChannel(channel, stats.MemberCount)), nil
}

func ListChannelMembersFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Channel] Called ListChannelMembersFn")

	args := req.GetArguments()

	channelID, err := params.GetString(args, "channel_id")
	if err != nil {
		return to.Error(fmt.Errorf("[channel_id] %v", err)), nil
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

	members, err := client.GetChannelMembers(ctx, channelID, int(page), int(perPage))
	if err != nil {
		return to.Error(fmt.Errorf("[channel] failed to list members: %v", err)), nil
	}

	results := make([]map[string]interface{}, 0, len(members))
	for _, m := range members {
		results = append(results, SlimChannelMember(m))
	}

	return to.Result(map[string]interface{}{
		"members":    results,
		"count":      len(results),
		"channel_id": channelID,
		"page":       page,
		"per_page":   perPage,
	}), nil
}
