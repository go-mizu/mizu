package cli

import (
	"fmt"

	"github.com/go-mizu/mizu/blueprints/search/pkg/arctic"
	"github.com/spf13/cobra"
)

func newArcticCatalogSizes() *cobra.Command {
	var repoRoot string

	cmd := &cobra.Command{
		Use:   "catalog-sizes",
		Short: "Fetch .zst file sizes from torrents and save to zst_sizes.json",
		Long: `Queries the Arctic Shift bundle torrent (2005-12 through 2023-12) and all
individual monthly torrents (2024-01 through 2026-02) for exact .zst file sizes.
Saves the result to zst_sizes.json in the repo root.

This is a one-time network operation — subsequent pipeline runs load from the
cached file. Run this once before starting the publish pipeline to populate the
size catalog used for README coverage stats.`,
		Example: `  search arctic catalog-sizes
  search arctic catalog-sizes --repo-root ~/data/arctic/repo`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := arctic.Config{RepoRoot: repoRoot}
			cfg = cfg.WithDefaults()
			if err := cfg.EnsureDirs(); err != nil {
				return fmt.Errorf("ensure dirs: %w", err)
			}
			fmt.Printf("  Fetching .zst sizes from torrents → %s\n", cfg.ZstSizesPath())
			sizes, err := arctic.FetchZstSizes(cmd.Context(), cfg, cfg.ZstSizesPath())
			if err != nil {
				return fmt.Errorf("fetch zst sizes: %w", err)
			}
			fmt.Printf("  Done: %d entries written to %s\n", sizes.Len(), cfg.ZstSizesPath())
			return nil
		},
	}

	cmd.Flags().StringVar(&repoRoot, "repo-root", "", "Local root directory (default: $HOME/data/arctic/repo)")
	return cmd
}
