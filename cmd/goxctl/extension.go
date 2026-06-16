package main

import (
	"fmt"
	"path"
	"strings"

	"github.com/chinayin/goxctl/internal/ext"
	"github.com/chinayin/goxctl/internal/ui"
	"github.com/spf13/cobra"
)

var extensionCmd = &cobra.Command{
	Use:     "extension",
	Aliases: []string{"ext"},
	Short:   "Manage goxctl extensions",
}

var extInstallCmd = &cobra.Command{
	Use:   "install <module> [version]",
	Short: "Install an extension into ~/.gox/extensions",
	Long: `Install an extension into ~/.gox/extensions.

Prefers a prebuilt binary from the extension's GitHub Releases (no Go required),
and falls back to "go install" when no binary matches the current platform.

<module> may be shortened to owner/repo (host defaults to github.com).`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		m, err := ext.NewManager()
		if err != nil {
			return err
		}
		var version string
		if len(args) == 2 {
			version = args[1]
		}
		if err := m.Install(cmd.Context(), args[0], version); err != nil {
			return err
		}
		name := extName(args[0])
		ui.Successf(cmd.OutOrStdout(), "installed %s %s", name, m.ExtVersion(name))
		return nil
	},
}

var extListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed extensions",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		m, err := ext.NewManager()
		if err != nil {
			return err
		}
		names, err := m.List()
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		if len(names) == 0 {
			fmt.Fprintln(out, "no extensions installed")
			return nil
		}
		t := ui.Table(out)
		fmt.Fprintln(t, "NAME\tVERSION\tMODULE")
		for _, n := range names {
			fmt.Fprintf(t, "%s\t%s\t%s\n", n, dash(m.ExtVersion(n)), dash(m.ExtModule(n)))
		}
		return t.Flush()
	},
}

var extRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an installed extension",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		m, err := ext.NewManager()
		if err != nil {
			return err
		}
		if err := m.Remove(args[0]); err != nil {
			return err
		}
		ui.Successf(cmd.OutOrStdout(), "removed %s", args[0])
		return nil
	},
}

var extUpgradeAll bool

var extUpgradeCmd = &cobra.Command{
	Use:   "upgrade [name]",
	Short: "Upgrade installed extensions to the latest release",
	Long: `Reinstall extensions at their latest release.

  goxctl extension upgrade <name>   upgrade one extension
  goxctl extension upgrade --all    upgrade all installed extensions`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		m, err := ext.NewManager()
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()

		var names []string
		switch {
		case extUpgradeAll:
			if names, err = m.List(); err != nil {
				return err
			}
		case len(args) == 1:
			names = []string{args[0]}
		default:
			return fmt.Errorf("ext: specify an extension name or --all")
		}

		for _, n := range names {
			// module 优先取自清单（任意已装扩展），回退注册表
			mod := m.ExtModule(n)
			if mod == "" {
				var ok bool
				if mod, ok = installHint(n); !ok {
					fmt.Fprintf(out, "%s: skipped (unknown module; reinstall manually)\n", n)
					continue
				}
			}
			current := m.ExtVersion(n)
			target, err := ext.LatestVersion(cmd.Context(), mod)
			if err != nil {
				return fmt.Errorf("ext: check %s: %w", n, err)
			}
			if current != "" && current == target {
				fmt.Fprintf(out, "%s already up to date (%s)\n", n, current)
				continue
			}
			if err := m.Install(cmd.Context(), mod, ""); err != nil {
				return fmt.Errorf("ext: upgrade %s: %w", n, err)
			}
			if current == "" {
				ui.Successf(out, "%s → %s", n, target)
			} else {
				ui.Successf(out, "%s %s → %s", n, current, target)
			}
		}
		return nil
	},
}

// extName 从 module 取扩展名（goxctl-claude → claude）。
func extName(module string) string {
	return strings.TrimPrefix(path.Base(module), "goxctl-")
}

// dash 把空串显示为 -（表格占位）。
func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func init() {
	extUpgradeCmd.Flags().BoolVar(&extUpgradeAll, "all", false, "upgrade all installed extensions")
	extensionCmd.AddCommand(extInstallCmd, extListCmd, extRemoveCmd, extUpgradeCmd)
}
