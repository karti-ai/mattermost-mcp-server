# Changelog

All notable changes to the Mattermost MCP Server will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-04-12

### Added
- Initial release of Mattermost MCP Server
- Support for 22 MCP tools categorized as Read, Write, and Admin operations

#### Read Operations (11)
- `mattermost_list_channels` - List accessible channels in a team
- `mattermost_get_channel_by_name` - Get a channel by name in a team
- `mattermost_get_channel_info` - Get detailed channel information including member count
- `mattermost_search_users` - Search for users by username, email, or display name
- `mattermost_list_teams` - List all teams the bot has access to
- `mattermost_search_posts` - Search for posts/messages in a team using search terms
- `mattermost_get_thread` - Read all messages in a thread conversation
- `mattermost_get_unread_count` - Get unread message counts for all channels in a team
- `mattermost_mark_channel_read` - Mark a channel as read (clear unread notifications)
- `mattermost_get_user_status` - Get the online status of a user (online, away, dnd, offline)
- `mattermost_get_channel_messages` - Read message history from a channel

#### Write Operations (8)
- `mattermost_send_message` - Send a message to a channel or user
- `mattermost_edit_message` - Edit an existing message
- `mattermost_delete_message` - Delete a message from a channel
- `mattermost_create_dm` - Create a direct message channel with a user
- `mattermost_add_reaction` - Add an emoji reaction to a message
- `mattermost_remove_reaction` - Remove an emoji reaction from a message
- `mattermost_upload_file` - Upload a file to a channel or user
- `mattermost_download_file` - Download a file from a message

#### Admin Operations (4)
- `mattermost_create_channel` - Create a new channel (public or private) in a team
- `mattermost_invite_to_channel` - Invite a user to a channel
- `mattermost_delete_channel` - Delete/archive a channel (soft delete by default)
- `mattermost_leave_channel` - Remove self from a channel

### Features
- Multiple authentication methods (Bot tokens, PAT, User tokens)
- Read-only mode for safe exploration (`--read-only` flag)
- Debug logging support (`--debug` flag)
- Configuration via environment variables or CLI flags
- Comprehensive error handling with descriptive messages
- File upload security with validation and restrictions
- Integration tests for core functionality
- OpenClaw and Claude Code configuration support

### Security
- Path traversal prevention for file operations
- Dangerous file extension blocking
- File size limits (50MB maximum)
- MIME type whitelist validation for uploads
- Token-safe logging (tokens are never logged)
- TLS verification enabled by default
- `--insecure` flag available only for development/testing

## [Unreleased]

### Planned
- WebSocket support for real-time notifications
- Additional search capabilities (file search, user directory)
- Message threading improvements
- Bulk operations support
- Rate limiting and throttling options
- Extended file type support for uploads
