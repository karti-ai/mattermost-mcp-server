package operation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/karti-ai/mattermost-mcp-server/operation/channel"
	"github.com/karti-ai/mattermost-mcp-server/operation/dm"
	"github.com/karti-ai/mattermost-mcp-server/operation/file"
	"github.com/karti-ai/mattermost-mcp-server/operation/messaging"
	"github.com/karti-ai/mattermost-mcp-server/operation/reaction"
	"github.com/karti-ai/mattermost-mcp-server/operation/team"
	"github.com/karti-ai/mattermost-mcp-server/operation/user"
	mmerrors "github.com/karti-ai/mattermost-mcp-server/pkg/errors"
	"github.com/karti-ai/mattermost-mcp-server/pkg/flag"
	"github.com/karti-ai/mattermost-mcp-server/pkg/mattermost"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMattermostServer creates a mock Mattermost server for testing
type mockMattermostServer struct {
	server      *httptest.Server
	users       map[string]*model.User
	channels    map[string]*model.Channel
	posts       map[string]*model.Post
	reactions   map[string]*model.Reaction
	callHistory []mockCall
}

type mockCall struct {
	Method string
	Path   string
	Body   interface{}
}

func newMockMattermostServer() *mockMattermostServer {
	m := &mockMattermostServer{
		users:     make(map[string]*model.User),
		channels:  make(map[string]*model.Channel),
		posts:     make(map[string]*model.Post),
		reactions: make(map[string]*model.Reaction),
	}

	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.handleRequest(w, r)
	}))

	return m
}

func (m *mockMattermostServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	m.callHistory = append(m.callHistory, mockCall{
		Method: r.Method,
		Path:   r.URL.Path,
	})

	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.URL.Path == "/api/v4/users/me":
		m.handleGetMe(w, r)
	case r.URL.Path == "/api/v4/users/search":
		m.handleSearchUsers(w, r)
	case r.URL.Path == "/api/v4/channels/direct":
		m.handleCreateDirectChannel(w, r)
	case r.URL.Path == "/api/v4/posts":
		m.handleCreatePost(w, r)
	case r.Method == "PUT" && len(r.URL.Path) > 14 && strings.HasPrefix(r.URL.Path, "/api/v4/posts/"):
		m.handleUpdatePost(w, r)
	case r.Method == "DELETE" && len(r.URL.Path) > 14 && strings.HasPrefix(r.URL.Path, "/api/v4/posts/"):
		m.handleDeletePost(w, r)
	case r.URL.Path == "/api/v4/reactions":
		m.handleReaction(w, r)
	case r.Method == "GET" && len(r.URL.Path) > 17 && strings.HasPrefix(r.URL.Path, "/api/v4/channels/"):
		m.handleGetChannel(w, r)
	case len(r.URL.Path) > 13 && strings.HasPrefix(r.URL.Path, "/api/v4/users/"):
		m.handleGetChannelsForTeamForUser(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Not found"})
	}
}

func (m *mockMattermostServer) handleGetMe(w http.ResponseWriter, r *http.Request) {
	user := &model.User{
		Id:        "user-current",
		Username:  "currentuser",
		Email:     "current@example.com",
		FirstName: "Current",
		LastName:  "User",
		Roles:     "system_user",
	}
	json.NewEncoder(w).Encode(user)
}

func (m *mockMattermostServer) handleSearchUsers(w http.ResponseWriter, r *http.Request) {
	var search model.UserSearch
	json.NewDecoder(r.Body).Decode(&search)

	users := []*model.User{}
	for _, u := range m.users {
		if search.Term == "" ||
			u.Username == search.Term ||
			u.Email == search.Term ||
			u.FirstName+" "+u.LastName == search.Term {
			users = append(users, u)
		}
	}

	// Add default user if no results
	if len(users) == 0 && search.Term != "" {
		users = append(users, &model.User{
			Id:        "user-" + search.Term,
			Username:  search.Term,
			Email:     search.Term + "@example.com",
			FirstName: "Test",
			LastName:  "User",
			Roles:     "system_user",
		})
	}

	json.NewEncoder(w).Encode(users)
}

func (m *mockMattermostServer) handleCreateDirectChannel(w http.ResponseWriter, r *http.Request) {
	var userIds []string
	json.NewDecoder(r.Body).Decode(&userIds)

	if len(userIds) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Need exactly 2 user IDs"})
		return
	}

	channel := &model.Channel{
		Id:          "dm-" + userIds[0] + "-" + userIds[1],
		Name:        userIds[0] + "__" + userIds[1],
		DisplayName: "Direct Message",
		Type:        model.ChannelTypeDirect,
		TeamId:      "",
	}
	m.channels[channel.Id] = channel
	json.NewEncoder(w).Encode(channel)
}

func (m *mockMattermostServer) handleCreatePost(w http.ResponseWriter, r *http.Request) {
	var post model.Post
	json.NewDecoder(r.Body).Decode(&post)

	post.Id = "post-" + fmt.Sprintf("%d", time.Now().UnixNano())
	post.CreateAt = time.Now().UnixMilli()
	post.UpdateAt = post.CreateAt
	m.posts[post.Id] = &post

	json.NewEncoder(w).Encode(post)
}

func (m *mockMattermostServer) handleUpdatePost(w http.ResponseWriter, r *http.Request) {
	postID := r.URL.Path[14:] // Remove "/api/v4/posts/"
	var post model.Post
	json.NewDecoder(r.Body).Decode(&post)

	if existing, ok := m.posts[postID]; ok {
		existing.Message = post.Message
		existing.UpdateAt = time.Now().UnixMilli()
		json.NewEncoder(w).Encode(existing)
		return
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"message": "Post not found"})
}

func (m *mockMattermostServer) handleDeletePost(w http.ResponseWriter, r *http.Request) {
	postID := r.URL.Path[14:] // Remove "/api/v4/posts/"

	if _, ok := m.posts[postID]; ok {
		delete(m.posts, postID)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "OK"})
		return
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"message": "Post not found"})
}

func (m *mockMattermostServer) handleReaction(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var reaction model.Reaction
		json.NewDecoder(r.Body).Decode(&reaction)
		reaction.CreateAt = time.Now().UnixMilli()
		m.reactions[reaction.PostId+"_"+reaction.EmojiName] = &reaction
		json.NewEncoder(w).Encode(reaction)
	case "DELETE":
		var reaction model.Reaction
		json.NewDecoder(r.Body).Decode(&reaction)
		delete(m.reactions, reaction.PostId+"_"+reaction.EmojiName)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "OK"})
	}
}

func (m *mockMattermostServer) handleGetChannel(w http.ResponseWriter, r *http.Request) {
	channelID := r.URL.Path[17:] // Remove "/api/v4/channels/"

	// Check if it's a channels endpoint
	if len(channelID) > 8 && channelID[len(channelID)-8:] == "/posts" {
		// This is a posts request, handle separately
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"posts": []map[string]interface{}{}})
		return
	}

	if channel, ok := m.channels[channelID]; ok {
		json.NewEncoder(w).Encode(channel)
		return
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"message": "Channel not found"})
}

func (m *mockMattermostServer) handleGetChannelsForTeamForUser(w http.ResponseWriter, r *http.Request) {
	// Path format: /api/v4/users/{userID}/teams/{teamID}/channels
	channels := []*model.Channel{
		{
			Id:          "channel-1",
			Name:        "town-square",
			DisplayName: "Town Square",
			Type:        model.ChannelTypeOpen,
			TeamId:      "team-1",
		},
		{
			Id:          "channel-2",
			Name:        "off-topic",
			DisplayName: "Off-Topic",
			Type:        model.ChannelTypeOpen,
			TeamId:      "team-1",
		},
		{
			Id:          "channel-3",
			Name:        "private-channel",
			DisplayName: "Private Channel",
			Type:        model.ChannelTypePrivate,
			TeamId:      "team-1",
		},
	}
	json.NewEncoder(w).Encode(channels)
}

func (m *mockMattermostServer) URL() string {
	return m.server.URL
}

func (m *mockMattermostServer) Close() {
	m.server.Close()
}

func (m *mockMattermostServer) GetCallHistory() []mockCall {
	return m.callHistory
}

func (m *mockMattermostServer) ClearHistory() {
	m.callHistory = nil
}

func setupMockClient(mockServer *mockMattermostServer) *mattermost.Client {
	client := mattermost.NewClient(mockServer.URL(), "test-bot-token", "test-pat")
	mattermost.SetGlobalClient(client)
	return client
}

// ==================== Cross-Tool Workflow Tests ====================

func TestIntegration_DMWorkflow_CreateDMSendMessageAddReaction(t *testing.T) {
	mock := newMockMattermostServer()
	defer mock.Close()

	setupMockClient(mock)

	// Step 1: Search for a user
	searchReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: user.SearchUsersToolName,
			Arguments: map[string]interface{}{
				"term": "testuser",
			},
		},
	}

	searchResult, err := user.SearchUsersFn(context.Background(), searchReq)
	require.NoError(t, err)
	require.NotNil(t, searchResult)
	require.False(t, searchResult.IsError, "Search should succeed")

	// Parse search result to get user ID
	var searchData map[string]interface{}
	err = json.Unmarshal([]byte(searchResult.Content[0].(mcp.TextContent).Text), &searchData)
	require.NoError(t, err)

	users := searchData["users"].([]interface{})
	require.Greater(t, len(users), 0, "Should find at least one user")

	userData := users[0].(map[string]interface{})
	userID := userData["id"].(string)

	// Step 2: Create DM channel with the user
	dmReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: dm.CreateDMToolName,
			Arguments: map[string]interface{}{
				"user_id": userID,
			},
		},
	}

	dmResult, err := dm.CreateDMFn(context.Background(), dmReq)
	require.NoError(t, err)
	require.NotNil(t, dmResult)
	require.False(t, dmResult.IsError, "DM creation should succeed")

	// Parse DM result to get channel ID
	var dmData map[string]interface{}
	err = json.Unmarshal([]byte(dmResult.Content[0].(mcp.TextContent).Text), &dmData)
	require.NoError(t, err)

	channelID := dmData["id"].(string)
	require.NotEmpty(t, channelID)

	// Step 3: Send a message to the DM channel
	msgReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: messaging.SendMessageToolName,
			Arguments: map[string]interface{}{
				"channel_id": channelID,
				"message":    "Hello from integration test!",
			},
		},
	}

	msgResult, err := messaging.SendMessageFn(context.Background(), msgReq)
	require.NoError(t, err)
	require.NotNil(t, msgResult)
	require.False(t, msgResult.IsError, "Message send should succeed")

	// Parse message result to get post ID
	var msgData map[string]interface{}
	err = json.Unmarshal([]byte(msgResult.Content[0].(mcp.TextContent).Text), &msgData)
	require.NoError(t, err)

	postID := msgData["id"].(string)
	require.NotEmpty(t, postID)

	// Step 4: Add a reaction to the message
	reactionReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: reaction.AddReactionToolName,
			Arguments: map[string]interface{}{
				"post_id":    postID,
				"emoji_name": "thumbsup",
			},
		},
	}

	reactionResult, err := reaction.AddReactionFn(context.Background(), reactionReq)
	require.NoError(t, err)
	require.NotNil(t, reactionResult)
	require.False(t, reactionResult.IsError, "Reaction add should succeed")

	// Verify the reaction result
	var reactionData map[string]interface{}
	err = json.Unmarshal([]byte(reactionResult.Content[0].(mcp.TextContent).Text), &reactionData)
	require.NoError(t, err)

	assert.Equal(t, postID, reactionData["post_id"])
	assert.Equal(t, "thumbsup", reactionData["emoji_name"])

	// Verify call history shows the workflow sequence
	history := mock.GetCallHistory()
	require.GreaterOrEqual(t, len(history), 4, "Should have at least 4 API calls")
}

func TestIntegration_ChannelWorkflow_ListSendEditDelete(t *testing.T) {
	mock := newMockMattermostServer()
	defer mock.Close()

	setupMockClient(mock)

	// Step 1: List channels for a team
	listReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: channel.ListChannelsToolName,
			Arguments: map[string]interface{}{
				"team_id": "team-1",
				"limit":   10,
			},
		},
	}

	listResult, err := channel.ListChannelsFn(context.Background(), listReq)
	require.NoError(t, err)
	require.NotNil(t, listResult)
	require.False(t, listResult.IsError, "List channels should succeed")

	// Parse channel list result
	var listData map[string]interface{}
	err = json.Unmarshal([]byte(listResult.Content[0].(mcp.TextContent).Text), &listData)
	require.NoError(t, err)

	channels := listData["channels"].([]interface{})
	require.GreaterOrEqual(t, len(channels), 1, "Should have at least one channel")

	channelData := channels[0].(map[string]interface{})
	channelID := channelData["id"].(string)

	// Step 2: Send a message to the channel
	msgReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: messaging.SendMessageToolName,
			Arguments: map[string]interface{}{
				"channel_id": channelID,
				"message":    "Original message",
			},
		},
	}

	msgResult, err := messaging.SendMessageFn(context.Background(), msgReq)
	require.NoError(t, err)
	require.NotNil(t, msgResult)
	require.False(t, msgResult.IsError)

	var msgData map[string]interface{}
	err = json.Unmarshal([]byte(msgResult.Content[0].(mcp.TextContent).Text), &msgData)
	require.NoError(t, err)

	postID := msgData["id"].(string)

	// Step 3: Edit the message
	editReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: messaging.EditMessageToolName,
			Arguments: map[string]interface{}{
				"post_id": postID,
				"message": "Edited message",
			},
		},
	}

	editResult, err := messaging.EditMessageFn(context.Background(), editReq)
	require.NoError(t, err)
	require.NotNil(t, editResult)
	require.False(t, editResult.IsError, "Edit message should succeed")

	// Verify edit result
	var editData map[string]interface{}
	err = json.Unmarshal([]byte(editResult.Content[0].(mcp.TextContent).Text), &editData)
	require.NoError(t, err)

	assert.Equal(t, "Edited message", editData["message"])

	// Step 4: Delete the message
	deleteReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: messaging.DeleteMessageToolName,
			Arguments: map[string]interface{}{
				"post_id": postID,
			},
		},
	}

	deleteResult, err := messaging.DeleteMessageFn(context.Background(), deleteReq)
	require.NoError(t, err)
	require.NotNil(t, deleteResult)
	require.False(t, deleteResult.IsError, "Delete message should succeed")

	// Verify delete result
	var deleteData map[string]interface{}
	err = json.Unmarshal([]byte(deleteResult.Content[0].(mcp.TextContent).Text), &deleteData)
	require.NoError(t, err)

	assert.True(t, deleteData["success"].(bool))
	assert.Equal(t, postID, deleteData["post_id"])
}

func TestIntegration_SearchUserCreateDMSendMessage(t *testing.T) {
	mock := newMockMattermostServer()
	defer mock.Close()

	setupMockClient(mock)

	// Add a specific user to search for
	targetUser := &model.User{
		Id:        "user-target-123",
		Username:  "targetuser",
		Email:     "target@example.com",
		FirstName: "Target",
		LastName:  "User",
		Roles:     "system_user",
	}
	mock.users[targetUser.Id] = targetUser

	// Step 1: Search for the user
	searchReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: user.SearchUsersToolName,
			Arguments: map[string]interface{}{
				"term":    "targetuser",
				"team_id": "team-1",
				"limit":   5,
			},
		},
	}

	searchResult, err := user.SearchUsersFn(context.Background(), searchReq)
	require.NoError(t, err)
	require.NotNil(t, searchResult)
	require.False(t, searchResult.IsError)

	var searchData map[string]interface{}
	err = json.Unmarshal([]byte(searchResult.Content[0].(mcp.TextContent).Text), &searchData)
	require.NoError(t, err)

	// Step 2: Create DM with found user
	foundUsers := searchData["users"].([]interface{})
	require.GreaterOrEqual(t, len(foundUsers), 1)

	foundUser := foundUsers[0].(map[string]interface{})
	userID := foundUser["id"].(string)

	dmReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: dm.CreateDMToolName,
			Arguments: map[string]interface{}{
				"user_id": userID,
			},
		},
	}

	dmResult, err := dm.CreateDMFn(context.Background(), dmReq)
	require.NoError(t, err)
	require.NotNil(t, dmResult)
	require.False(t, dmResult.IsError)

	var dmData map[string]interface{}
	err = json.Unmarshal([]byte(dmResult.Content[0].(mcp.TextContent).Text), &dmData)
	require.NoError(t, err)

	channelID := dmData["id"].(string)

	// Step 3: Send message in DM
	msgReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: messaging.SendMessageToolName,
			Arguments: map[string]interface{}{
				"channel_id": channelID,
				"message":    "Message to target user",
			},
		},
	}

	msgResult, err := messaging.SendMessageFn(context.Background(), msgReq)
	require.NoError(t, err)
	require.NotNil(t, msgResult)
	require.False(t, msgResult.IsError)

	var msgData map[string]interface{}
	err = json.Unmarshal([]byte(msgResult.Content[0].(mcp.TextContent).Text), &msgData)
	require.NoError(t, err)

	assert.Equal(t, channelID, msgData["channel_id"])
	assert.Equal(t, "Message to target user", msgData["message"])
}

// ==================== Read-Only Mode Tests ====================

func TestIntegration_ReadOnlyMode_WriteToolsExcluded(t *testing.T) {
	// Save original read-only setting
	originalReadOnly := flag.ReadOnly
	defer func() {
		flag.ReadOnly = originalReadOnly
	}()

	// Test with read-only = false (full access)
	flag.ReadOnly = false
	allTools := Register()

	// Count tools
	writeToolCount := 0
	readToolCount := 0
	for _, tool := range allTools {
		switch tool.Tool.Name {
		case messaging.SendMessageToolName, messaging.EditMessageToolName, messaging.DeleteMessageToolName,
			dm.CreateDMToolName, reaction.AddReactionToolName, reaction.RemoveReactionToolName,
			file.UploadFileToolName, file.DownloadFileToolName:
			writeToolCount++
		case channel.ListChannelsToolName, user.SearchUsersToolName,
			team.ListTeamsToolName, messaging.GetChannelMessagesToolName, messaging.GetThreadToolName,
			messaging.SearchPostsToolName, user.GetUserStatusToolName,
			channel.GetChannelByNameToolName, channel.GetUnreadCountToolName, channel.MarkChannelReadToolName:
			readToolCount++
		}
	}

	assert.Greater(t, writeToolCount, 0, "Should have write tools in non-read-only mode")
	assert.Greater(t, readToolCount, 0, "Should have read tools in non-read-only mode")

	// Test with read-only = true (read-only access)
	flag.ReadOnly = true
	readOnlyTools := Register()

	// In read-only mode, only read tools should be present
	for _, tool := range readOnlyTools {
		switch tool.Tool.Name {
		case messaging.SendMessageToolName, messaging.EditMessageToolName, messaging.DeleteMessageToolName,
			dm.CreateDMToolName, reaction.AddReactionToolName, reaction.RemoveReactionToolName:
			t.Errorf("Write tool %s should not be present in read-only mode", tool.Tool.Name)
		}
	}

	// Verify read tools are still present
	foundReadTools := 0
	for _, tool := range readOnlyTools {
		switch tool.Tool.Name {
		case channel.ListChannelsToolName, user.SearchUsersToolName,
			team.ListTeamsToolName, messaging.GetChannelMessagesToolName, messaging.GetThreadToolName,
			messaging.SearchPostsToolName, user.GetUserStatusToolName,
			channel.GetChannelByNameToolName, channel.GetUnreadCountToolName, channel.MarkChannelReadToolName:
			foundReadTools++
		}
	}

	assert.Greater(t, foundReadTools, 0, "Read tools should be present in read-only mode")
}

func TestIntegration_ReadOnlyMode_ToolCountVerification(t *testing.T) {
	originalReadOnly := flag.ReadOnly
	defer func() {
		flag.ReadOnly = originalReadOnly
	}()

	// Full mode - should have all tools
	flag.ReadOnly = false
	fullTools := Register()
	fullCount := len(fullTools)

	// Read-only mode - should have only read tools
	flag.ReadOnly = true
	readOnlyTools := Register()
	readOnlyCount := len(readOnlyTools)

	// Read-only should have fewer tools
	assert.Less(t, readOnlyCount, fullCount, "Read-only mode should have fewer tools than full mode")

	// Verify the difference is the count of write tools
	flag.ReadOnly = false
	tools := Register()
	writeCount := 0
	readCount := 0
	for _, tool := range tools {
		switch tool.Tool.Name {
		case messaging.SendMessageToolName, messaging.EditMessageToolName, messaging.DeleteMessageToolName,
			dm.CreateDMToolName, reaction.AddReactionToolName, reaction.RemoveReactionToolName,
			file.UploadFileToolName, file.DownloadFileToolName,
			channel.CreateChannelToolName, channel.InviteToChannelToolName, channel.DeleteChannelToolName, channel.LeaveChannelToolName:
			writeCount++
		case channel.ListChannelsToolName, user.SearchUsersToolName,
			team.ListTeamsToolName, messaging.GetChannelMessagesToolName, messaging.GetThreadToolName,
			messaging.SearchPostsToolName, user.GetUserStatusToolName,
			channel.GetChannelByNameToolName, channel.GetUnreadCountToolName, channel.MarkChannelReadToolName,
			channel.GetChannelInfoToolName:
			readCount++
		}
	}

	assert.Equal(t, fullCount, writeCount+readCount, "Total tools should equal write + read tools")
	assert.Equal(t, readCount, readOnlyCount, "Read-only mode should only expose read tools")
}

// ==================== Error Propagation Tests ====================

func TestIntegration_ErrorPropagation_ClientNotInitialized(t *testing.T) {
	// Ensure no client is set
	mattermost.SetGlobalClient(nil)

	testCases := []struct {
		name     string
		toolName string
		handler  func(interface{}, mcp.CallToolRequest) (*mcp.CallToolResult, error)
		request  mcp.CallToolRequest
	}{
		{
			name:     "SendMessage",
			toolName: messaging.SendMessageToolName,
			handler: func(_ interface{}, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return messaging.SendMessageFn(context.Background(), req)
			},
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: messaging.SendMessageToolName,
					Arguments: map[string]interface{}{
						"channel_id": "channel123",
						"message":    "test",
					},
				},
			},
		},
		{
			name:     "EditMessage",
			toolName: messaging.EditMessageToolName,
			handler: func(_ interface{}, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return messaging.EditMessageFn(context.Background(), req)
			},
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: messaging.EditMessageToolName,
					Arguments: map[string]interface{}{
						"post_id": "post123",
						"message": "test",
					},
				},
			},
		},
		{
			name:     "DeleteMessage",
			toolName: messaging.DeleteMessageToolName,
			handler: func(_ interface{}, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return messaging.DeleteMessageFn(context.Background(), req)
			},
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: messaging.DeleteMessageToolName,
					Arguments: map[string]interface{}{
						"post_id": "post123",
					},
				},
			},
		},
		{
			name:     "CreateDM",
			toolName: dm.CreateDMToolName,
			handler: func(_ interface{}, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return dm.CreateDMFn(context.Background(), req)
			},
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: dm.CreateDMToolName,
					Arguments: map[string]interface{}{
						"user_id": "user123",
					},
				},
			},
		},
		{
			name:     "AddReaction",
			toolName: reaction.AddReactionToolName,
			handler: func(_ interface{}, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return reaction.AddReactionFn(context.Background(), req)
			},
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: reaction.AddReactionToolName,
					Arguments: map[string]interface{}{
						"post_id":    "post123",
						"emoji_name": "thumbsup",
					},
				},
			},
		},
		{
			name:     "RemoveReaction",
			toolName: reaction.RemoveReactionToolName,
			handler: func(_ interface{}, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return reaction.RemoveReactionFn(context.Background(), req)
			},
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: reaction.RemoveReactionToolName,
					Arguments: map[string]interface{}{
						"post_id":    "post123",
						"emoji_name": "thumbsup",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.handler(nil, tc.request)
			assert.NoError(t, err) // Handler should not return error, just error result
			assert.NotNil(t, result)
			assert.True(t, result.IsError, "Should return error when client not initialized")

			// Verify error message contains expected text
			errText := result.Content[0].(mcp.TextContent).Text
			assert.Contains(t, errText, "client not initialized")
		})
	}
}

func TestIntegration_ErrorPropagation_InvalidParams(t *testing.T) {
	mock := newMockMattermostServer()
	defer mock.Close()

	setupMockClient(mock)

	testCases := []struct {
		name        string
		toolName    string
		handler     func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
		request     mcp.CallToolRequest
		expectedErr string
	}{
		{
			name:     "SendMessage_MissingChannelId",
			toolName: messaging.SendMessageToolName,
			handler:  messaging.SendMessageFn,
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: messaging.SendMessageToolName,
					Arguments: map[string]interface{}{
						"message": "test message",
					},
				},
			},
			expectedErr: "channel_id",
		},
		{
			name:     "SendMessage_MissingMessage",
			toolName: messaging.SendMessageToolName,
			handler:  messaging.SendMessageFn,
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: messaging.SendMessageToolName,
					Arguments: map[string]interface{}{
						"channel_id": "channel123",
					},
				},
			},
			expectedErr: "message",
		},
		{
			name:     "EditMessage_MissingPostId",
			toolName: messaging.EditMessageToolName,
			handler:  messaging.EditMessageFn,
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: messaging.EditMessageToolName,
					Arguments: map[string]interface{}{
						"message": "updated",
					},
				},
			},
			expectedErr: "post_id",
		},
		{
			name:     "DeleteMessage_MissingPostId",
			toolName: messaging.DeleteMessageToolName,
			handler:  messaging.DeleteMessageFn,
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      messaging.DeleteMessageToolName,
					Arguments: map[string]interface{}{},
				},
			},
			expectedErr: "post_id",
		},
		{
			name:     "CreateDM_MissingUserId",
			toolName: dm.CreateDMToolName,
			handler:  dm.CreateDMFn,
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      dm.CreateDMToolName,
					Arguments: map[string]interface{}{},
				},
			},
			expectedErr: "user_id",
		},
		{
			name:     "AddReaction_MissingPostId",
			toolName: reaction.AddReactionToolName,
			handler:  reaction.AddReactionFn,
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: reaction.AddReactionToolName,
					Arguments: map[string]interface{}{
						"emoji_name": "thumbsup",
					},
				},
			},
			expectedErr: "post_id",
		},
		{
			name:     "AddReaction_MissingEmojiName",
			toolName: reaction.AddReactionToolName,
			handler:  reaction.AddReactionFn,
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: reaction.AddReactionToolName,
					Arguments: map[string]interface{}{
						"post_id": "post123",
					},
				},
			},
			expectedErr: "emoji_name",
		},
		{
			name:     "ListChannels_MissingTeamId",
			toolName: channel.ListChannelsToolName,
			handler:  channel.ListChannelsFn,
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      channel.ListChannelsToolName,
					Arguments: map[string]interface{}{},
				},
			},
			expectedErr: "team_id",
		},
		{
			name:     "SearchUsers_MissingTerm",
			toolName: user.SearchUsersToolName,
			handler:  user.SearchUsersFn,
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      user.SearchUsersToolName,
					Arguments: map[string]interface{}{},
				},
			},
			expectedErr: "term",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.handler(nil, tc.request)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.True(t, result.IsError, "Should return error for missing required parameter")

			errText := result.Content[0].(mcp.TextContent).Text
			assert.Contains(t, errText, tc.expectedErr)
		})
	}
}

// ==================== Enhanced Error Tests ====================

func TestIntegration_ErrorEnhancedError_Translation(t *testing.T) {
	testCases := []struct {
		name             string
		err              error
		context          map[string]string
		expectedCategory mmerrors.ErrorCategory
		isNotFound       bool
		isAuthError      bool
	}{
		{
			name:             "404 not found error",
			err:              fmt.Errorf("request failed with status 404: channel not found"),
			context:          map[string]string{"operation": "GetChannel", "channel_id": "abc123"},
			expectedCategory: mmerrors.CategoryUnknown,
			isNotFound:       true,
		},
		{
			name:             "401 authentication error",
			err:              fmt.Errorf("request failed with status 401: unauthorized"),
			context:          map[string]string{"operation": "GetUser"},
			expectedCategory: mmerrors.CategoryAuth,
			isAuthError:      true,
		},
		{
			name:             "403 permission denied",
			err:              fmt.Errorf("request failed with status 403: forbidden"),
			context:          map[string]string{"operation": "CreatePost"},
			expectedCategory: mmerrors.CategoryAuth,
			isAuthError:      true,
		},
		{
			name:             "500 server error",
			err:              fmt.Errorf("request failed with status 500: internal server error"),
			context:          map[string]string{"operation": "SearchUsers"},
			expectedCategory: mmerrors.CategoryUnknown,
		},
		{
			name:             "Network timeout error",
			err:              fmt.Errorf("request timeout: connection to server timed out"),
			context:          map[string]string{"operation": "SendMessage"},
			expectedCategory: mmerrors.CategoryNetwork,
		},
		{
			name:             "Connection refused error",
			err:              fmt.Errorf("connection refused to mattermost.example.com:443"),
			context:          map[string]string{"operation": "GetMe"},
			expectedCategory: mmerrors.CategoryNetwork,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			translated := mmerrors.TranslateError(tc.err, tc.context)
			require.NotNil(t, translated)

			var enhanced *mmerrors.EnhancedError
			require.True(t, errors.As(translated, &enhanced), "Should be an EnhancedError")

			assert.Equal(t, tc.expectedCategory, enhanced.Category)
			assert.Equal(t, tc.isNotFound, mmerrors.IsNotFound(translated))
			assert.Equal(t, tc.isAuthError, mmerrors.IsAuthError(translated))

			// Verify context is preserved
			for key, value := range tc.context {
				assert.Equal(t, value, enhanced.Context[key])
			}

			// Verify formatting works
			formatted := enhanced.Format()
			assert.NotEmpty(t, formatted)
			assert.Contains(t, formatted, tc.context["operation"])

			detailed := enhanced.FormatDetailed()
			assert.NotEmpty(t, detailed)
		})
	}
}

func TestIntegration_ErrorEnhancedError_NilHandling(t *testing.T) {
	// Test that nil errors are handled properly
	assert.Nil(t, mmerrors.TranslateError(nil, nil))
	assert.Nil(t, mmerrors.Wrap(nil, "operation"))

	// Test that nil error checking functions return false
	assert.False(t, mmerrors.IsNotFound(nil))
	assert.False(t, mmerrors.IsAuthError(nil))
	assert.False(t, mmerrors.IsTimeout(nil))
	assert.False(t, mmerrors.IsNetworkError(nil))
	assert.False(t, mmerrors.IsServerError(nil))
}

func TestIntegration_ErrorEnhancedError_FluentAPI(t *testing.T) {
	original := fmt.Errorf("original error")

	enhanced := mmerrors.TranslateError(original, nil).(*mmerrors.EnhancedError)

	// Test fluent API chaining
	result := enhanced.
		WithOperation("TestOperation").
		WithParam("param1", "value1").
		WithParam("param2", "value2").
		WithContext("extra", "context")

	// Should return same instance for chaining
	assert.Equal(t, enhanced, result)

	// Verify all values set
	assert.Equal(t, "TestOperation", enhanced.Operation)
	assert.Equal(t, "value1", enhanced.Context["param1"])
	assert.Equal(t, "value2", enhanced.Context["param2"])
	assert.Equal(t, "context", enhanced.Context["extra"])

	// Verify original is preserved
	assert.Equal(t, original, enhanced.Unwrap())
}

func TestIntegration_ErrorHTTPStatusCodes(t *testing.T) {
	testCases := []struct {
		name           string
		err            error
		isUnauthorized bool
		isForbidden    bool
		isNotFound     bool
		isServerError  bool
	}{
		{
			name:           "401 Unauthorized",
			err:            &testHTTPError{status: 401, message: "unauthorized"},
			isUnauthorized: true,
			isForbidden:    false,
			isNotFound:     false,
			isServerError:  false,
		},
		{
			name:           "403 Forbidden",
			err:            &testHTTPError{status: 403, message: "forbidden"},
			isUnauthorized: false,
			isForbidden:    true,
			isNotFound:     false,
			isServerError:  false,
		},
		{
			name:           "404 Not Found",
			err:            &testHTTPError{status: 404, message: "not found"},
			isUnauthorized: false,
			isForbidden:    false,
			isNotFound:     true,
			isServerError:  false,
		},
		{
			name:           "500 Internal Server Error",
			err:            &testHTTPError{status: 500, message: "internal error"},
			isUnauthorized: false,
			isForbidden:    false,
			isNotFound:     false,
			isServerError:  true,
		},
		{
			name:           "502 Bad Gateway",
			err:            &testHTTPError{status: 502, message: "bad gateway"},
			isUnauthorized: false,
			isForbidden:    false,
			isNotFound:     false,
			isServerError:  true,
		},
		{
			name:           "503 Service Unavailable",
			err:            &testHTTPError{status: 503, message: "service unavailable"},
			isUnauthorized: false,
			isForbidden:    false,
			isNotFound:     false,
			isServerError:  true,
		},
		{
			name:           "200 OK",
			err:            &testHTTPError{status: 200, message: "ok"},
			isUnauthorized: false,
			isForbidden:    false,
			isNotFound:     false,
			isServerError:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.isUnauthorized, mmerrors.IsUnauthorized(tc.err))
			assert.Equal(t, tc.isForbidden, mmerrors.IsForbidden(tc.err))
			assert.Equal(t, tc.isNotFound, mmerrors.IsNotFoundHTTP(tc.err))
			assert.Equal(t, tc.isServerError, mmerrors.IsServerError(tc.err))
		})
	}
}

// testHTTPError implements HTTPError interface for testing
type testHTTPError struct {
	status  int
	message string
}

func (e *testHTTPError) Error() string { return e.message }
func (e *testHTTPError) Status() int   { return e.status }

// ==================== Cross-Tool Error Consistency Tests ====================

func TestIntegration_CrossToolErrorConsistency(t *testing.T) {
	mock := newMockMattermostServer()
	defer mock.Close()

	// Start server but return errors to test error handling
	originalHandler := mock.server.Config.Handler
	mock.server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate various error conditions
		switch r.URL.Path {
		case "/api/v4/users/me":
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"message": "authentication failed"})
		case "/api/v4/posts":
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"message": "permission denied"})
		case "/api/v4/channels/direct":
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"message": "user not found"})
		default:
			originalHandler.ServeHTTP(w, r)
		}
	})

	setupMockClient(mock)

	testCases := []struct {
		name          string
		toolName      string
		handler       func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
		request       mcp.CallToolRequest
		expectedError bool
		errorCheck    func(error) bool
	}{
		{
			name:     "DM Creation - Auth Error",
			toolName: dm.CreateDMToolName,
			handler:  dm.CreateDMFn,
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: dm.CreateDMToolName,
					Arguments: map[string]interface{}{
						"user_id": "user123",
					},
				},
			},
			expectedError: true,
		},
		{
			name:     "Send Message - Forbidden",
			toolName: messaging.SendMessageToolName,
			handler:  messaging.SendMessageFn,
			request: mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: messaging.SendMessageToolName,
					Arguments: map[string]interface{}{
						"channel_id": "channel123",
						"message":    "test",
					},
				},
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.handler(context.Background(), tc.request)
			assert.NoError(t, err) // Handler shouldn't return error
			assert.NotNil(t, result)
			assert.True(t, result.IsError, "Should return error result")
		})
	}
}

// ==================== End-to-End Complex Workflow Tests ====================

func TestIntegration_ComplexWorkflow_MultiStepWithRollback(t *testing.T) {
	mock := newMockMattermostServer()
	defer mock.Close()

	setupMockClient(mock)

	// Complex workflow: Search user → Create DM → Send message → Add reaction → Remove reaction → Delete message

	// Step 1: Search for user
	searchReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: user.SearchUsersToolName,
			Arguments: map[string]interface{}{
				"term":  "workflowuser",
				"limit": 1,
			},
		},
	}

	searchResult, err := user.SearchUsersFn(context.Background(), searchReq)
	require.NoError(t, err)
	require.False(t, searchResult.IsError)

	var searchData map[string]interface{}
	json.Unmarshal([]byte(searchResult.Content[0].(mcp.TextContent).Text), &searchData)
	users := searchData["users"].([]interface{})
	require.GreaterOrEqual(t, len(users), 1)
	userID := users[0].(map[string]interface{})["id"].(string)

	// Step 2: Create DM
	dmReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: dm.CreateDMToolName,
			Arguments: map[string]interface{}{
				"user_id": userID,
			},
		},
	}

	dmResult, err := dm.CreateDMFn(context.Background(), dmReq)
	require.NoError(t, err)
	require.False(t, dmResult.IsError)

	var dmData map[string]interface{}
	json.Unmarshal([]byte(dmResult.Content[0].(mcp.TextContent).Text), &dmData)
	channelID := dmData["id"].(string)

	// Step 3: Send message
	msgReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: messaging.SendMessageToolName,
			Arguments: map[string]interface{}{
				"channel_id": channelID,
				"message":    "Complex workflow test",
			},
		},
	}

	msgResult, err := messaging.SendMessageFn(context.Background(), msgReq)
	require.NoError(t, err)
	require.False(t, msgResult.IsError)

	var msgData map[string]interface{}
	json.Unmarshal([]byte(msgResult.Content[0].(mcp.TextContent).Text), &msgData)
	postID := msgData["id"].(string)

	// Step 4: Add reaction
	addReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: reaction.AddReactionToolName,
			Arguments: map[string]interface{}{
				"post_id":    postID,
				"emoji_name": "heart",
			},
		},
	}

	addResult, err := reaction.AddReactionFn(context.Background(), addReq)
	require.NoError(t, err)
	require.False(t, addResult.IsError)

	// Step 5: Remove reaction
	removeReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: reaction.RemoveReactionToolName,
			Arguments: map[string]interface{}{
				"post_id":    postID,
				"emoji_name": "heart",
			},
		},
	}

	removeResult, err := reaction.RemoveReactionFn(context.Background(), removeReq)
	require.NoError(t, err)
	require.False(t, removeResult.IsError)

	var removeData map[string]interface{}
	json.Unmarshal([]byte(removeResult.Content[0].(mcp.TextContent).Text), &removeData)
	assert.True(t, removeData["success"].(bool))
	assert.Equal(t, postID, removeData["post_id"])

	// Step 6: Delete message (cleanup)
	deleteReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: messaging.DeleteMessageToolName,
			Arguments: map[string]interface{}{
				"post_id": postID,
			},
		},
	}

	deleteResult, err := messaging.DeleteMessageFn(context.Background(), deleteReq)
	require.NoError(t, err)
	require.False(t, deleteResult.IsError)

	var deleteData map[string]interface{}
	json.Unmarshal([]byte(deleteResult.Content[0].(mcp.TextContent).Text), &deleteData)
	assert.True(t, deleteData["success"].(bool))
}

func TestIntegration_WorkflowErrorChaining_FirstFailureStops(t *testing.T) {
	mock := newMockMattermostServer()
	defer mock.Close()

	// Setup mock to fail on DM creation
	mock.server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/users/me":
			user := &model.User{
				Id:        "user-current",
				Username:  "currentuser",
				Email:     "current@example.com",
				FirstName: "Current",
				LastName:  "User",
				Roles:     "system_user",
			}
			json.NewEncoder(w).Encode(user)
		case "/api/v4/channels/direct":
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"message": "cannot create DM with yourself"})
		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"message": "not found"})
		}
	})

	setupMockClient(mock)

	// Try to create DM with ourselves (which will fail)
	dmReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: dm.CreateDMToolName,
			Arguments: map[string]interface{}{
				"user_id": "user-current", // Same as current user
			},
		},
	}

	// This should fail
	result, err := dm.CreateDMFn(context.Background(), dmReq)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.IsError, "DM creation should fail")

	// Verify the error message is descriptive
	errText := result.Content[0].(mcp.TextContent).Text
	assert.NotEmpty(t, errText)
}

func TestDebugChannelList(t *testing.T) {
	mock := newMockMattermostServer()
	defer mock.Close()

	setupMockClient(mock)

	listReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: channel.ListChannelsToolName,
			Arguments: map[string]interface{}{
				"team_id": "team-1",
				"limit":   10,
			},
		},
	}

	listResult, err := channel.ListChannelsFn(context.Background(), listReq)
	require.NoError(t, err)
	require.NotNil(t, listResult)

	if listResult.IsError {
		t.Fatalf("Error result: %s", listResult.Content[0].(mcp.TextContent).Text)
	}
}
