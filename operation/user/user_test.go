package user

import (
	"testing"

	"github.com/karti-ai/mattermost-mcp-server/pkg/mattermost"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/stretchr/testify/assert"
)

func TestSlimUser(t *testing.T) {
	u := &model.User{
		Id:        "user123",
		Username:  "johndoe",
		Email:     "john@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Roles:     "system_user",
	}

	slim := SlimUser(u)
	assert.NotNil(t, slim)
	assert.Equal(t, "user123", slim["id"])
	assert.Equal(t, "johndoe", slim["username"])
	assert.Equal(t, "john@example.com", slim["email"])
	assert.Equal(t, "John", slim["first_name"])
	assert.Equal(t, "Doe", slim["last_name"])
	assert.Equal(t, "system_user", slim["roles"])
}

func TestSlimUser_Nil(t *testing.T) {
	slim := SlimUser(nil)
	assert.Nil(t, slim)
}

func TestToolRegistration(t *testing.T) {
	tools := Tool.Tools()
	assert.Len(t, tools, 2)

	toolNames := make(map[string]bool)
	for _, t := range tools {
		toolNames[t.Tool.Name] = true
	}

	assert.True(t, toolNames[SearchUsersToolName], "SearchUsers tool should be registered")
	assert.True(t, toolNames[GetUserStatusToolName], "GetUserStatus tool should be registered")
}

func TestSearchUsersFn_ClientNotInitialized(t *testing.T) {
	mattermost.SetGlobalClient(nil)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: SearchUsersToolName,
			Arguments: map[string]interface{}{
				"term": "john",
			},
		},
	}

	result, err := SearchUsersFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestSearchUsersFn_MissingTerm(t *testing.T) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      SearchUsersToolName,
			Arguments: map[string]interface{}{},
		},
	}

	result, err := SearchUsersFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}
