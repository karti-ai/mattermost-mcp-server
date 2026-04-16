package dm

import (
	"testing"

	"github.com/karti-ai/mattermost-mcp-server/pkg/mattermost"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/stretchr/testify/assert"
)

func TestSlimChannel(t *testing.T) {
	c := &model.Channel{
		Id:          "channel123",
		Name:        "user1__user2",
		DisplayName: "user1, user2",
		Type:        model.ChannelTypeDirect,
		TeamId:      "",
	}

	slim := SlimChannel(c)
	assert.NotNil(t, slim)
	assert.Equal(t, "channel123", slim["id"])
	assert.Equal(t, "user1__user2", slim["name"])
	assert.Equal(t, "user1, user2", slim["display_name"])
	assert.Equal(t, model.ChannelTypeDirect, slim["type"])
	assert.Equal(t, "", slim["team_id"])
}

func TestSlimChannel_Nil(t *testing.T) {
	slim := SlimChannel(nil)
	assert.Nil(t, slim)
}

func TestToolRegistration(t *testing.T) {
	tools := Tool.Tools()
	assert.Len(t, tools, 1)

	toolNames := make(map[string]bool)
	for _, t := range tools {
		toolNames[t.Tool.Name] = true
	}

	assert.True(t, toolNames[CreateDMToolName], "CreateDM tool should be registered")
}

func TestCreateDMFn_ClientNotInitialized(t *testing.T) {
	mattermost.SetGlobalClient(nil)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: CreateDMToolName,
			Arguments: map[string]interface{}{
				"user_id": "user123",
			},
		},
	}

	result, err := CreateDMFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestCreateDMFn_MissingUserId(t *testing.T) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      CreateDMToolName,
			Arguments: map[string]interface{}{},
		},
	}

	result, err := CreateDMFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}
