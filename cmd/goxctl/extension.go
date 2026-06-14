package main

import (
	"fmt"

	"github.com/chinayin/goxctl/internal/ext"
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
		cmd.SilenceUsage = true // 进入业务逻辑后，错误不再是用法问题
		m, err := ext.NewManager()
		if err != nil {
			return err
		}
		var version string
		if len(args) == 2 {
			version = args[1]
		}
		return m.Install(cmd.Context(), args[0], version)
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
			fmt.Fprintln(out, "(no extensions installed)")
			return nil
		}
		for _, n := range names {
			fmt.Fprintf(out, "  %s\n", n)
		}
		return nil
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
		return m.Remove(args[0])
	},
}

func init() {
	extensionCmd.AddCommand(extInstallCmd, extListCmd, extRemoveCmd)
}
