package reaction

import (
	"testing"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/stretchr/testify/assert"
)

func TestSlimReaction(t *testing.T) {
	tests := []struct {
		name     string
		reaction *model.Reaction
		expected map[string]interface{}
	}{
		{
			name: "valid reaction",
			reaction: &model.Reaction{
				UserId:    "user123",
				PostId:    "post456",
				EmojiName: "thumbsup",
				CreateAt:  1234567890,
			},
			expected: map[string]interface{}{
				"user_id":    "user123",
				"post_id":    "post456",
				"emoji_name": "thumbsup",
				"create_at":  int64(1234567890),
			},
		},
		{
			name:     "nil reaction",
			reaction: nil,
			expected: nil,
		},
		{
			name: "reaction with different emoji",
			reaction: &model.Reaction{
				UserId:    "user789",
				PostId:    "post012",
				EmojiName: "rocket",
				CreateAt:  9876543210,
			},
			expected: map[string]interface{}{
				"user_id":    "user789",
				"post_id":    "post012",
				"emoji_name": "rocket",
				"create_at":  int64(9876543210),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SlimReaction(tt.reaction)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToolConstants(t *testing.T) {
	assert.Equal(t, "mattermost_add_reaction", AddReactionToolName)
	assert.Equal(t, "mattermost_remove_reaction", RemoveReactionToolName)
}

func TestToolRegistration(t *testing.T) {
	// Verify tools are registered
	tools := Tool.Tools()
	assert.Len(t, tools, 2)

	toolNames := make([]string, len(tools))
	for i, t := range tools {
		toolNames[i] = t.Tool.Name
	}

	assert.Contains(t, toolNames, AddReactionToolName)
	assert.Contains(t, toolNames, RemoveReactionToolName)
}
