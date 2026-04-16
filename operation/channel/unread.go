package channel

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
	GetUnreadCountToolName  = "mattermost_get_unread_count"
	MarkChannelReadToolName = "mattermost_mark_channel_read"
)

var (
	GetUnreadCountTool = mcp.NewTool(
		GetUnreadCountToolName,
		mcp.WithDescription("Get unread message counts for all channels in a team"),
		mcp.WithString("team_id", mcp.Required(), mcp.Description("Team ID to get unread counts for")),
	)

	MarkChannelReadTool = mcp.NewTool(
		MarkChannelReadToolName,
		mcp.WithDescription("Mark a channel as read (clear unread notifications)"),
		mcp.WithString("channel_id", mcp.Required(), mcp.Description("Channel ID to mark as read")),
	)
)

func init() {
	registerUnreadTools()
}

func registerUnreadTools() {
	tools := []server.ServerTool{
		{Tool: GetUnreadCountTool, Handler: GetUnreadCountFn},
		{Tool: MarkChannelReadTool, Handler: MarkChannelReadFn},
	}
	for _, t := range tools {
		Tool.RegisterRead(t)
	}
}

func GetUnreadCountFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Channel] Called GetUnreadCountFn")

	args := req.GetArguments()

	teamID, err := params.GetString(args, "team_id")
	if err != nil {
		return to.Error(fmt.Errorf("[team_id] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	user, err := client.GetMe(ctx)
	if err != nil {
		return to.Error(fmt.Errorf("[user] failed to get current user: %v", err)), nil
	}

	members, err := client.GetChannelMembersForUser(ctx, user.Id, teamID)
	if err != nil {
		return to.Error(fmt.Errorf("[channels] failed to get channel members: %v", err)), nil
	}

	memberResults := make([]map[string]interface{}, 0, len(members))
	for _, member := range members {
		memberResults = append(memberResults, map[string]interface{}{
			"channel_id":      member.ChannelId,
			"user_id":         member.UserId,
			"unread_messages": member.MsgCount,
			"unread_mentions": member.MentionCount,
			"last_viewed_at":  member.LastViewedAt,
		})
	}

	return to.Result(map[string]interface{}{
		"channels": memberResults,
		"count":    len(memberResults),
	}), nil
}

func MarkChannelReadFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[Channel] Called MarkChannelReadFn")

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

	_, err = client.MarkChannelAsRead(ctx, channelID, user.Id)
	if err != nil {
		return to.Error(fmt.Errorf("[channel] failed to mark channel as read: %v", err)), nil
	}

	return to.Result(map[string]interface{}{
		"success":    true,
		"channel_id": channelID,
		"message":    "Channel marked as read successfully",
	}), nil
}
