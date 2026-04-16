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
	"github.com/mattermost/mattermost-server/v6/model"
)

var AdminTool = tool.New()

const (
	CreateChannelToolName   = "mattermost_create_channel"
	InviteToChannelToolName = "mattermost_invite_to_channel"
	DeleteChannelToolName   = "mattermost_delete_channel"
	LeaveChannelToolName    = "mattermost_leave_channel"
)

var (
	CreateChannelTool = mcp.NewTool(
		CreateChannelToolName,
		mcp.WithDescription("Create a new channel (public or private) in a team"),
		mcp.WithString("team_id", mcp.Required(), mcp.Description("Team to create channel in")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Channel name (lowercase, no spaces, 2-64 characters)")),
		mcp.WithString("display_name", mcp.Required(), mcp.Description("Display name for the channel (2-64 characters)")),
		mcp.WithString("type", mcp.Required(), mcp.Description("Channel type: 'O' for public, 'P' for private")),
		mcp.WithString("purpose", mcp.Description("Channel description/purpose (optional)")),
	)

	InviteToChannelTool = mcp.NewTool(
		InviteToChannelToolName,
		mcp.WithDescription("Invite a user to a channel"),
		mcp.WithString("channel_id", mcp.Required(), mcp.Description("Channel to invite user to")),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User to invite")),
	)

	DeleteChannelTool = mcp.NewTool(
		DeleteChannelToolName,
		mcp.WithDescription("Delete/archive a channel (soft delete by default)"),
		mcp.WithString("channel_id", mcp.Required(), mcp.Description("Channel to delete")),
		mcp.WithBoolean("permanent", mcp.Description("Permanently delete instead of archive (default: false)")),
	)

	LeaveChannelTool = mcp.NewTool(
		LeaveChannelToolName,
		mcp.WithDescription("Remove self from a channel"),
		mcp.WithString("channel_id", mcp.Required(), mcp.Description("Channel to leave")),
	)
)

func init() {
	registerAdminTools()
}

func registerAdminTools() {
	tools := []server.ServerTool{
		{Tool: CreateChannelTool, Handler: CreateChannelFn},
		{Tool: InviteToChannelTool, Handler: InviteToChannelFn},
		{Tool: DeleteChannelTool, Handler: DeleteChannelFn},
		{Tool: LeaveChannelTool, Handler: LeaveChannelFn},
	}
	for _, t := range tools {
		AdminTool.RegisterWrite(t)
	}
}

func CreateChannelFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Channel] Called CreateChannelFn")

	args := req.GetArguments()

	teamID, err := params.GetString(args, "team_id")
	if err != nil {
		return to.Error(fmt.Errorf("[team_id] %v", err)), nil
	}

	name, err := params.GetString(args, "name")
	if err != nil {
		return to.Error(fmt.Errorf("[name] %v", err)), nil
	}

	displayName, err := params.GetString(args, "display_name")
	if err != nil {
		return to.Error(fmt.Errorf("[display_name] %v", err)), nil
	}

	channelType, err := params.GetString(args, "type")
	if err != nil {
		return to.Error(fmt.Errorf("[type] %v", err)), nil
	}

	if channelType != "O" && channelType != "P" {
		return to.Error(fmt.Errorf("[type] must be 'O' (public) or 'P' (private)")), nil
	}

	purpose := params.GetOptionalString(args, "purpose", "")

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	channel := &model.Channel{
		TeamId:      teamID,
		Name:        name,
		DisplayName: displayName,
		Type:        model.ChannelType(channelType),
		Purpose:     purpose,
	}

	created, err := client.CreateChannel(ctx, channel)
	if err != nil {
		return to.Error(fmt.Errorf("[channel] failed to create channel: %v", err)), nil
	}

	return to.Result(SlimChannel(created)), nil
}

func InviteToChannelFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Channel] Called InviteToChannelFn")

	args := req.GetArguments()

	channelID, err := params.GetString(args, "channel_id")
	if err != nil {
		return to.Error(fmt.Errorf("[channel_id] %v", err)), nil
	}

	userID, err := params.GetString(args, "user_id")
	if err != nil {
		return to.Error(fmt.Errorf("[user_id] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	member, err := client.AddChannelMember(ctx, channelID, userID)
	if err != nil {
		return to.Error(fmt.Errorf("[channel] failed to invite user to channel: %v", err)), nil
	}

	return to.Result(map[string]interface{}{
		"success":    true,
		"channel_id": channelID,
		"user_id":    userID,
		"member":     member,
		"message":    "User invited to channel successfully",
	}), nil
}

func DeleteChannelFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Channel] Called DeleteChannelFn")

	args := req.GetArguments()

	channelID, err := params.GetString(args, "channel_id")
	if err != nil {
		return to.Error(fmt.Errorf("[channel_id] %v", err)), nil
	}

	_ = params.GetOptionalBool(args, "permanent", false)

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	err = client.DeleteChannel(ctx, channelID)
	if err != nil {
		return to.Error(fmt.Errorf("[channel] failed to delete channel: %v", err)), nil
	}

	return to.Result(map[string]interface{}{
		"success":    true,
		"channel_id": channelID,
		"message":    "Channel deleted successfully",
	}), nil
}

func LeaveChannelFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Channel] Called LeaveChannelFn")

	args := req.GetArguments()

	channelID, err := params.GetString(args, "channel_id")
	if err != nil {
		return to.Error(fmt.Errorf("[channel_id] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	user, err := client.GetMe(ctx)
	if err != nil {
		return to.Error(fmt.Errorf("[user] failed to get current user: %v", err)), nil
	}

	err = client.RemoveChannelMember(ctx, channelID, user.Id)
	if err != nil {
		return to.Error(fmt.Errorf("[channel] failed to leave channel: %v", err)), nil
	}

	return to.Result(map[string]interface{}{
		"success":    true,
		"channel_id": channelID,
		"user_id":    user.Id,
		"message":    "Successfully left the channel",
	}), nil
}
