package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// version 在构建时通过 -ldflags "-X ...cmd.version=vX.Y.Z" 注入。
var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show goxctl version",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "goxctl %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
	},
}
