package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/karti-ai/mattermost-mcp-server/pkg/file"
	"github.com/karti-ai/mattermost-mcp-server/pkg/log"
	"github.com/karti-ai/mattermost-mcp-server/pkg/mattermost"
	"github.com/karti-ai/mattermost-mcp-server/pkg/params"
	"github.com/karti-ai/mattermost-mcp-server/pkg/to"
	"github.com/karti-ai/mattermost-mcp-server/pkg/tool"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mattermost/mattermost-server/v6/model"
)

var Tool = tool.New()

const (
	UploadFileToolName   = "mattermost_upload_file"
	DownloadFileToolName = "mattermost_download_file"
)

var (
	UploadFileTool = mcp.NewTool(
		UploadFileToolName,
		mcp.WithDescription("Upload file to channel"),
		mcp.WithString("channel_id", mcp.Required(), mcp.Description("Channel ID to upload file to")),
		mcp.WithString("file_path", mcp.Required(), mcp.Description("Local file path to upload")),
		mcp.WithString("message", mcp.Description("Message to post with file (optional)")),
	)

	DownloadFileTool = mcp.NewTool(
		DownloadFileToolName,
		mcp.WithDescription("Download file from Mattermost"),
		mcp.WithString("file_id", mcp.Required(), mcp.Description("File ID to download")),
		mcp.WithString("download_path", mcp.Required(), mcp.Description("Local path where file should be saved")),
	)
)

func init() {
	registerTools()
}

func registerTools() {
	tools := []server.ServerTool{
		{Tool: UploadFileTool, Handler: UploadFileFn},
		{Tool: DownloadFileTool, Handler: DownloadFileFn},
	}
	for _, t := range tools {
		Tool.RegisterWrite(t)
	}
}

func UploadFileFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[File] Called UploadFileFn")

	args := req.GetArguments()

	channelId, err := params.GetString(args, "channel_id")
	if err != nil {
		return to.Error(fmt.Errorf("[channel_id] %v", err)), nil
	}

	filePath, err := params.GetString(args, "file_path")
	if err != nil {
		return to.Error(fmt.Errorf("[file_path] %v", err)), nil
	}

	message := params.GetOptionalString(args, "message", "")

	if !file.IsValidPath(filePath) {
		return to.Error(fmt.Errorf("[file_path] path traversal detected: %s", filePath)), nil
	}

	fileInfo, err := file.GetFileInfo(filePath)
	if err != nil {
		return to.Error(fmt.Errorf("[file_path] failed to access file: %v", err)), nil
	}

	if fileInfo.IsDir() {
		return to.Error(fmt.Errorf("[file_path] path is a directory, not a file: %s", filePath)), nil
	}

	if err := file.ValidateFileSize(fileInfo.Size()); err != nil {
		return to.Error(fmt.Errorf("[file_path] %v", err)), nil
	}

	if err := file.ValidateFilename(fileInfo.Name()); err != nil {
		return to.Error(fmt.Errorf("[file_path] %v", err)), nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return to.Error(fmt.Errorf("[file_path] failed to read file: %v", err)), nil
	}

	if _, err := file.ValidateMimeType(data); err != nil {
		return to.Error(fmt.Errorf("[file_path] %v", err)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	filename := filepath.Base(filePath)
	uploadResp, err := client.UploadFile(ctx, data, channelId, filename)
	if err != nil {
		return to.Error(fmt.Errorf("[upload] failed to upload file: %v", err)), nil
	}

	if message != "" && len(uploadResp.FileInfos) > 0 {
		post := &model.Post{
			ChannelId: channelId,
			Message:   message,
			FileIds:   []string{uploadResp.FileInfos[0].Id},
		}
		_, err := client.CreatePost(ctx, post)
		if err != nil {
			return to.Result(map[string]interface{}{
				"success":    true,
				"file_id":    uploadResp.FileInfos[0].Id,
				"file_infos": SlimFileInfos(uploadResp.FileInfos),
				"warning":    fmt.Sprintf("File uploaded but message failed to post: %v", err),
			}), nil
		}
	}

	return to.Result(map[string]interface{}{
		"success":    true,
		"file_id":    uploadResp.FileInfos[0].Id,
		"file_infos": SlimFileInfos(uploadResp.FileInfos),
	}), nil
}

func DownloadFileFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Debugf("[File] Called DownloadFileFn")

	args := req.GetArguments()

	fileId, err := params.GetString(args, "file_id")
	if err != nil {
		return to.Error(fmt.Errorf("[file_id] %v", err)), nil
	}

	downloadPath, err := params.GetString(args, "download_path")
	if err != nil {
		return to.Error(fmt.Errorf("[download_path] %v", err)), nil
	}

	if !file.IsValidPath(downloadPath) {
		return to.Error(fmt.Errorf("[download_path] path traversal detected: %s", downloadPath)), nil
	}

	if file.Exists(downloadPath) {
		return to.Error(fmt.Errorf("[download_path] file already exists: %s", downloadPath)), nil
	}

	client := mattermost.GetGlobalClient()
	if client == nil {
		return to.Error(fmt.Errorf("[internal] client not initialized")), nil
	}

	data, _, err := client.GetFile(ctx, fileId)
	if err != nil {
		return to.Error(fmt.Errorf("[download] failed to download file: %v", err)), nil
	}

	if int64(len(data)) > file.MaxFileSize {
		return to.Error(fmt.Errorf("[download] downloaded file size %d exceeds maximum allowed %d", len(data), file.MaxFileSize)), nil
	}

	if err := file.CheckDiskSpace(downloadPath, int64(len(data))); err != nil {
		return to.Error(fmt.Errorf("[download_path] %v", err)), nil
	}

	if err := os.WriteFile(downloadPath, data, 0644); err != nil {
		return to.Error(fmt.Errorf("[download_path] failed to save file: %v", err)), nil
	}

	return to.Result(map[string]interface{}{
		"success":       true,
		"file_id":       fileId,
		"download_path": downloadPath,
		"size":          len(data),
	}), nil
}
