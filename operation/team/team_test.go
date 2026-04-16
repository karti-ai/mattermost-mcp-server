package team

import (
	"testing"

	"github.com/karti-ai/mattermost-mcp-server/pkg/mattermost"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/stretchr/testify/assert"
)

func TestSlimTeam(t *testing.T) {
	team := &model.Team{
		Id:          "team123",
		Name:        "my-team",
		DisplayName: "My Team",
		Description: "A test team for unit testing",
	}

	slim := SlimTeam(team)
	assert.NotNil(t, slim)
	assert.Equal(t, "team123", slim["id"])
	assert.Equal(t, "my-team", slim["name"])
	assert.Equal(t, "My Team", slim["display_name"])
	assert.Equal(t, "A test team for unit testing", slim["description"])
}

func TestSlimTeam_Nil(t *testing.T) {
	slim := SlimTeam(nil)
	assert.Nil(t, slim)
}

func TestToolRegistration(t *testing.T) {
	tools := Tool.Tools()
	assert.Len(t, tools, 1)

	toolNames := make(map[string]bool)
	for _, t := range tools {
		toolNames[t.Tool.Name] = true
	}

	assert.True(t, toolNames[ListTeamsToolName], "ListTeams tool should be registered")
}

func TestListTeamsFn_ClientNotInitialized(t *testing.T) {
	mattermost.SetGlobalClient(nil)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      ListTeamsToolName,
			Arguments: map[string]interface{}{},
		},
	}

	result, err := ListTeamsFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}
