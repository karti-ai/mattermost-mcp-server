package file

import "github.com/mattermost/mattermost-server/v6/model"

func SlimFileInfo(f *model.FileInfo) map[string]interface{} {
	if f == nil {
		return nil
	}
	return map[string]interface{}{
		"id":         f.Id,
		"name":       f.Name,
		"extension":  f.Extension,
		"size":       f.Size,
		"mime_type":  f.MimeType,
		"channel_id": f.ChannelId,
		"create_at":  f.CreateAt,
	}
}

func SlimFileInfos(infos []*model.FileInfo) []map[string]interface{} {
	if infos == nil {
		return nil
	}
	result := make([]map[string]interface{}, 0, len(infos))
	for _, info := range infos {
		if slim := SlimFileInfo(info); slim != nil {
			result = append(result, slim)
		}
	}
	return result
}
