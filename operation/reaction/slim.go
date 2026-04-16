package reaction

import "github.com/mattermost/mattermost-server/v6/model"

func SlimReaction(r *model.Reaction) map[string]interface{} {
	if r == nil {
		return nil
	}
	return map[string]interface{}{
		"user_id":    r.UserId,
		"post_id":    r.PostId,
		"emoji_name": r.EmojiName,
		"create_at":  r.CreateAt,
	}
}
