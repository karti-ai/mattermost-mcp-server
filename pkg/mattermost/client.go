// Package mattermost provides a wrapper around the Mattermost Go client
// with dual token support, timeout handling, retry logic, and error categorization.
package mattermost

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/karti-ai/mattermost-mcp-server/pkg/errors"
	"github.com/karti-ai/mattermost-mcp-server/pkg/log"
	"github.com/mattermost/mattermost-server/v6/model"
	"go.uber.org/zap"
)

const (
	// DefaultTimeout is the default timeout for all API requests
	DefaultTimeout = 30 * time.Second
	// MaxRetries is the maximum number of retries for 5xx errors
	MaxRetries = 3
	// InitialBackoff is the initial backoff duration for retries
	InitialBackoff = 500 * time.Millisecond
)

// Client wraps the Mattermost Go client with enhanced functionality
type Client struct {
	client   *model.Client4
	botToken string
	pat      string
	host     string
}

// Global client instance for use by operation handlers
var globalClient *Client

// SetGlobalClient sets the global client instance
func SetGlobalClient(c *Client) {
	globalClient = c
}

// GetGlobalClient returns the global client instance
func GetGlobalClient() *Client {
	return globalClient
}

// NewClient creates a new Mattermost client with the given configuration
func NewClient(host, botToken, pat string) *Client {
	c := model.NewAPIv4Client(host)
	// Initialize HTTPHeader map to store custom headers
	c.HTTPHeader = make(map[string]string)
	return &Client{
		client:   c,
		botToken: botToken,
		pat:      pat,
		host:     host,
	}
}

// getToken returns the appropriate token based on operation type
// Uses BotToken for read operations (GET), PAT for write operations (POST/PUT/DELETE)
func (c *Client) getToken(isWrite bool) string {
	if isWrite && c.pat != "" {
		return c.pat
	}
	if c.botToken != "" {
		return c.botToken
	}
	return c.pat
}

// setToken sets the appropriate token on the client based on HTTP method
func (c *Client) setToken(method string) {
	isWrite := method != http.MethodGet && method != http.MethodHead
	token := c.getToken(isWrite)
	if token != "" {
		c.client.HTTPHeader["Authorization"] = "Bearer " + token
	}
}

// isRetryableError checks if an error warrants a retry
func isRetryableError(resp *model.Response, err error) bool {
	if resp == nil {
		return true // Network errors should be retried
	}
	return resp.StatusCode >= 500 && resp.StatusCode < 600
}

// calculateBackoff calculates the backoff duration for retry attempts
func calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: 500ms, 1s, 2s
	backoff := InitialBackoff * time.Duration(math.Pow(2, float64(attempt)))
	// Add jitter
	return backoff + time.Duration(time.Now().UnixNano()%100)*time.Millisecond
}

// logRequest logs API request details (NEVER logs tokens)
func (c *Client) logRequest(ctx context.Context, method, path string) {
	logger := log.WithContext(ctx)
	logger.Debug("sending API request",
		zap.String("method", method),
		zap.String("path", path),
		zap.String("host", c.host),
	)
}

// logResponse logs API response details
func (c *Client) logResponse(ctx context.Context, method, path string, statusCode int, duration time.Duration, err error) {
	logger := log.WithContext(ctx)
	if err != nil {
		logger.Error("API request failed",
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status_code", statusCode),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
	} else {
		logger.Debug("API request completed",
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status_code", statusCode),
			zap.Duration("duration", duration),
		)
	}
}

// mapError converts Mattermost errors to categorized errors
func (c *Client) mapError(resp *model.Response, err error, operation string) error {
	if err == nil {
		return nil
	}

	var category errors.ErrorCategory
	var translated string
	statusCode := 0

	if resp != nil {
		statusCode = resp.StatusCode
	}

	switch statusCode {
	case 401:
		category = errors.CategoryAuth
		translated = "Authentication failed - check your access token"
	case 403:
		category = errors.CategoryAuth
		translated = "Permission denied - you don't have access to this resource"
	case 404:
		category = errors.CategoryUnknown
		translated = "Resource not found"
	case 429:
		category = errors.CategoryNetwork
		translated = "Rate limited - too many requests, please try again later"
	case 500, 502, 503, 504:
		category = errors.CategoryNetwork
		translated = "Server error - the Mattermost server encountered a problem"
	default:
		if resp == nil {
			category = errors.CategoryNetwork
			translated = "Connection failed - unable to reach Mattermost server"
		} else {
			// Try to extract error message from Mattermost API error
			if appErr, ok := err.(*model.AppError); ok && appErr.Message != "" {
				translated = appErr.Message
				category = c.categorizeAppError(appErr)
			} else {
				translated = err.Error()
				category = errors.CategoryUnknown
			}
		}
	}

	enhanced := errors.NewEnhancedError(err, translated, category)
	enhanced.WithOperation(operation)

	return enhanced
}

// categorizeAppError determines error category from Mattermost AppError
func (c *Client) categorizeAppError(appErr *model.AppError) errors.ErrorCategory {
	id := strings.ToLower(appErr.Id)
	message := strings.ToLower(appErr.Message)

	// Check error ID patterns
	switch {
	case strings.Contains(id, "authentication") || strings.Contains(id, "auth"):
		return errors.CategoryAuth
	case strings.Contains(id, "channel"):
		return errors.CategoryChannel
	case strings.Contains(id, "user"):
		return errors.CategoryUser
	case strings.Contains(id, "post") || strings.Contains(id, "message"):
		return errors.CategoryMessage
	case strings.Contains(id, "file"):
		return errors.CategoryFile
	}

	// Check message patterns
	switch {
	case strings.Contains(message, "channel") && strings.Contains(message, "not found"):
		return errors.CategoryChannel
	case strings.Contains(message, "user") && strings.Contains(message, "not found"):
		return errors.CategoryUser
	case strings.Contains(message, "post") && strings.Contains(message, "not found"):
		return errors.CategoryMessage
	case strings.Contains(message, "file") && strings.Contains(message, "not found"):
		return errors.CategoryFile
	case strings.Contains(message, "permission") || strings.Contains(message, "unauthorized"):
		return errors.CategoryAuth
	}

	return errors.CategoryUnknown
}

// executeWithRetry executes an API call with retry logic for 5xx errors
func (c *Client) executeWithRetry(
	ctx context.Context,
	operation string,
	method string,
	path string,
	fn func() (*model.Response, error),
) (*model.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	c.setToken(method)
	c.logRequest(ctx, method, path)

	start := time.Now()
	var lastErr error
	var resp *model.Response

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := calculateBackoff(attempt - 1)
			log.WithContext(ctx).Debug("retrying request",
				zap.String("operation", operation),
				zap.Int("attempt", attempt),
				zap.Duration("backoff", backoff),
			)
			time.Sleep(backoff)
		}

		resp, lastErr = fn()
		duration := time.Since(start)

		if lastErr == nil {
			if resp != nil {
				c.logResponse(ctx, method, path, resp.StatusCode, duration, nil)
			}
			return resp, nil
		}

		if !isRetryableError(resp, lastErr) {
			c.logResponse(ctx, method, path, 0, duration, lastErr)
			return resp, c.mapError(resp, lastErr, operation)
		}

		// Log retryable error
		if resp != nil {
			log.WithContext(ctx).Warn("retryable error occurred",
				zap.String("operation", operation),
				zap.Int("attempt", attempt),
				zap.Int("status_code", resp.StatusCode),
				zap.Error(lastErr),
			)
		}
	}

	// All retries exhausted
	duration := time.Since(start)
	c.logResponse(ctx, method, path, 0, duration, fmt.Errorf("max retries exceeded: %w", lastErr))
	return resp, c.mapError(resp, fmt.Errorf("max retries exceeded: %w", lastErr), operation)
}

// ==================== User Operations ====================

// GetMe retrieves the current authenticated user
func (c *Client) GetMe(ctx context.Context) (*model.User, error) {
	c.setToken(http.MethodGet)
	user, resp, err := c.client.GetMe("")
	if err != nil {
		return nil, c.mapError(resp, err, "GetMe")
	}
	return user, nil
}

// GetUser retrieves a user by ID
func (c *Client) GetUser(ctx context.Context, userID string) (*model.User, error) {
	c.setToken(http.MethodGet)
	user, resp, err := c.client.GetUser(userID, "")
	if err != nil {
		return nil, c.mapError(resp, err, "GetUser")
	}
	return user, nil
}

// GetUserByUsername retrieves a user by username
func (c *Client) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	c.setToken(http.MethodGet)
	user, resp, err := c.client.GetUserByUsername(username, "")
	if err != nil {
		return nil, c.mapError(resp, err, "GetUserByUsername")
	}
	return user, nil
}

// SearchUsers searches for users based on search criteria
func (c *Client) SearchUsers(ctx context.Context, search *model.UserSearch) ([]*model.User, error) {
	c.setToken(http.MethodPost)
	users, resp, err := c.client.SearchUsers(search)
	if err != nil {
		return nil, c.mapError(resp, err, "SearchUsers")
	}
	return users, nil
}

// GetUserStatus retrieves the status of a user (online, away, offline, dnd)
func (c *Client) GetUserStatus(ctx context.Context, userID string) (*model.Status, error) {
	c.setToken(http.MethodGet)
	status, resp, err := c.client.GetUserStatus(userID, "")
	if err != nil {
		return nil, c.mapError(resp, err, "GetUserStatus")
	}
	return status, nil
}

// ==================== Channel Operations ====================

// GetChannel retrieves a channel by ID
func (c *Client) GetChannel(ctx context.Context, channelID string) (*model.Channel, error) {
	c.setToken(http.MethodGet)
	channel, resp, err := c.client.GetChannel(channelID, "")
	if err != nil {
		return nil, c.mapError(resp, err, "GetChannel")
	}
	return channel, nil
}

// GetChannelByName retrieves a channel by name in a team
func (c *Client) GetChannelByName(ctx context.Context, teamID, channelName string) (*model.Channel, error) {
	c.setToken(http.MethodGet)
	channel, resp, err := c.client.GetChannelByName(channelName, teamID, "")
	if err != nil {
		return nil, c.mapError(resp, err, "GetChannelByName")
	}
	return channel, nil
}

// GetChannelsForTeamForUser retrieves channels for a user in a team
func (c *Client) GetChannelsForTeamForUser(ctx context.Context, teamID, userID string, includeDeleted bool) ([]*model.Channel, error) {
	c.setToken(http.MethodGet)
	channels, resp, err := c.client.GetChannelsForTeamForUser(teamID, userID, includeDeleted, "")
	if err != nil {
		return nil, c.mapError(resp, err, "GetChannelsForTeamForUser")
	}
	return channels, nil
}

// CreateDirectChannel creates a direct message channel between two users
func (c *Client) CreateDirectChannel(ctx context.Context, userID1, userID2 string) (*model.Channel, error) {
	c.setToken(http.MethodPost)
	channel, resp, err := c.client.CreateDirectChannel(userID1, userID2)
	if err != nil {
		return nil, c.mapError(resp, err, "CreateDirectChannel")
	}
	return channel, nil
}

// GetTeamUnread retrieves unread counts for a team
func (c *Client) GetTeamUnread(ctx context.Context, userID, teamID string) (*model.TeamUnread, error) {
	c.setToken(http.MethodGet)
	unread, resp, err := c.client.GetTeamUnread(userID, teamID)
	if err != nil {
		return nil, c.mapError(resp, err, "GetTeamUnread")
	}
	return unread, nil
}

// GetChannelMembersForUser retrieves channel members for a user in a team
func (c *Client) GetChannelMembersForUser(ctx context.Context, userID, teamID string) (model.ChannelMembers, error) {
	c.setToken(http.MethodGet)
	members, resp, err := c.client.GetChannelMembersForUser(userID, teamID, "")
	if err != nil {
		return nil, c.mapError(resp, err, "GetChannelMembersForUser")
	}
	return members, nil
}

// MarkChannelAsRead marks a channel as read for a user
func (c *Client) MarkChannelAsRead(ctx context.Context, channelID, userID string) (*model.ChannelViewResponse, error) {
	c.setToken(http.MethodPut)
	viewResp, resp, err := c.client.ViewChannel(userID, &model.ChannelView{ChannelId: channelID, PrevChannelId: ""})
	if err != nil {
		return nil, c.mapError(resp, err, "MarkChannelAsRead")
	}
	return viewResp, nil
}

// ==================== Post Operations ====================

// GetPostsForChannel retrieves posts for a channel
func (c *Client) GetPostsForChannel(ctx context.Context, channelID string, page, perPage int, collapsedThreads bool) (*model.PostList, error) {
	c.setToken(http.MethodGet)
	posts, resp, err := c.client.GetPostsForChannel(channelID, page, perPage, "", collapsedThreads)
	if err != nil {
		return nil, c.mapError(resp, err, "GetPostsForChannel")
	}
	return posts, nil
}

// SearchPosts searches for posts using terms
func (c *Client) SearchPosts(ctx context.Context, teamID, terms string, isOrSearch bool) (*model.PostList, error) {
	c.setToken(http.MethodPost)
	results, resp, err := c.client.SearchPosts(teamID, terms, isOrSearch)
	if err != nil {
		return nil, c.mapError(resp, err, "SearchPosts")
	}
	return results, nil
}

// CreatePost creates a new post in a channel
func (c *Client) CreatePost(ctx context.Context, post *model.Post) (*model.Post, error) {
	c.setToken(http.MethodPost)
	created, resp, err := c.client.CreatePost(post)
	if err != nil {
		return nil, c.mapError(resp, err, "CreatePost")
	}
	return created, nil
}

// UpdatePost updates an existing post
func (c *Client) UpdatePost(ctx context.Context, postID string, post *model.Post) (*model.Post, error) {
	c.setToken(http.MethodPut)
	updated, resp, err := c.client.UpdatePost(postID, post)
	if err != nil {
		return nil, c.mapError(resp, err, "UpdatePost")
	}
	return updated, nil
}

// DeletePost deletes a post by ID
func (c *Client) DeletePost(ctx context.Context, postID string) error {
	_, err := c.executeWithRetry(
		ctx,
		"DeletePost",
		http.MethodDelete,
		fmt.Sprintf("/api/v4/posts/%s", postID),
		func() (*model.Response, error) {
			r, e := c.client.DeletePost(postID)
			return r, e
		},
	)
	return err
}

// ==================== Reaction Operations ====================

// GetReactions retrieves all reactions for a post
func (c *Client) GetReactions(ctx context.Context, postID string) ([]*model.Reaction, error) {
	c.setToken(http.MethodGet)
	reactions, resp, err := c.client.GetReactions(postID)
	if err != nil {
		return nil, c.mapError(resp, err, "GetReactions")
	}
	return reactions, nil
}

// SaveReaction adds a reaction to a post
func (c *Client) SaveReaction(ctx context.Context, reaction *model.Reaction) (*model.Reaction, error) {
	c.setToken(http.MethodPost)
	saved, resp, err := c.client.SaveReaction(reaction)
	if err != nil {
		return nil, c.mapError(resp, err, "SaveReaction")
	}
	return saved, nil
}

// DeleteReaction removes a reaction from a post
func (c *Client) DeleteReaction(ctx context.Context, reaction *model.Reaction) error {
	_, err := c.executeWithRetry(
		ctx,
		"DeleteReaction",
		http.MethodDelete,
		"/api/v4/reactions",
		func() (*model.Response, error) {
			r, e := c.client.DeleteReaction(reaction)
			return r, e
		},
	)
	return err
}

// ==================== File Operations ====================

// UploadFile uploads a file to a channel
func (c *Client) UploadFile(ctx context.Context, data []byte, channelID, filename string) (*model.FileUploadResponse, error) {
	c.setToken(http.MethodPost)
	uploadResp, resp, err := c.client.UploadFile(data, channelID, filename)
	if err != nil {
		return nil, c.mapError(resp, err, "UploadFile")
	}
	return uploadResp, nil
}

// GetFile retrieves a file by ID
func (c *Client) GetFile(ctx context.Context, fileID string) ([]byte, *model.Response, error) {
	c.setToken(http.MethodGet)
	data, resp, err := c.client.GetFile(fileID)
	if err != nil {
		return nil, resp, c.mapError(resp, err, "GetFile")
	}
	return data, resp, nil
}

// ==================== Team Operations ====================

// GetTeamsForUser retrieves all teams a user is a member of
func (c *Client) GetTeamsForUser(ctx context.Context, userID string) ([]*model.Team, error) {
	c.setToken(http.MethodGet)
	teams, resp, err := c.client.GetTeamsForUser(userID, "")
	if err != nil {
		return nil, c.mapError(resp, err, "GetTeamsForUser")
	}
	return teams, nil
}

// GetChannelPosts retrieves posts for a channel with pagination support
func (c *Client) GetChannelPosts(ctx context.Context, channelID string, limit int, before, after string) (*model.PostList, error) {
	c.setToken(http.MethodGet)

	var posts *model.PostList
	var resp *model.Response
	var err error

	if before != "" {
		posts, resp, err = c.client.GetPostsBefore(channelID, before, 0, limit, "", false)
	} else if after != "" {
		posts, resp, err = c.client.GetPostsAfter(channelID, after, 0, limit, "", false)
	} else {
		posts, resp, err = c.client.GetPostsForChannel(channelID, 0, limit, "", false)
	}

	if err != nil {
		return nil, c.mapError(resp, err, "GetChannelPosts")
	}
	return posts, nil
}

// GetPostThread retrieves all posts in a thread (parent + replies)
func (c *Client) GetPostThread(ctx context.Context, postID string) (*model.PostList, error) {
	c.setToken(http.MethodGet)
	posts, resp, err := c.client.GetPostThread(postID, "", false)
	if err != nil {
		return nil, c.mapError(resp, err, "GetPostThread")
	}
	return posts, nil
}

// GetChannelMembers retrieves all members of a channel
func (c *Client) GetChannelMembers(ctx context.Context, channelID string, page, perPage int) (model.ChannelMembers, error) {
	c.setToken(http.MethodGet)
	members, resp, err := c.client.GetChannelMembers(channelID, page, perPage, "")
	if err != nil {
		return nil, c.mapError(resp, err, "GetChannelMembers")
	}
	return members, nil
}

// CreateGroupChannel creates a group DM channel with multiple users
func (c *Client) CreateGroupChannel(ctx context.Context, userIDs []string) (*model.Channel, error) {
	c.setToken(http.MethodPost)
	channel, resp, err := c.client.CreateGroupChannel(userIDs)
	if err != nil {
		return nil, c.mapError(resp, err, "CreateGroupChannel")
	}
	return channel, nil
}

// UpdateUserStatus updates the current user's status (online, away, dnd, offline)
func (c *Client) UpdateUserStatus(ctx context.Context, userID string, status string) (*model.Status, error) {
	c.setToken(http.MethodPut)
	userStatus := &model.Status{
		UserId: userID,
		Status: status,
	}
	updated, resp, err := c.client.UpdateUserStatus(userID, userStatus)
	if err != nil {
		return nil, c.mapError(resp, err, "UpdateUserStatus")
	}
	return updated, nil
}

// ==================== Admin Channel Operations ====================

// CreateChannel creates a new channel in a team
func (c *Client) CreateChannel(ctx context.Context, channel *model.Channel) (*model.Channel, error) {
	c.setToken(http.MethodPost)
	created, resp, err := c.client.CreateChannel(channel)
	if err != nil {
		return nil, c.mapError(resp, err, "CreateChannel")
	}
	return created, nil
}

// AddChannelMember adds a user to a channel
func (c *Client) AddChannelMember(ctx context.Context, channelID, userID string) (*model.ChannelMember, error) {
	c.setToken(http.MethodPost)
	member, resp, err := c.client.AddChannelMember(channelID, userID)
	if err != nil {
		return nil, c.mapError(resp, err, "AddChannelMember")
	}
	return member, nil
}

// DeleteChannel deletes/archival a channel (soft delete by default)
func (c *Client) DeleteChannel(ctx context.Context, channelID string) error {
	_, err := c.executeWithRetry(
		ctx,
		"DeleteChannel",
		http.MethodDelete,
		fmt.Sprintf("/api/v4/channels/%s", channelID),
		func() (*model.Response, error) {
			r, e := c.client.DeleteChannel(channelID)
			return r, e
		},
	)
	return err
}

// GetChannelStats retrieves statistics for a channel including member count
func (c *Client) GetChannelStats(ctx context.Context, channelID string) (*model.ChannelStats, error) {
	c.setToken(http.MethodGet)
	stats, resp, err := c.client.GetChannelStats(channelID, "")
	if err != nil {
		return nil, c.mapError(resp, err, "GetChannelStats")
	}
	return stats, nil
}

// RemoveChannelMember removes a user from a channel
func (c *Client) RemoveChannelMember(ctx context.Context, channelID, userID string) error {
	_, err := c.executeWithRetry(
		ctx,
		"RemoveChannelMember",
		http.MethodDelete,
		fmt.Sprintf("/api/v4/channels/%s/members/%s", channelID, userID),
		func() (*model.Response, error) {
			r, e := c.client.RemoveUserFromChannel(channelID, userID)
			return r, e
		},
	)
	return err
}

// ==================== Pin Operations ====================

// PinPost pins a post to a channel
func (c *Client) PinPost(ctx context.Context, postID string) error {
	c.setToken(http.MethodPost)
	_, err := c.client.PinPost(postID)
	if err != nil {
		return c.mapError(nil, err, "PinPost")
	}
	return nil
}

// UnpinPost unpins a post from a channel
func (c *Client) UnpinPost(ctx context.Context, postID string) error {
	c.setToken(http.MethodPost)
	_, err := c.client.UnpinPost(postID)
	if err != nil {
		return c.mapError(nil, err, "UnpinPost")
	}
	return nil
}

// GetPinnedPosts retrieves all pinned posts in a channel
func (c *Client) GetPinnedPosts(ctx context.Context, channelID string) (*model.PostList, error) {
	c.setToken(http.MethodGet)
	posts, resp, err := c.client.GetPinnedPosts(channelID, "")
	if err != nil {
		return nil, c.mapError(resp, err, "GetPinnedPosts")
	}
	return posts, nil
}

// ==================== Bulk Status Operations ====================

// GetUsersStatus retrieves status for multiple users by IDs
func (c *Client) GetUsersStatus(ctx context.Context, userIDs []string) ([]*model.Status, error) {
	c.setToken(http.MethodPost)
	statuses, resp, err := c.client.GetUsersStatusesByIds(userIDs)
	if err != nil {
		return nil, c.mapError(resp, err, "GetUsersStatus")
	}
	return statuses, nil
}

// ==================== Webhook Operations ====================

// CreateIncomingWebhook creates an incoming webhook for a channel
func (c *Client) CreateIncomingWebhook(ctx context.Context, channelID string, displayName string) (*model.IncomingWebhook, error) {
	c.setToken(http.MethodPost)

	// Get current user for webhook creation
	me, err := c.GetMe(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	hook := &model.IncomingWebhook{
		ChannelId:   channelID,
		DisplayName: displayName,
		UserId:      me.Id,
	}

	created, resp, err := c.client.CreateIncomingWebhook(hook)
	if err != nil {
		return nil, c.mapError(resp, err, "CreateIncomingWebhook")
	}
	return created, nil
}

// ListIncomingWebhooks lists incoming webhooks for a team
func (c *Client) ListIncomingWebhooks(ctx context.Context, teamID string, page, perPage int) ([]*model.IncomingWebhook, error) {
	c.setToken(http.MethodGet)
	hooks, resp, err := c.client.GetIncomingWebhooksForTeam(teamID, page, perPage, "")
	if err != nil {
		return nil, c.mapError(resp, err, "ListIncomingWebhooks")
	}
	return hooks, nil
}

// DeleteIncomingWebhook deletes an incoming webhook
func (c *Client) DeleteIncomingWebhook(ctx context.Context, hookID string) error {
	_, err := c.executeWithRetry(
		ctx,
		"DeleteIncomingWebhook",
		http.MethodDelete,
		fmt.Sprintf("/api/v4/hooks/incoming/%s", hookID),
		func() (*model.Response, error) {
			r, e := c.client.DeleteIncomingWebhook(hookID)
			return r, e
		},
	)
	return err
}

// ==================== Slash Commands ====================

// ExecuteSlashCommand runs a slash command in a channel
func (c *Client) ExecuteSlashCommand(ctx context.Context, channelID string, command string) (*model.CommandResponse, error) {
	c.setToken(http.MethodPost)
	resp, _, err := c.client.ExecuteCommand(channelID, command)
	if err != nil {
		return nil, c.mapError(nil, err, "ExecuteSlashCommand")
	}
	return resp, nil
}

// ==================== Team Administration ====================

// InviteUserToTeam adds a user to a team
func (c *Client) InviteUserToTeam(ctx context.Context, teamID, userID string) (*model.TeamMember, error) {
	c.setToken(http.MethodPost)
	member, resp, err := c.client.AddTeamMember(teamID, userID)
	if err != nil {
		return nil, c.mapError(resp, err, "InviteUserToTeam")
	}
	return member, nil
}

// RemoveUserFromTeam removes a user from a team
func (c *Client) RemoveUserFromTeam(ctx context.Context, teamID, userID string) error {
	_, err := c.executeWithRetry(
		ctx,
		"RemoveUserFromTeam",
		http.MethodDelete,
		fmt.Sprintf("/api/v4/teams/%s/members/%s", teamID, userID),
		func() (*model.Response, error) {
			r, e := c.client.RemoveTeamMember(teamID, userID)
			return r, e
		},
	)
	return err
}

// ListTeamMembers gets members of a team
func (c *Client) ListTeamMembers(ctx context.Context, teamID string, page, perPage int) ([]*model.TeamMember, error) {
	c.setToken(http.MethodGet)
	members, resp, err := c.client.GetTeamMembers(teamID, page, perPage, "")
	if err != nil {
		return nil, c.mapError(resp, err, "ListTeamMembers")
	}
	return members, nil
}

// GetTeamStats gets statistics for a team
func (c *Client) GetTeamStats(ctx context.Context, teamID string) (*model.TeamStats, error) {
	c.setToken(http.MethodGet)
	stats, resp, err := c.client.GetTeamStats(teamID, "")
	if err != nil {
		return nil, c.mapError(resp, err, "GetTeamStats")
	}
	return stats, nil
}

// ==================== Post Operations ====================

// GetPost retrieves a single post by ID
func (c *Client) GetPost(ctx context.Context, postID string) (*model.Post, error) {
	c.setToken(http.MethodGet)
	post, resp, err := c.client.GetPost(postID, "")
	if err != nil {
		return nil, c.mapError(resp, err, "GetPost")
	}
	return post, nil
}

// ==================== Outgoing Webhooks ====================

// CreateOutgoingWebhook creates an outgoing webhook
func (c *Client) CreateOutgoingWebhook(ctx context.Context, teamID string, displayName string, triggerWords []string, callbackURL string) (*model.OutgoingWebhook, error) {
	c.setToken(http.MethodPost)

	hook := &model.OutgoingWebhook{
		TeamId:       teamID,
		DisplayName:  displayName,
		TriggerWords: triggerWords,
		CallbackURLs: []string{callbackURL},
	}

	created, resp, err := c.client.CreateOutgoingWebhook(hook)
	if err != nil {
		return nil, c.mapError(resp, err, "CreateOutgoingWebhook")
	}
	return created, nil
}

// ListOutgoingWebhooks lists outgoing webhooks for a team
func (c *Client) ListOutgoingWebhooks(ctx context.Context, teamID string, page, perPage int) ([]*model.OutgoingWebhook, error) {
	c.setToken(http.MethodGet)
	hooks, resp, err := c.client.GetOutgoingWebhooksForTeam(teamID, page, perPage, "")
	if err != nil {
		return nil, c.mapError(resp, err, "ListOutgoingWebhooks")
	}
	return hooks, nil
}

// DeleteOutgoingWebhook deletes an outgoing webhook
func (c *Client) DeleteOutgoingWebhook(ctx context.Context, hookID string) error {
	_, err := c.executeWithRetry(
		ctx,
		"DeleteOutgoingWebhook",
		http.MethodDelete,
		fmt.Sprintf("/api/v4/hooks/outgoing/%s", hookID),
		func() (*model.Response, error) {
			r, e := c.client.DeleteOutgoingWebhook(hookID)
			return r, e
		},
	)
	return err
}

// ==================== System & Config ====================

// GetSystemLogs retrieves system logs (requires admin)
func (c *Client) GetSystemLogs(ctx context.Context, page, perPage int) ([]string, *model.Response, error) {
	c.setToken(http.MethodGet)
	logs, resp, err := c.client.GetLogs(page, perPage)
	if err != nil {
		return nil, resp, c.mapError(resp, err, "GetSystemLogs")
	}
	return logs, resp, nil
}

// GetConfig retrieves server configuration (requires admin)
func (c *Client) GetConfig(ctx context.Context) (*model.Config, error) {
	c.setToken(http.MethodGet)
	config, resp, err := c.client.GetConfig()
	if err != nil {
		return nil, c.mapError(resp, err, "GetConfig")
	}
	return config, nil
}
