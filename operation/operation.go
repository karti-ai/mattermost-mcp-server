package operation

import (
	"github.com/karti-ai/mattermost-mcp-server/operation/channel"
	"github.com/karti-ai/mattermost-mcp-server/operation/command"
	"github.com/karti-ai/mattermost-mcp-server/operation/dm"
	"github.com/karti-ai/mattermost-mcp-server/operation/file"
	"github.com/karti-ai/mattermost-mcp-server/operation/messaging"
	"github.com/karti-ai/mattermost-mcp-server/operation/outgoing"
	"github.com/karti-ai/mattermost-mcp-server/operation/reaction"
	"github.com/karti-ai/mattermost-mcp-server/operation/system"
	"github.com/karti-ai/mattermost-mcp-server/operation/team"
	"github.com/karti-ai/mattermost-mcp-server/operation/user"
	"github.com/karti-ai/mattermost-mcp-server/operation/webhook"
	"github.com/karti-ai/mattermost-mcp-server/pkg/flag"
	"github.com/karti-ai/mattermost-mcp-server/pkg/log"
	"github.com/karti-ai/mattermost-mcp-server/pkg/tool"
	"github.com/mark3labs/mcp-go/server"
)

func Register() []server.ServerTool {
	log.Infof("Registering tools for Mattermost MCP server %s", flag.Version)

	tools := tool.New()

	for _, t := range messaging.Tool.WriteTools() {
		tools.RegisterWrite(t)
	}

	for _, t := range file.Tool.WriteTools() {
		tools.RegisterWrite(t)
	}

	for _, t := range dm.Tool.WriteTools() {
		tools.RegisterWrite(t)
	}

	for _, t := range reaction.Tool.WriteTools() {
		tools.RegisterWrite(t)
	}

	for _, t := range channel.AdminTool.WriteTools() {
		tools.RegisterWrite(t)
	}

	for _, t := range messaging.Tool.ReadTools() {
		tools.RegisterRead(t)
	}

	for _, t := range file.Tool.ReadTools() {
		tools.RegisterRead(t)
	}

	for _, t := range channel.Tool.ReadTools() {
		tools.RegisterRead(t)
	}

	for _, t := range user.Tool.ReadTools() {
		tools.RegisterRead(t)
	}

	for _, t := range team.Tool.ReadTools() {
		tools.RegisterRead(t)
	}

	for _, t := range dm.Tool.ReadTools() {
		tools.RegisterRead(t)
	}

	for _, t := range reaction.Tool.ReadTools() {
		tools.RegisterRead(t)
	}

	for _, t := range webhook.Tool.WriteTools() {
		tools.RegisterWrite(t)
	}

	for _, t := range webhook.Tool.ReadTools() {
		tools.RegisterRead(t)
	}

	for _, t := range command.Tool.WriteTools() {
		tools.RegisterWrite(t)
	}

	for _, t := range outgoing.Tool.WriteTools() {
		tools.RegisterWrite(t)
	}

	for _, t := range outgoing.Tool.ReadTools() {
		tools.RegisterRead(t)
	}

	for _, t := range system.Tool.ReadTools() {
		tools.RegisterRead(t)
	}

	return tools.Tools()
}
