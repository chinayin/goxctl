package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chinayin/goxctl/internal/ext"
	"github.com/chinayin/goxctl/internal/ui"
	"github.com/spf13/cobra"
)

// selfModule 是 goxctl 自身的仓库，upgrade 从它的 release 自更新。
const selfModule = "chinayin/goxctl"

var upgradeCheck bool

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade goxctl to the latest release",
	Long: `Download the latest goxctl release for this platform and replace the running binary.

Use --check to only report whether a newer version is available, without installing.
Updating extensions is separate: goxctl extension upgrade <name>|--all.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		out := cmd.OutOrStdout()

		latest, err := ext.LatestVersion(cmd.Context(), selfModule)
		if err != nil {
			return err
		}
		if sameVersion(latest, version) {
			fmt.Fprintf(out, "goxctl already up to date (%s)\n", version)
			return nil
		}
		if upgradeCheck {
			fmt.Fprintf(out, "new version available: %s (current %s) — run: goxctl upgrade\n", latest, version)
			return nil
		}

		self, err := os.Executable()
		if err != nil {
			return err
		}
		if resolved, e := filepath.EvalSymlinks(self); e == nil {
			self = resolved
		}

		tag, err := ext.SelfUpdate(cmd.Context(), selfModule, self)
		if err != nil {
			return err
		}
		ui.Successf(out, "upgraded goxctl %s → %s", version, tag)
		return nil
	},
}

// sameVersion 比较当前注入版本与最新 tag，容忍 v 前缀差异。
func sameVersion(latest, current string) bool {
	return strings.TrimPrefix(latest, "v") == strings.TrimPrefix(current, "v")
}

func init() {
	upgradeCmd.Flags().BoolVar(&upgradeCheck, "check", false, "only check for a newer version, don't install")
	rootCmd.AddCommand(upgradeCmd)
}
