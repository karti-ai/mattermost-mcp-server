package dm

import "github.com/mattermost/mattermost-server/v6/model"

func SlimChannel(c *model.Channel) map[string]interface{} {
	if c == nil {
		return nil
	}
	return map[string]interface{}{
		"id":           c.Id,
		"name":         c.Name,
		"display_name": c.DisplayName,
		"type":         c.Type,
		"team_id":      c.TeamId,
	}
}
