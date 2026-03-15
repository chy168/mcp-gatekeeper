package main

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/chy168/mcp-gatekeeper/internal/secret"
)

func main() {
	var secretSource string
	var secretSourceName string

	rootCmd := &cobra.Command{
		Use:   "mcp-gatekeeper-secret",
		Short: "Manage secrets in the mcp-gatekeeper YAML bundle",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if secretSource == "" {
				return fmt.Errorf("--secret-source is required (gcp, aws, or keychain)")
			}
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&secretSource, "secret-source", "", "Secret backend: gcp, aws, or keychain")
	rootCmd.PersistentFlags().StringVar(&secretSourceName, "secret-source-name", "mcp-gatekeeper", "Bundle name in the backend")

	// ── list ──────────────────────────────────────────────────────────────────
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all keys in the bundle",
		RunE: func(cmd *cobra.Command, args []string) error {
			backend, err := secret.NewBackend(secretSource)
			if err != nil {
				return err
			}
			ctx := context.Background()
			content, err := backend.Get(ctx, secretSourceName)
			if err != nil {
				return err
			}
			bundle, err := secret.ParseBundle(secretSourceName, content)
			if err != nil {
				return err
			}
			keys := make([]string, 0, len(bundle))
			for k := range bundle {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Println(k)
			}
			return nil
		},
	}

	// ── get ───────────────────────────────────────────────────────────────────
	var reveal bool
	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a key's value from the bundle",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			backend, err := secret.NewBackend(secretSource)
			if err != nil {
				return err
			}
			ctx := context.Background()
			content, err := backend.Get(ctx, secretSourceName)
			if err != nil {
				return err
			}
			bundle, err := secret.ParseBundle(secretSourceName, content)
			if err != nil {
				return err
			}
			val, err := secret.LookupBundle(bundle, key, secretSourceName)
			if err != nil {
				return err
			}
			if reveal {
				fmt.Printf("%s: %s\n", key, val)
			} else {
				fmt.Printf("%s: ****\n", key)
			}
			return nil
		},
	}
	getCmd.Flags().BoolVar(&reveal, "reveal", false, "Show the actual value instead of masking it")

	// ── set ───────────────────────────────────────────────────────────────────
	var fromFile string
	setCmd := &cobra.Command{
		Use:   "set <key> [value]",
		Short: "Set a key in the bundle",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			var value string

			if fromFile != "" {
				data, err := os.ReadFile(fromFile)
				if err != nil {
					return fmt.Errorf("failed to read file %q: %w", fromFile, err)
				}
				value = string(data)
			} else if len(args) == 2 {
				value = args[1]
			} else {
				return fmt.Errorf("value required: provide a positional argument or --from-file=<path>")
			}

			backend, err := secret.NewBackend(secretSource)
			if err != nil {
				return err
			}
			ctx := context.Background()
			if err := secret.SetBundleKey(ctx, backend, secretSourceName, key, value); err != nil {
				return err
			}
			fmt.Printf("✓ Set %q in bundle %q\n", key, secretSourceName)
			return nil
		},
	}
	setCmd.Flags().StringVar(&fromFile, "from-file", "", "Read value from file (supports multiline)")

	// ── delete ────────────────────────────────────────────────────────────────
	deleteCmd := &cobra.Command{
		Use:   "delete <key>",
		Short: "Delete a key from the bundle",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			backend, err := secret.NewBackend(secretSource)
			if err != nil {
				return err
			}
			ctx := context.Background()
			if err := secret.DeleteBundleKey(ctx, backend, secretSourceName, key); err != nil {
				return err
			}
			fmt.Printf("✓ Deleted %q from bundle %q\n", key, secretSourceName)
			return nil
		},
	}

	rootCmd.AddCommand(listCmd, getCmd, setCmd, deleteCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
