package storage

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	var (
		bucketFlag string
		typeFlag   string
	)

	cmd := &cobra.Command{
		Use:     "search <query>",
		Short:   "Search for objects by name",
		Aliases: []string{"find"},
		Example: `  storage search report
  storage search --bucket docs --type image/png
  storage search report --json`,
		Args: cobra.MaximumNArgs(1),
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			query := ""
			if len(args) > 0 {
				query = args[0]
			}
			if query == "" && typeFlag == "" {
				return &CLIError{Code: ExitUsage, Msg: "query required", Hint: "Usage: storage search <query> [--bucket <name>] [--type <mime>]"}
			}

			if bucketFlag != "" {
				return searchBucket(d, bucketFlag, query, typeFlag)
			}
			return searchAll(d, query, typeFlag)
		}),
	}

	cmd.Flags().StringVarP(&bucketFlag, "bucket", "b", "", "search in specific bucket")
	cmd.Flags().StringVarP(&typeFlag, "type", "T", "", "filter by MIME type")
	return cmd
}

func searchBucket(d *Deps, bucket, query, mimeType string) error {
	params := url.Values{}
	params.Set("bucket", bucket)
	if query != "" {
		params.Set("query", query)
	}
	if mimeType != "" {
		params.Set("type", mimeType)
	}

	data, err := d.Client.Get("/drive/search?" + params.Encode())
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

	var resp struct {
		Items []objectInfo `json:"items"`
	}
	json.Unmarshal(data, &resp)

	if len(resp.Items) == 0 {
		d.Out.Info("No results", "")
		return nil
	}

	fmt.Printf("%-12s %-30s %10s  %s\n", "BUCKET", "PATH", "SIZE", "TYPE")
	for _, o := range resp.Items {
		fmt.Printf("%-12s %-30s %10s  %s\n", bucket, o.Path, HumanSize(o.Size), o.ContentType)
	}
	return nil
}

func searchAll(d *Deps, query, mimeType string) error {
	// List all buckets first
	bucketsData, err := d.Client.Get("/bucket")
	if err != nil {
		return err
	}

	var buckets []bucketInfo
	json.Unmarshal(bucketsData, &buckets)

	if !globalFlags.json {
		fmt.Printf("%-12s %-30s %10s  %s\n", "BUCKET", "PATH", "SIZE", "TYPE")
	}

	var allResults []json.RawMessage

	for _, b := range buckets {
		params := url.Values{}
		params.Set("bucket", b.Name)
		if query != "" {
			params.Set("query", query)
		}
		if mimeType != "" {
			params.Set("type", mimeType)
		}

		data, err := d.Client.Get("/drive/search?" + params.Encode())
		if err != nil {
			continue
		}

		if globalFlags.json {
			var resp struct {
				Items []json.RawMessage `json:"items"`
			}
			json.Unmarshal(data, &resp)
			allResults = append(allResults, resp.Items...)
		} else {
			var resp struct {
				Items []objectInfo `json:"items"`
			}
			json.Unmarshal(data, &resp)
			for _, o := range resp.Items {
				fmt.Printf("%-12s %-30s %10s  %s\n", b.Name, o.Path, HumanSize(o.Size), o.ContentType)
			}
		}
	}

	if globalFlags.json {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(allResults)
	}

	return nil
}
