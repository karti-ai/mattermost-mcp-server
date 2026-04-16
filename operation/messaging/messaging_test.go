package messaging

import (
	"testing"

	"github.com/karti-ai/mattermost-mcp-server/pkg/mattermost"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/stretchr/testify/assert"
)

func TestSlimPost(t *testing.T) {
	p := &model.Post{
		Id:        "abc123",
		ChannelId: "channel456",
		UserId:    "user789",
		Message:   "Hello World",
		CreateAt:  1234567890000,
		UpdateAt:  1234567890001,
	}

	slim := SlimPost(p)
	assert.NotNil(t, slim)
	assert.Equal(t, "abc123", slim["id"])
	assert.Equal(t, "channel456", slim["channel_id"])
	assert.Equal(t, "user789", slim["user_id"])
	assert.Equal(t, "Hello World", slim["message"])
	assert.Equal(t, int64(1234567890000), slim["create_at"])
	assert.Equal(t, int64(1234567890001), slim["update_at"])
}

func TestSlimPost_Nil(t *testing.T) {
	slim := SlimPost(nil)
	assert.Nil(t, slim)
}

func TestToolRegistration(t *testing.T) {
	writeTools := Tool.WriteTools()
	assert.Len(t, writeTools, 3)

	readTools := Tool.ReadTools()
	assert.Len(t, readTools, 3)

	toolNames := make(map[string]bool)
	for _, t := range writeTools {
		toolNames[t.Tool.Name] = true
	}
	for _, t := range readTools {
		toolNames[t.Tool.Name] = true
	}

	assert.True(t, toolNames[SendMessageToolName], "SendMessage tool should be registered")
	assert.True(t, toolNames[EditMessageToolName], "EditMessage tool should be registered")
	assert.True(t, toolNames[DeleteMessageToolName], "DeleteMessage tool should be registered")
	assert.True(t, toolNames[GetChannelMessagesToolName], "GetChannelMessages tool should be registered")
	assert.True(t, toolNames[GetThreadToolName], "GetThread tool should be registered")
	assert.True(t, toolNames[SearchPostsToolName], "SearchPosts tool should be registered")
}

func TestSendMessageFn_ClientNotInitialized(t *testing.T) {
	mattermost.SetGlobalClient(nil)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: SendMessageToolName,
			Arguments: map[string]interface{}{
				"channel_id": "channel123",
				"message":    "Test message",
			},
		},
	}

	result, err := SendMessageFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestEditMessageFn_ClientNotInitialized(t *testing.T) {
	mattermost.SetGlobalClient(nil)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: EditMessageToolName,
			Arguments: map[string]interface{}{
				"post_id": "post123",
				"message": "Updated message",
			},
		},
	}

	result, err := EditMessageFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestDeleteMessageFn_ClientNotInitialized(t *testing.T) {
	mattermost.SetGlobalClient(nil)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: DeleteMessageToolName,
			Arguments: map[string]interface{}{
				"post_id": "post123",
			},
		},
	}

	result, err := DeleteMessageFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestSendMessageFn_MissingChannelId(t *testing.T) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: SendMessageToolName,
			Arguments: map[string]interface{}{
				"message": "Test message",
			},
		},
	}

	result, err := SendMessageFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestSendMessageFn_MissingMessage(t *testing.T) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: SendMessageToolName,
			Arguments: map[string]interface{}{
				"channel_id": "channel123",
			},
		},
	}

	result, err := SendMessageFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestEditMessageFn_MissingPostId(t *testing.T) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: EditMessageToolName,
			Arguments: map[string]interface{}{
				"message": "Updated message",
			},
		},
	}

	result, err := EditMessageFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestEditMessageFn_MissingMessage(t *testing.T) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: EditMessageToolName,
			Arguments: map[string]interface{}{
				"post_id": "post123",
			},
		},
	}

	result, err := EditMessageFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestDeleteMessageFn_MissingPostId(t *testing.T) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      DeleteMessageToolName,
			Arguments: map[string]interface{}{},
		},
	}

	result, err := DeleteMessageFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}
