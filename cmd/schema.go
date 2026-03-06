package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func init() {
	rootCmd.AddCommand(schemaCmd)

	schemaCmd.Flags().String("command", "", "Command name (e.g., 'tickets create')")
	schemaCmd.MarkFlagRequired("command")
}

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Output JSON Schema for a command's input",
	Long:  "Generate a JSON Schema describing the flags and arguments of a given command, for AI agent tool calling.",
	RunE: func(cmd *cobra.Command, args []string) error {
		commandName, _ := cmd.Flags().GetString("command")

		target := findCommand(rootCmd, commandName)
		if target == nil {
			return fmt.Errorf("command not found: %s", commandName)
		}

		schema := generateSchema(target)

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(schema)
	},
}

func findCommand(root *cobra.Command, name string) *cobra.Command {
	parts := strings.Fields(name)
	current := root
	for _, part := range parts {
		found := false
		for _, child := range current.Commands() {
			if child.Name() == part {
				current = child
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}
	if current == root {
		return nil
	}
	return current
}

func generateSchema(cmd *cobra.Command) map[string]interface{} {
	schema := map[string]interface{}{
		"$schema":     "https://json-schema.org/draft/2020-12/schema",
		"title":       cmd.CommandPath(),
		"description": cmd.Short,
		"type":        "object",
	}

	properties := map[string]interface{}{}
	required := []string{}

	cmd.NonInheritedFlags().VisitAll(func(f *pflag.Flag) {
		prop := map[string]interface{}{
			"description": f.Usage,
		}

		switch f.Value.Type() {
		case "string":
			prop["type"] = "string"
			if f.DefValue != "" {
				prop["default"] = f.DefValue
			}
		case "int", "int64":
			prop["type"] = "integer"
			if f.DefValue != "0" {
				prop["default"] = f.DefValue
			}
		case "bool":
			prop["type"] = "boolean"
			if f.DefValue == "true" {
				prop["default"] = true
			}
		case "stringSlice", "stringArray":
			prop["type"] = "array"
			prop["items"] = map[string]interface{}{"type": "string"}
		default:
			prop["type"] = "string"
		}

		properties[f.Name] = prop
	})

	// Check for required flags
	cmd.NonInheritedFlags().VisitAll(func(f *pflag.Flag) {
		for _, ann := range f.Annotations {
			for _, v := range ann {
				if v == "true" {
					required = append(required, f.Name)
				}
			}
		}
	})

	// Parse positional args from Use
	useArgs := splitArgs(cmd.Use)
	for _, arg := range useArgs {
		name := arg
		if len(name) > 2 && name[0] == '<' {
			name = name[1 : len(name)-1]
		}
		properties[name] = map[string]interface{}{
			"type":        "string",
			"description": "Positional argument: " + name,
		}
		required = append(required, name)
	}

	schema["properties"] = properties
	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}
