package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/karti-ai/mattermost-mcp-server/pkg/file"
	"github.com/karti-ai/mattermost-mcp-server/pkg/mattermost"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/stretchr/testify/assert"
)

func TestSlimFileInfo(t *testing.T) {
	f := &model.FileInfo{
		Id:        "file123",
		Name:      "test.pdf",
		Extension: "pdf",
		Size:      1024,
		MimeType:  "application/pdf",
		ChannelId: "channel456",
		CreateAt:  1234567890000,
	}

	slim := SlimFileInfo(f)
	assert.NotNil(t, slim)
	assert.Equal(t, "file123", slim["id"])
	assert.Equal(t, "test.pdf", slim["name"])
	assert.Equal(t, "pdf", slim["extension"])
	assert.Equal(t, int64(1024), slim["size"])
	assert.Equal(t, "application/pdf", slim["mime_type"])
	assert.Equal(t, "channel456", slim["channel_id"])
	assert.Equal(t, int64(1234567890000), slim["create_at"])
}

func TestSlimFileInfo_Nil(t *testing.T) {
	slim := SlimFileInfo(nil)
	assert.Nil(t, slim)
}

func TestSlimFileInfos(t *testing.T) {
	infos := []*model.FileInfo{
		{
			Id:        "file1",
			Name:      "test1.pdf",
			Extension: "pdf",
			Size:      1024,
			MimeType:  "application/pdf",
			ChannelId: "channel1",
			CreateAt:  1234567890000,
		},
		{
			Id:        "file2",
			Name:      "test2.png",
			Extension: "png",
			Size:      2048,
			MimeType:  "image/png",
			ChannelId: "channel2",
			CreateAt:  1234567890001,
		},
	}

	slim := SlimFileInfos(infos)
	assert.NotNil(t, slim)
	assert.Len(t, slim, 2)
	assert.Equal(t, "file1", slim[0]["id"])
	assert.Equal(t, "file2", slim[1]["id"])
}

func TestSlimFileInfos_Nil(t *testing.T) {
	slim := SlimFileInfos(nil)
	assert.Nil(t, slim)
}

func TestSlimFileInfos_WithNilItem(t *testing.T) {
	infos := []*model.FileInfo{
		{
			Id:        "file1",
			Name:      "test1.pdf",
			Extension: "pdf",
			Size:      1024,
			MimeType:  "application/pdf",
			ChannelId: "channel1",
			CreateAt:  1234567890000,
		},
		nil,
		{
			Id:        "file2",
			Name:      "test2.png",
			Extension: "png",
			Size:      2048,
			MimeType:  "image/png",
			ChannelId: "channel2",
			CreateAt:  1234567890001,
		},
	}

	slim := SlimFileInfos(infos)
	assert.NotNil(t, slim)
	assert.Len(t, slim, 2)
}

func TestToolRegistration(t *testing.T) {
	tools := Tool.Tools()
	assert.Len(t, tools, 2)

	toolNames := make(map[string]bool)
	for _, t := range tools {
		toolNames[t.Tool.Name] = true
	}

	assert.True(t, toolNames[UploadFileToolName], "UploadFile tool should be registered")
	assert.True(t, toolNames[DownloadFileToolName], "DownloadFile tool should be registered")
}

func TestUploadFileFn_ClientNotInitialized(t *testing.T) {
	mattermost.SetGlobalClient(nil)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: UploadFileToolName,
			Arguments: map[string]interface{}{
				"channel_id": "channel123",
				"file_path":  "/tmp/test.txt",
			},
		},
	}

	result, err := UploadFileFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestUploadFileFn_MissingChannelId(t *testing.T) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: UploadFileToolName,
			Arguments: map[string]interface{}{
				"file_path": "/tmp/test.txt",
			},
		},
	}

	result, err := UploadFileFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestUploadFileFn_MissingFilePath(t *testing.T) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: UploadFileToolName,
			Arguments: map[string]interface{}{
				"channel_id": "channel123",
			},
		},
	}

	result, err := UploadFileFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestUploadFileFn_PathTraversal(t *testing.T) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: UploadFileToolName,
			Arguments: map[string]interface{}{
				"channel_id": "channel123",
				"file_path":  "../../../etc/passwd",
			},
		},
	}

	result, err := UploadFileFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "path traversal")
}

func TestUploadFileFn_AbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	assert.NoError(t, err)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: UploadFileToolName,
			Arguments: map[string]interface{}{
				"channel_id": "channel123",
				"file_path":  testFile,
			},
		},
	}

	result, err := UploadFileFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "client not initialized")
}

func TestUploadFileFn_DangerousExtension(t *testing.T) {
	tmpDir := t.TempDir()
	dangerousFile := filepath.Join(tmpDir, "malicious.sh")
	err := os.WriteFile(dangerousFile, []byte("#!/bin/bash\necho 'pwned'"), 0644)
	assert.NoError(t, err)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: UploadFileToolName,
			Arguments: map[string]interface{}{
				"channel_id": "channel123",
				"file_path":  dangerousFile,
			},
		},
	}

	result, err := UploadFileFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "not allowed")
}

func TestUploadFileFn_NonExistentFile(t *testing.T) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: UploadFileToolName,
			Arguments: map[string]interface{}{
				"channel_id": "channel123",
				"file_path":  "/tmp/nonexistent_file_12345.txt",
			},
		},
	}

	result, err := UploadFileFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "failed to access")
}

func TestUploadFileFn_DirectoryInsteadOfFile(t *testing.T) {
	tmpDir := t.TempDir()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: UploadFileToolName,
			Arguments: map[string]interface{}{
				"channel_id": "channel123",
				"file_path":  tmpDir,
			},
		},
	}

	result, err := UploadFileFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "directory")
}

func TestDownloadFileFn_ClientNotInitialized(t *testing.T) {
	mattermost.SetGlobalClient(nil)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: DownloadFileToolName,
			Arguments: map[string]interface{}{
				"file_id":       "file123",
				"download_path": "/tmp/downloaded.txt",
			},
		},
	}

	result, err := DownloadFileFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestDownloadFileFn_MissingFileId(t *testing.T) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: DownloadFileToolName,
			Arguments: map[string]interface{}{
				"download_path": "/tmp/downloaded.txt",
			},
		},
	}

	result, err := DownloadFileFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestDownloadFileFn_MissingDownloadPath(t *testing.T) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: DownloadFileToolName,
			Arguments: map[string]interface{}{
				"file_id": "file123",
			},
		},
	}

	result, err := DownloadFileFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestDownloadFileFn_PathTraversal(t *testing.T) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: DownloadFileToolName,
			Arguments: map[string]interface{}{
				"file_id":       "file123",
				"download_path": "../../../etc/passwd",
			},
		},
	}

	result, err := DownloadFileFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "path traversal")
}

func TestDownloadFileFn_FileAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "exists.txt")
	err := os.WriteFile(existingFile, []byte("existing content"), 0644)
	assert.NoError(t, err)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: DownloadFileToolName,
			Arguments: map[string]interface{}{
				"file_id":       "file123",
				"download_path": existingFile,
			},
		},
	}

	result, err := DownloadFileFn(nil, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "already exists")
}

func TestFileSecurityValidation(t *testing.T) {
	t.Run("IsValidPath allows safe relative paths", func(t *testing.T) {
		assert.True(t, file.IsValidPath("document.pdf"))
		assert.True(t, file.IsValidPath("subdir/file.png"))
		assert.True(t, file.IsValidPath("./file.txt"))
	})

	t.Run("IsValidPath blocks path traversal", func(t *testing.T) {
		assert.False(t, file.IsValidPath("../file.txt"))
		assert.False(t, file.IsValidPath("../../etc/passwd"))
		assert.False(t, file.IsValidPath("subdir/../../../etc/passwd"))
	})

	t.Run("IsValidPath allows absolute paths without traversal", func(t *testing.T) {
		assert.True(t, file.IsValidPath("/etc/passwd"))
		assert.True(t, file.IsValidPath("/tmp/file.txt"))
	})

	t.Run("IsDangerousExtension blocks dangerous types", func(t *testing.T) {
		assert.True(t, file.IsDangerousExtension("file.exe"))
		assert.True(t, file.IsDangerousExtension("script.sh"))
		assert.True(t, file.IsDangerousExtension("run.bat"))
		assert.True(t, file.IsDangerousExtension("malicious.js"))
	})

	t.Run("IsDangerousExtension allows safe types", func(t *testing.T) {
		assert.False(t, file.IsDangerousExtension("document.pdf"))
		assert.False(t, file.IsDangerousExtension("image.png"))
		assert.False(t, file.IsDangerousExtension("notes.txt"))
	})

	t.Run("ValidateFileSize blocks oversized files", func(t *testing.T) {
		err := file.ValidateFileSize(100 * 1024 * 1024) // 100MB
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum")
	})

	t.Run("ValidateFileSize allows files under limit", func(t *testing.T) {
		err := file.ValidateFileSize(10 * 1024 * 1024) // 10MB
		assert.NoError(t, err)
	})

	t.Run("IsAllowedMimeType allows whitelist types", func(t *testing.T) {
		assert.True(t, file.IsAllowedMimeType("image/jpeg"))
		assert.True(t, file.IsAllowedMimeType("image/png"))
		assert.True(t, file.IsAllowedMimeType("application/pdf"))
		assert.True(t, file.IsAllowedMimeType("text/plain"))
	})

	t.Run("IsAllowedMimeType blocks non-whitelist types", func(t *testing.T) {
		assert.False(t, file.IsAllowedMimeType("application/x-executable"))
		assert.False(t, file.IsAllowedMimeType("application/x-sh"))
		assert.False(t, file.IsAllowedMimeType("text/html"))
	})

	t.Run("DetectMimeType detects file types correctly", func(t *testing.T) {
		pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
		assert.Equal(t, "image/png", file.DetectMimeType(pngData))

		textData := []byte("Hello, World!")
		assert.Equal(t, "text/plain; charset=utf-8", file.DetectMimeType(textData))
	})
}

func TestSanitizePath(t *testing.T) {
	t.Run("SanitizePath returns clean path for valid input", func(t *testing.T) {
		path, err := file.SanitizePath("subdir//file.txt")
		assert.NoError(t, err)
		assert.Equal(t, "subdir/file.txt", path)
	})

	t.Run("SanitizePath rejects traversal attempts", func(t *testing.T) {
		_, err := file.SanitizePath("../file.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "path traversal")
	})

	t.Run("SanitizePath allows absolute paths without traversal", func(t *testing.T) {
		path, err := file.SanitizePath("/etc/passwd")
		assert.NoError(t, err)
		assert.Equal(t, "/etc/passwd", path)
	})
}

func TestValidateFilename(t *testing.T) {
	t.Run("ValidateFilename rejects empty filename", func(t *testing.T) {
		err := file.ValidateFilename("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("ValidateFilename rejects dangerous extensions", func(t *testing.T) {
		err := file.ValidateFilename("malicious.exe")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not allowed")
	})

	t.Run("ValidateFilename accepts safe filenames", func(t *testing.T) {
		err := file.ValidateFilename("document.pdf")
		assert.NoError(t, err)
	})

	t.Run("ValidateFilename rejects null bytes", func(t *testing.T) {
		err := file.ValidateFilename("file\x00.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid characters")
	})
}

func TestCheckDiskSpace(t *testing.T) {
	t.Run("CheckDiskSpace succeeds for writable directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		testPath := filepath.Join(tmpDir, "subdir", "file.txt")
		err := file.CheckDiskSpace(testPath, 1024)
		assert.NoError(t, err)
	})
}

func TestValidateMimeType(t *testing.T) {
	t.Run("ValidateMimeType accepts PNG images", func(t *testing.T) {
		pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52}
		mimeType, err := file.ValidateMimeType(pngData)
		assert.NoError(t, err)
		assert.Equal(t, "image/png", mimeType)
	})

	t.Run("ValidateMimeType accepts plain text", func(t *testing.T) {
		textData := []byte("Hello, World! This is plain text.")
		mimeType, err := file.ValidateMimeType(textData)
		assert.NoError(t, err)
		assert.Contains(t, mimeType, "text/plain")
	})
}
