package messaging

import "github.com/mattermost/mattermost-server/v6/model"

func SlimPost(p *model.Post) map[string]interface{} {
	if p == nil {
		return nil
	}
	return map[string]interface{}{
		"id":         p.Id,
		"channel_id": p.ChannelId,
		"user_id":    p.UserId,
		"message":    p.Message,
		"create_at":  p.CreateAt,
		"update_at":  p.UpdateAt,
	}
}
