package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func init() {
	rootCmd.AddCommand(commandsCmd)
}

var commandsCmd = &cobra.Command{
	Use:   "commands",
	Short: "List all available commands with their flags",
	Long:  "Output a JSON description of all commands, flags, and arguments for AI agent discovery.",
	RunE: func(cmd *cobra.Command, args []string) error {
		formatter := formatterFromCtx(cmd.Context())

		commands := traverseCommands(rootCmd, "")

		items := make([]interface{}, len(commands))
		for i, c := range commands {
			items[i] = c
		}

		return formatter.FormatList(os.Stdout, items, []string{"command", "description"})
	},
}

type CommandInfo struct {
	Command     string     `json:"command"`
	Description string     `json:"description"`
	Flags       []FlagInfo `json:"flags,omitempty"`
	Args        []ArgInfo  `json:"args,omitempty"`
}

type FlagInfo struct {
	Name        string `json:"name"`
	Shorthand   string `json:"shorthand,omitempty"`
	Type        string `json:"type"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
	Required    bool   `json:"required,omitempty"`
}

type ArgInfo struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
}

func traverseCommands(cmd *cobra.Command, prefix string) []CommandInfo {
	var result []CommandInfo

	fullName := cmd.Name()
	if prefix != "" {
		fullName = prefix + " " + cmd.Name()
	}

	if cmd.Runnable() {
		info := CommandInfo{
			Command:     fullName,
			Description: cmd.Short,
		}

		cmd.NonInheritedFlags().VisitAll(func(f *pflag.Flag) {
			fi := FlagInfo{
				Name:        f.Name,
				Shorthand:   f.Shorthand,
				Type:        f.Value.Type(),
				Default:     f.DefValue,
				Description: f.Usage,
			}
			info.Flags = append(info.Flags, fi)
		})

		// Parse args from Use string
		if cmd.Args != nil {
			use := cmd.Use
			for _, part := range splitArgs(use) {
				if part[0] == '<' {
					info.Args = append(info.Args, ArgInfo{
						Name:     part[1 : len(part)-1],
						Required: true,
					})
				}
			}
		}

		result = append(result, info)
	}

	for _, child := range cmd.Commands() {
		if !child.Hidden {
			result = append(result, traverseCommands(child, fullName)...)
		}
	}

	return result
}

func splitArgs(use string) []string {
	var args []string
	parts := strings.Fields(use)
	if len(parts) <= 1 {
		return nil
	}
	for _, p := range parts[1:] {
		if len(p) < 2 {
			continue
		}
		if (p[0] == '<' && p[len(p)-1] == '>') || (p[0] == '[' && p[len(p)-1] == ']') {
			args = append(args, p)
		}
	}
	return args
}
