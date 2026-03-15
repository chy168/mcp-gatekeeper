package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

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

	// promptCreate asks the user if they want to create a new empty bundle.
	promptCreate := func(bundleName, src string) bool {
		fmt.Fprintf(os.Stderr, "Bundle %q not found in %s backend. Create it? [y/N] ", bundleName, src)
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil {
			return false
		}
		answer := strings.TrimSpace(strings.ToLower(line))
		return answer == "y" || answer == "yes"
	}

	// createEmptyBundle writes a valid empty YAML bundle to the backend.
	// GCP (and other backends) reject empty payloads, so we serialize {} rather
	// than passing an empty string.
	createEmptyBundle := func(ctx context.Context, backend secret.Backend) error {
		empty, err := secret.SerializeBundle(map[string]string{})
		if err != nil {
			return err
		}
		return backend.Set(ctx, secretSourceName, empty)
	}

	// getBundle fetches and parses the bundle, prompting to create if not found.
	getBundle := func(ctx context.Context, backend secret.Backend) (map[string]string, error) {
		content, err := backend.Get(ctx, secretSourceName)
		if err != nil {
			if errors.Is(err, secret.ErrBundleNotFound) {
				if !promptCreate(secretSourceName, secretSource) {
					return nil, fmt.Errorf("bundle %q not found", secretSourceName)
				}
				if setErr := createEmptyBundle(ctx, backend); setErr != nil {
					return nil, fmt.Errorf("failed to create bundle: %w", setErr)
				}
				fmt.Fprintf(os.Stderr, "✓ Created empty bundle %q\n", secretSourceName)
				return map[string]string{}, nil
			}
			return nil, err
		}
		return secret.ParseBundle(secretSourceName, content)
	}

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
			bundle, err := getBundle(ctx, backend)
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
			bundle, err := getBundle(ctx, backend)
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
				if errors.Is(err, secret.ErrBundleNotFound) {
					if !promptCreate(secretSourceName, secretSource) {
						return fmt.Errorf("bundle %q not found", secretSourceName)
					}
					if setErr := createEmptyBundle(ctx, backend); setErr != nil {
						return fmt.Errorf("failed to create bundle: %w", setErr)
					}
					fmt.Fprintf(os.Stderr, "✓ Created empty bundle %q\n", secretSourceName)
					return fmt.Errorf("key %q not found in the newly created bundle", key)
				}
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
