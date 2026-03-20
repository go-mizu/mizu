package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newBucketCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bucket",
		Short: "Manage buckets",
	}

	cmd.AddCommand(
		newBucketCreateCmd(),
		newBucketRmCmd(),
	)
	return cmd
}

func newBucketCreateCmd() *cobra.Command {
	var (
		public    bool
		sizeLimit string
		types     string
	)

	cmd := &cobra.Command{
		Use:     "create <name>",
		Short:   "Create a new bucket",
		Aliases: []string{"new"},
		Example: `  storage bucket create avatars --public
  storage bucket create logs
  storage bucket create images --public --size-limit 5MB --types image/png,image/jpeg`,
		Args: cobra.ExactArgs(1),
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			name := args[0]
			body := map[string]any{
				"name":   name,
				"public": public,
			}

			if sizeLimit != "" {
				bytes := parseSizeLimit(sizeLimit)
				if bytes > 0 {
					body["file_size_limit"] = bytes
				}
			}

			if types != "" {
				typeList := strings.Split(types, ",")
				for i, t := range typeList {
					typeList[i] = strings.TrimSpace(t)
				}
				body["allowed_mime_types"] = typeList
			}

			data, err := d.Client.DoJSON("POST", "/bucket", body)
			if err != nil {
				return err
			}

			if globalFlags.json {
				var raw json.RawMessage
				json.Unmarshal(data, &raw)
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(raw)
			}

			vis := "private"
			if public {
				vis = "public"
			}
			d.Out.Info("Created", fmt.Sprintf("bucket %s (%s)", name, vis))
			return nil
		}),
	}

	cmd.Flags().BoolVar(&public, "public", false, "make bucket publicly readable")
	cmd.Flags().StringVar(&sizeLimit, "size-limit", "", "max file size (e.g. 5MB, 100KB)")
	cmd.Flags().StringVar(&types, "types", "", "allowed MIME types (comma-separated)")
	return cmd
}

func newBucketRmCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "rm <name>",
		Short:   "Delete a bucket",
		Aliases: []string{"delete"},
		Example: `  storage bucket rm old-bucket
  storage bucket rm old-bucket --force`,
		Args: cobra.ExactArgs(1),
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			name := args[0]

			// Get bucket ID
			bucketsData, err := d.Client.Get("/bucket")
			if err != nil {
				return err
			}

			var buckets []bucketInfo
			json.Unmarshal(bucketsData, &buckets)

			var bucketID string
			for _, b := range buckets {
				if b.Name == name {
					bucketID = b.ID
					break
				}
			}
			if bucketID == "" {
				return &CLIError{Code: ExitNotFound, Msg: "bucket not found", Hint: fmt.Sprintf("No bucket named '%s'", name)}
			}

			if force {
				d.Client.DoJSON("POST", "/bucket/"+bucketID+"/empty", nil)
				d.Out.Info("Emptied", "bucket "+name)
			}

			if _, err := d.Client.Delete("/bucket/" + bucketID); err != nil {
				return err
			}

			d.Out.Info("Deleted", "bucket "+name)
			return nil
		}),
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "empty bucket before deleting")
	return cmd
}

// parseSizeLimit converts a size string like "5MB" to bytes.
func parseSizeLimit(s string) int64 {
	s = strings.TrimSpace(s)
	upper := strings.ToUpper(s)

	var multiplier int64 = 1
	numStr := s

	switch {
	case strings.HasSuffix(upper, "GB"):
		multiplier = 1 << 30
		numStr = s[:len(s)-2]
	case strings.HasSuffix(upper, "MB"):
		multiplier = 1 << 20
		numStr = s[:len(s)-2]
	case strings.HasSuffix(upper, "KB"):
		multiplier = 1 << 10
		numStr = s[:len(s)-2]
	}

	n, err := strconv.ParseInt(strings.TrimSpace(numStr), 10, 64)
	if err != nil {
		return 0
	}
	return n * multiplier
}
