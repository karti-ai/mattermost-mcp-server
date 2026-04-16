package user

import "github.com/mattermost/mattermost-server/v6/model"

func SlimUser(u *model.User) map[string]interface{} {
	if u == nil {
		return nil
	}
	return map[string]interface{}{
		"id":         u.Id,
		"username":   u.Username,
		"email":      u.Email,
		"first_name": u.FirstName,
		"last_name":  u.LastName,
		"roles":      u.Roles,
	}
}
