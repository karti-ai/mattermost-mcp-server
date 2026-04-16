package channel

import "github.com/mattermost/mattermost-server/v6/model"

func SlimChannel(c *model.Channel) map[string]interface{} {
	if c == nil {
		return nil
	}
	return map[string]interface{}{
		"id":           c.Id,
		"name":         c.Name,
		"display_name": c.DisplayName,
		"type":         string(c.Type),
		"team_id":      c.TeamId,
	}
}

func DetailedChannel(c *model.Channel, memberCount int64) map[string]interface{} {
	if c == nil {
		return nil
	}
	return map[string]interface{}{
		"id":           c.Id,
		"name":         c.Name,
		"display_name": c.DisplayName,
		"type":         string(c.Type),
		"team_id":      c.TeamId,
		"purpose":      c.Purpose,
		"header":       c.Header,
		"creator_id":   c.CreatorId,
		"create_at":    c.CreateAt,
		"member_count": memberCount,
	}
}

func SlimChannelMember(m model.ChannelMember) map[string]interface{} {
	return map[string]interface{}{
		"user_id":       m.UserId,
		"channel_id":    m.ChannelId,
		"roles":         m.Roles,
		"last_viewed":   m.LastViewedAt,
		"msg_count":     m.MsgCount,
		"mention_count": m.MentionCount,
	}
}
