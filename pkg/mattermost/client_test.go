package mattermost

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	client := NewClient("https://test.mattermost.com", "bot-token", "pat-token")
	assert.NotNil(t, client)
	assert.Equal(t, "https://test.mattermost.com", client.host)
	assert.Equal(t, "bot-token", client.botToken)
	assert.Equal(t, "pat-token", client.pat)
}

func TestGetToken_ReadOperation(t *testing.T) {
	client := NewClient("https://test.mattermost.com", "bot-token", "pat-token")
	token := client.getToken(false)
	assert.Equal(t, "bot-token", token)
}

func TestGetToken_WriteOperation(t *testing.T) {
	client := NewClient("https://test.mattermost.com", "bot-token", "pat-token")
	token := client.getToken(true)
	assert.Equal(t, "pat-token", token)
}

func TestGetToken_FallbackToPAT(t *testing.T) {
	client := NewClient("https://test.mattermost.com", "", "pat-token")
	token := client.getToken(false)
	assert.Equal(t, "pat-token", token)
}

func TestSetGlobalClient(t *testing.T) {
	client := NewClient("https://test.mattermost.com", "bot-token", "pat-token")
	SetGlobalClient(client)
	assert.Equal(t, client, GetGlobalClient())
}
