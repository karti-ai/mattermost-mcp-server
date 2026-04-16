package cmd

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/karti-ai/mattermost-mcp-server/operation"
	pkgflag "github.com/karti-ai/mattermost-mcp-server/pkg/flag"
	"github.com/karti-ai/mattermost-mcp-server/pkg/log"
	"github.com/karti-ai/mattermost-mcp-server/pkg/mattermost"
	"github.com/mark3labs/mcp-go/server"
)

func Execute() {
	// Parse CLI flags
	flag.StringVar(&pkgflag.Token, "token", os.Getenv("MATTERMOST_TOKEN"), "Mattermost access token")
	flag.StringVar(&pkgflag.Token, "t", os.Getenv("MATTERMOST_TOKEN"), "Mattermost access token (short)")
	flag.StringVar(&pkgflag.BotToken, "bot-token", os.Getenv("MATTERMOST_BOT_TOKEN"), "Mattermost bot token")
	flag.StringVar(&pkgflag.PAT, "pat", os.Getenv("MATTERMOST_PAT"), "Mattermost personal access token")
	flag.StringVar(&pkgflag.Host, "host", os.Getenv("MATTERMOST_HOST"), "Mattermost server URL")
	flag.StringVar(&pkgflag.Host, "H", os.Getenv("MATTERMOST_HOST"), "Mattermost server URL (short)")
	flag.StringVar(&pkgflag.Team, "team", os.Getenv("MATTERMOST_TEAM"), "Mattermost team (optional)")
	flag.BoolVar(&pkgflag.ReadOnly, "read-only", os.Getenv("MATTERMOST_READONLY") == "true", "Enable read-only mode")
	flag.BoolVar(&pkgflag.ReadOnly, "r", os.Getenv("MATTERMOST_READONLY") == "true", "Enable read-only mode (short)")
	flag.BoolVar(&pkgflag.Debug, "debug", os.Getenv("MATTERMOST_DEBUG") == "true", "Enable debug logging")
	flag.BoolVar(&pkgflag.Debug, "d", os.Getenv("MATTERMOST_DEBUG") == "true", "Enable debug logging (short)")
	flag.BoolVar(&pkgflag.Insecure, "insecure", os.Getenv("MATTERMOST_INSECURE") == "true", "Allow insecure TLS connections")
	flag.BoolVar(&pkgflag.ShowVersion, "version", false, "Show version and exit")
	flag.BoolVar(&pkgflag.ShowVersion, "v", false, "Show version (short)")

	flag.Parse()

	if pkgflag.ShowVersion {
		fmt.Printf("mattermost-mcp-server %s\n", pkgflag.Version)
		os.Exit(0)
	}

	// Validate at least one token is provided
	if pkgflag.Token == "" && pkgflag.BotToken == "" && pkgflag.PAT == "" {
		fmt.Fprintln(os.Stderr, "Error: At least one token is required (via --token, --bot-token, --pat, or environment variables)")
		flag.Usage()
		os.Exit(1)
	}

	if pkgflag.Host == "" {
		fmt.Fprintln(os.Stderr, "Error: Mattermost host is required (via --host or MATTERMOST_HOST environment variable)")
		flag.Usage()
		os.Exit(1)
	}

	// Initialize logging
	logLevel := "info"
	if pkgflag.Debug {
		logLevel = "debug"
	}
	if err := log.Initialize(logLevel); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logging: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	log.Infof("Starting mattermost-mcp-server %s", pkgflag.Version)
	log.Infof("Host: %s", pkgflag.Host)
	log.Infof("Read-only mode: %v", pkgflag.ReadOnly)

	client := mattermost.NewClient(pkgflag.Host, pkgflag.BotToken, pkgflag.PAT)
	mattermost.SetGlobalClient(client)
	log.Infof("Initialized Mattermost client for host: %s", pkgflag.Host)

	// Create MCP server
	mcpServer := server.NewMCPServer(
		"mattermost-mcp",
		pkgflag.Version,
	)

	// Register tools
	tools := operation.Register()
	for _, tool := range tools {
		mcpServer.AddTool(tool.Tool, tool.Handler)
	}

	log.Infof("Registered %d tools", len(tools))

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info("Received shutdown signal, shutting down gracefully...")
	}()

	// Start stdio server
	log.Info("Starting stdio server...")
	if err := server.ServeStdio(mcpServer); err != nil {
		log.Errorf("Server error: %v", err)
		os.Exit(1)
	}

	log.Info("Server stopped")
}
