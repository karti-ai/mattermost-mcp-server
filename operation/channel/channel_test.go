package channel

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
		Name:        "general",
		DisplayName: "General",
		Type:        "O",
		TeamId:      "team456",
	}

	slim := SlimChannel(c)
	assert.NotNil(t, slim)
	assert.Equal(t, "channel123", slim["id"])
	assert.Equal(t, "general", slim["name"])
	assert.Equal(t, "General", slim["display_name"])
	assert.Equal(t, "O", slim["type"])
	assert.Equal(t, "team456", slim["team_id"])
}

func TestSlimChannel_Nil(t *testing.T) {
	slim := SlimChannel(nil)
	assert.Nil(t, slim)
}

func TestSlimChannel_DifferentTypes(t *testing.T) {
	tests := []struct {
		name     string
		chType   model.ChannelType
		expected string
	}{
		{"Open channel", model.ChannelTypeOpen, "O"},
		{"Private channel", model.ChannelTypePrivate, "P"},
		{"Direct message", model.ChannelTypeDirect, "D"},
		{"Group message", model.ChannelTypeGroup, "G"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &model.Channel{
				Id:          "channel123",
				Name:        "test-channel",
				DisplayName: "Test Channel",
				Type:        tt.chType,
				TeamId:      "team456",
			}
			slim := SlimChannel(c)
			assert.Equal(t, tt.expected, slim["type"])
		})
	}
}

func TestToolRegistration(t *testing.T) {
	tools := Tool.Tools()
	assert.Len(t, tools, 5)

	toolNames := make(map[string]bool)
	for _, t := range tools {
		toolNames[t.Tool.Name] = true
	}

	assert.True(t, toolNames[ListChannelsToolName], "ListChannels tool should be registered")
	assert.True(t, toolNames[GetChannelByNameToolName], "GetChannelByName tool should be registered")
	assert.True(t, toolNames[GetChannelInfoToolName], "GetChannelInfo tool should be registered")
	assert.True(t, toolNames[GetUnreadCountToolName], "GetUnreadCount tool should be registered")
	assert.True(t, toolNames[MarkChannelReadToolName], "MarkChannelRead tool should be registered")
}

func TestListChannelsFn_MissingTeamId(t *testing.T) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      ListChannelsToolName,
			Arguments: map[string]interface{}{},
		},
	}

	result, err := ListChannelsFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestListChannelsFn_ClientNotInitialized(t *testing.T) {
	mattermost.SetGlobalClient(nil)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: ListChannelsToolName,
			Arguments: map[string]interface{}{
				"team_id": "team123",
			},
		},
	}

	result, err := ListChannelsFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}
