package main

import (
	"runtime/debug"

	"github.com/karti-ai/mattermost-mcp-server/cmd"
	"github.com/karti-ai/mattermost-mcp-server/pkg/flag"
)

var Version = "dev"

func init() {
	if Version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			Version = info.Main.Version
		}
	}
	flag.Version = Version
}

func main() {
	cmd.Execute()
}
