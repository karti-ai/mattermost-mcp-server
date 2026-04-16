# Mattermost MCP Server

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.23%2B-brightgreen.svg)](go.mod)

A Model Context Protocol (MCP) server for Mattermost, enabling AI assistants to interact with your Mattermost workspace. This server provides tools for sending messages, managing channels, searching users, and handling file uploads/downloads.

## Features

- **27 Total Tools**: Comprehensive coverage of Mattermost operations
- **Read Operations**: Teams, channels, users, messages, reactions, files
- **Write Operations**: Send/edit/delete messages, reactions, channel management, file operations
- **Read-Only Mode**: Safe mode with only read operations enabled
- **Dual Token Support**: Bot token for reads, PAT for writes (enhanced security)
- **Debug Logging**: Optional verbose logging for troubleshooting

## Installation

### Method 1: Using Go Install

```bash
go install github.com/karti-ai/mattermost-mcp-server@latest
```

### Method 2: Binary Download

Download the latest binary from the [releases page](https://github.com/karti-ai/mattermost-mcp-server/releases):

```bash
curl -L -o mattermost-mcp-server https://github.com/karti-ai/mattermost-mcp-server/releases/latest/download/mattermost-mcp-server-linux-amd64
chmod +x mattermost-mcp-server
```

### Method 3: Build from Source

```bash
git clone https://github.com/karti-ai/mattermost-mcp-server.git
cd mattermost-mcp-server
go build -ldflags "-s -w" -o mattermost-mcp-server .
```

## Configuration

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `MATTERMOST_HOST` | Mattermost server URL (e.g., `https://mattermost.example.com`) | Yes |
| `MATTERMOST_TOKEN` | Access token (bot, PAT, or user) | Yes* |
| `MATTERMOST_BOT_TOKEN` | Bot token (alternative to TOKEN) | Yes* |
| `MATTERMOST_PAT` | Personal Access Token (alternative to TOKEN) | Yes* |
| `MATTERMOST_TEAM` | Default team name | No |

*At least one token type is required.

### CLI Flags

| Flag | Short | Description |
|------|-------|-------------|
| `-host` | `-H` | Mattermost server URL |
| `-token` | `-t` | Mattermost access token |
| `-bot-token` | | Mattermost bot token |
| `-pat` | | Mattermost personal access token |
| `-team` | | Mattermost team name |
| `-debug` | `-d` | Enable debug logging |
| `-insecure` | | Allow insecure TLS connections |
| `-read-only` | `-r` | Enable read-only mode |
| `-version` | `-v` | Show version and exit |

### Example .env File

```bash
# Mattermost Configuration
MATTERMOST_HOST=https://mattermost.company.com
MATTERMOST_TOKEN=your-bot-token-here
MATTERMOST_TEAM=general

# Optional: Enable debug logging
# DEBUG=true
```

## Available Tools

The server exposes **27 tools** grouped by category:

### Team Operations (1)
| Tool | Description |
|------|-------------|
| `mattermost_list_teams` | List all teams the bot has access to |

### Channel Operations (5)
| Tool | Description |
|------|-------------|
| `mattermost_list_channels` | List all accessible channels in a team |
| `mattermost_get_channel_by_name` | Get a channel by name in a team |
| `mattermost_get_channel_info` | Get detailed channel info with member count |
| `mattermost_list_channel_members` | List all members of a channel |
| `mattermost_mark_channel_read` | Mark a channel as read (clear unread) |

### Channel Admin (4)
| Tool | Description |
|------|-------------|
| `mattermost_create_channel` | Create a new public or private channel |
| `mattermost_invite_to_channel` | Invite a user to a channel |
| `mattermost_leave_channel` | Leave a channel |
| `mattermost_delete_channel` | Delete/archive a channel |

### Messaging (6)
| Tool | Description |
|------|-------------|
| `mattermost_send_message` | Send a message (supports thread replies) |
| `mattermost_edit_message` | Edit an existing message |
| `mattermost_delete_message` | Delete a message |
| `mattermost_get_channel_messages` | Read message history with pagination |
| `mattermost_get_thread` | Read all messages in a thread |
| `mattermost_search_posts` | Search messages in a team |

### User Operations (4)
| Tool | Description |
|------|-------------|
| `mattermost_search_users` | Search users by username, email, or name |
| `mattermost_get_user` | Get a specific user by ID or username |
| `mattermost_get_user_status` | Get online status (online/away/dnd/offline) |
| `mattermost_update_user_status` | Set your status (requires PAT) |

### Direct Messages (2)
| Tool | Description |
|------|-------------|
| `mattermost_create_dm` | Create a 1:1 direct message channel |
| `mattermost_create_group_dm` | Create a group DM with multiple users |

### Reactions (3)
| Tool | Description |
|------|-------------|
| `mattermost_add_reaction` | Add an emoji reaction to a message |
| `mattermost_remove_reaction` | Remove a reaction from a message |
| `mattermost_list_reactions` | List all reactions on a message |

### File Operations (2)
| Tool | Description |
|------|-------------|
| `mattermost_upload_file` | Upload a file to a channel |
| `mattermost_download_file` | Download a file from a message |

### Unread/Notifications (1)
| Tool | Description |
|------|-------------|
| `mattermost_get_unread_count` | Get unread message counts per channel |

### Read-Only Mode

When running with `--read-only` or `-r` flag, only the 14 read tools are available, preventing any modifications to your Mattermost workspace.

## OpenClaw Setup

Add the following to your `~/.config/openclaw/openclaw.json`:

```json
{
  "mcpServers": {
    "mattermost": {
      "command": "mattermost-mcp-server",
      "env": {
        "MATTERMOST_HOST": "https://mattermost.example.com",
        "MATTERMOST_TOKEN": "your-token-here",
        "MATTERMOST_TEAM": "general"
      }
    }
  }
}
```

With read-only mode:

```json
{
  "mcpServers": {
    "mattermost": {
      "command": "mattermost-mcp-server",
      "args": ["--read-only"],
      "env": {
        "MATTERMOST_HOST": "https://mattermost.example.com",
        "MATTERMOST_TOKEN": "your-token-here"
      }
    }
  }
}
```

## Claude Code Setup

### Method 1: Using .mcp.json

Create `.mcp.json` in your project root:

```json
{
  "mcpServers": {
    "mattermost": {
      "command": "mattermost-mcp-server",
      "env": {
        "MATTERMOST_HOST": "https://mattermost.example.com",
        "MATTERMOST_TOKEN": "your-token-here",
        "MATTERMOST_TEAM": "general"
      }
    }
  }
}
```

### Method 2: CLI Setup

```bash
# Add to Claude Code configuration
claude config add mcpServer mattermost mattermost-mcp-server

# Set environment variables
claude config set mattermost.env.MATTERMOST_HOST https://mattermost.example.com
claude config set mattermost.env.MATTERMOST_TOKEN your-token-here
```

### Method 3: Direct Command

```bash
claude --mcp mattermost mattermost-mcp-server -- -H https://mattermost.example.com -t your-token-here
```

## Security Considerations

### Token Storage

- **Never commit tokens to version control** - Always use environment variables
- **Use bot tokens when possible** - Bot tokens have limited permissions and are safer than user tokens
- **Rotate tokens regularly** - Change tokens periodically and after any suspected compromise
- **Use secrets management** - For production, use tools like HashiCorp Vault, AWS Secrets Manager, or Kubernetes Secrets

### Read-Only Mode

- **Enable for safe exploration** - Use `--read-only` flag when first testing or giving AI assistants limited access
- **Audit before write access** - Review all write tool descriptions before disabling read-only mode
- **Principle of least privilege** - Only enable write access when absolutely necessary

### File Upload Restrictions

- The server allows file uploads to Mattermost channels
- Configure Mattermost server-side file upload limits and restrictions
- Be aware that AI assistants can upload arbitrary files if given write access
- Consider read-only mode if file upload functionality is not required

### TLS/SSL

- **Always use HTTPS** in production
- The `--insecure` flag disables TLS certificate verification - **only use for development/testing**
- Ensure your Mattermost server has valid TLS certificates

### Network Security

- Run the MCP server on localhost or behind a firewall
- Do not expose the MCP server to untrusted networks
- Use VPN or SSH tunnels when accessing remote Mattermost servers

## Usage Examples

### Listing Channels

```
Please list all channels in the general team
```

### Sending a Message

```
Send a message to the "announcements" channel saying "Meeting starts in 5 minutes"
```

### Searching Users

```
Find users with "john" in their username
```

### Managing Reactions

```
Add a :thumbsup: reaction to the last message in the general channel
```

### File Operations

```
Upload the report.pdf file to the marketing channel
```

## Troubleshooting

### Debug Mode

Enable debug logging to see detailed request/response information:

```bash
mattermost-mcp-server -d
```

### Common Issues

1. **Connection refused**: Check that `MATTERMOST_HOST` includes the full URL with protocol (http/https)
2. **401 Unauthorized**: Verify your token is valid and has not expired
3. **403 Forbidden**: Ensure the token has appropriate permissions for the operations you're trying to perform
4. **Team not found**: Verify the team name exactly matches the URL slug in Mattermost

## Development

### Building

```bash
go build -ldflags "-s -w" -o mattermost-mcp-server .
```

### Testing

```bash
go test ./...
```

### Smoke Tests

```bash
# Build and verify
make build

# Run tests
./mattermost-mcp-server --version
./mattermost-mcp-server --help
```

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Feel free to open issues and pull requests.

## Support

- **Issues**: [GitHub Issues](https://github.com/karti-ai/mattermost-mcp-server/issues)
- **Discussions**: [GitHub Discussions](https://github.com/karti-ai/mattermost-mcp-server/discussions)

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history and changes.
