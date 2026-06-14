package main

import (
	"fmt"

	"github.com/chinayin/goxctl/internal/ext"
	"github.com/spf13/cobra"
)

var extensionCmd = &cobra.Command{
	Use:     "extension",
	Aliases: []string{"ext"},
	Short:   "管理 goxctl 扩展",
}

var extInstallCmd = &cobra.Command{
	Use:   "install <module> [version]",
	Short: "用 go install 安装扩展到 ~/.goxctl/extensions",
	Long:  "用 go install 安装扩展到 ~/.goxctl/extensions。\n\nmodule 可简写为 owner/repo（默认 github.com）；按约定从 <module>/cmd/<repo名> 安装。",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
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
	Short: "列出已安装扩展",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
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
	Short: "删除已安装扩展",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
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
