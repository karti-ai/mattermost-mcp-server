package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/karti-ai/mattermost-mcp-server/pkg/log"
	"github.com/karti-ai/mattermost-mcp-server/pkg/mattermost"
	"github.com/karti-ai/mattermost-mcp-server/pkg/params"
	"github.com/karti-ai/mattermost-mcp-server/pkg/to"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	GetUserStatusToolName      = "mattermost_get_user_status"
	UpdateUserStatusToolName   = "mattermost_update_user_status"
	GetUsersStatusBulkToolName = "mattermost_get_users_status_bulk"
)

var (
	GetUserStatusTool = mcp.NewTool(
		GetUserStatusToolName,
		mcp.WithDescription("Get the online status of a user (online, away, dnd, offline)"),
		mcp.WithString("user_id", mcp.Required(), mcp.Description("User ID to check status for")),
	)

	UpdateUserStatusTool = mcp.NewTool(
		UpdateUserStatusToolName,
		mcp.WithDescription("Update your status (online, away, dnd, offline)"),
		mcp.WithString("status", mcp.Required(), mcp.Description("Status to set: online, away, dnd, or offline")),
	)

	GetUsersStatusBulkTool = mcp.NewTool(
		GetUsersStatusBulkToolName,
		mcp.WithDescription("Get status for multiple users at once (up to 100)"),
		mcp.WithString("user_ids", mcp.Required(), mcp.Description("Comma-separated list of user IDs (max 100)")),
	)
)

func init() {
	registerStatusTools()
}

func registerStatusTools() {
	tools := []server.ServerTool{
		{Tool: GetUserStatusTool, Handler: GetUserStatusFn},
		{Tool: UpdateUserStatusTool, Handler: UpdateUserStatusFn},
		{Tool: GetUsersStatusBulkTool, Handler: GetUsersStatusBulkFn},
	}
	for _, t := range tools {
		if t.Tool.Name == UpdateUserStatusToolName {
			Tool.RegisterWrite(t)
		} else {
			Tool.RegisterRead(t)
		}
	}
}

func GetUserStatusFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[User] Called GetUserStatusFn")

	args := req.GetArguments()

	userID, err := params.GetString(args, "user_id")
	if err != nil {
		return to.Error(fmt.Errorf("[user_id] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	status, err := client.GetUserStatus(ctx, userID)
	if err != nil {
		return to.Error(fmt.Errorf("[status] failed to get user status: %v", err)), nil
	}

	return to.Result(map[string]interface{}{
		"user_id":          status.UserId,
		"status":           status.Status,
		"manual":           status.Manual,
		"last_activity_at": status.LastActivityAt,
	}), nil
}

func UpdateUserStatusFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[User] Called UpdateUserStatusFn")

	args := req.GetArguments()

	status, err := params.GetString(args, "status")
	if err != nil {
		return to.Error(fmt.Errorf("[status] %v", err)), nil
	}

	// Validate status value
	validStatuses := map[string]bool{"online": true, "away": true, "dnd": true, "offline": true}
	if !validStatuses[status] {
		return to.Error(fmt.Errorf("[status] invalid status '%s', must be one of: online, away, dnd, offline", status)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	// Get current user ID
	me, err := client.GetMe(ctx)
	if err != nil {
		return to.Error(fmt.Errorf("[me] failed to get current user: %v", err)), nil
	}

	updatedStatus, err := client.UpdateUserStatus(ctx, me.Id, status)
	if err != nil {
		return to.Error(fmt.Errorf("[status] failed to update status: %v", err)), nil
	}

	return to.Result(map[string]interface{}{
		"user_id": updatedStatus.UserId,
		"status":  updatedStatus.Status,
		"message": "Status updated successfully",
	}), nil
}

func GetUsersStatusBulkFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[User] Called GetUsersStatusBulkFn")

	args := req.GetArguments()

	userIDsStr, err := params.GetString(args, "user_ids")
	if err != nil {
		return to.Error(fmt.Errorf("[user_ids] %v", err)), nil
	}

	userIDs := strings.Split(userIDsStr, ",")
	if len(userIDs) > 100 {
		return to.Error(fmt.Errorf("[user_ids] too many user IDs (max 100, got %d)", len(userIDs))), nil
	}

	for i, id := range userIDs {
		userIDs[i] = strings.TrimSpace(id)
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	statuses, err := client.GetUsersStatus(ctx, userIDs)
	if err != nil {
		return to.Error(fmt.Errorf("[status] failed to get users status: %v", err)), nil
	}

	results := make([]map[string]interface{}, 0, len(statuses))
	for _, status := range statuses {
		results = append(results, map[string]interface{}{
			"user_id":          status.UserId,
			"status":           status.Status,
			"manual":           status.Manual,
			"last_activity_at": status.LastActivityAt,
		})
	}

	return to.Result(map[string]interface{}{
		"statuses": results,
		"count":    len(results),
	}), nil
}
