package main

import "github.com/mesosphere/installer/cmd/cli/cmd"

// Set via linker flag
var version string
var buildDate string

func main() {
	cmd.Execute(cmd.Version{Version: version, BuildDate: buildDate})
}
