package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/johanviberg/zd/internal/config"
	"github.com/johanviberg/zd/internal/output"
)

type contextKey string

const (
	ctxKeyConfig    contextKey = "config"
	ctxKeyFormatter contextKey = "formatter"
)

var rootCmd = &cobra.Command{
	Use:           "zd",
	Short:         "Zendesk CLI - AI agent-friendly command-line interface for Zendesk",
	Long:          "A CLI tool for interacting with Zendesk's ticketing REST API, designed for both human users and AI agents.",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		profile, _ := cmd.Flags().GetString("profile")
		cfg, err := config.Load(profile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		subdomain, _ := cmd.Flags().GetString("subdomain")
		if subdomain != "" {
			cfg.Subdomain = subdomain
		}

		outputFmt, _ := cmd.Flags().GetString("output")
		fields, _ := cmd.Flags().GetStringSlice("fields")
		noHeaders, _ := cmd.Flags().GetBool("no-headers")

		formatter, err := output.NewFormatter(outputFmt, fields, noHeaders)
		if err != nil {
			return err
		}

		ctx := cmd.Context()
		ctx = context.WithValue(ctx, ctxKeyConfig, cfg)
		ctx = context.WithValue(ctx, ctxKeyFormatter, formatter)
		cmd.SetContext(ctx)
		return nil
	},
}

func Execute(version, commit, date string) {
	buildVersion = version
	buildCommit = commit
	buildDate = date

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("output", "o", "text", "Output format: text, json, ndjson")
	rootCmd.PersistentFlags().StringSlice("fields", nil, "Field projection (comma-separated)")
	rootCmd.PersistentFlags().Bool("no-headers", false, "Omit table headers in text mode")
	rootCmd.PersistentFlags().Bool("non-interactive", false, "Never prompt for input")
	rootCmd.PersistentFlags().Bool("yes", false, "Auto-confirm prompts")
	rootCmd.PersistentFlags().Bool("debug", false, "Debug logging to stderr")
	rootCmd.PersistentFlags().String("trace-id", "", "Trace ID attached to API requests")
	rootCmd.PersistentFlags().String("subdomain", "", "Override Zendesk subdomain")
	rootCmd.PersistentFlags().String("profile", "default", "Config profile")
}

func isNonInteractive(cmd *cobra.Command) bool {
	flag, _ := cmd.Flags().GetBool("non-interactive")
	if flag {
		return true
	}
	return !term.IsTerminal(int(os.Stdin.Fd()))
}

func configFromCtx(ctx context.Context) *config.Config {
	v, ok := ctx.Value(ctxKeyConfig).(*config.Config)
	if !ok || v == nil {
		panic("configFromCtx called before PersistentPreRunE — this is a bug")
	}
	return v
}

func formatterFromCtx(ctx context.Context) output.Formatter {
	v, ok := ctx.Value(ctxKeyFormatter).(output.Formatter)
	if !ok || v == nil {
		panic("formatterFromCtx called before PersistentPreRunE — this is a bug")
	}
	return v
}
