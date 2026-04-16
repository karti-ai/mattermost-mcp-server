package team

import "github.com/mattermost/mattermost-server/v6/model"

func SlimTeam(t *model.Team) map[string]interface{} {
	if t == nil {
		return nil
	}
	return map[string]interface{}{
		"id":           t.Id,
		"name":         t.Name,
		"display_name": t.DisplayName,
		"description":  t.Description,
	}
}
