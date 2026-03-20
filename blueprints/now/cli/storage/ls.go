package storage

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newLsCmd() *cobra.Command {
	var (
		limit  int
		search string
	)

	cmd := &cobra.Command{
		Use:     "ls [bucket] [prefix]",
		Short:   "List buckets or objects",
		Aliases: []string{"list"},
		Example: `  storage ls
  storage ls docs
  storage ls docs reports/
  storage ls --json docs | jq '.[].path'`,
		Run: wrapRun(func(cmd *cobra.Command, args []string) error {
			d := deps()
			if err := RequireToken(d.Config); err != nil {
				return err
			}

			if len(args) == 0 {
				return listBuckets(d)
			}

			bucket := args[0]
			prefix := ""
			if len(args) > 1 {
				prefix = args[1]
			}
			return listObjects(d, bucket, prefix, limit, search)
		}),
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 100, "max results")
	cmd.Flags().StringVarP(&search, "search", "s", "", "search filter")
	return cmd
}

type bucketInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Public      any    `json:"public"`
	ObjectCount int    `json:"object_count"`
	TotalSize   int64  `json:"total_size"`
	CreatedAt   int64  `json:"created_at"`
}

func (b bucketInfo) IsPublic() bool {
	switch v := b.Public.(type) {
	case bool:
		return v
	case float64:
		return v == 1
	default:
		return false
	}
}

func listBuckets(d *Deps) error {
	data, err := d.Client.Get("/bucket")
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

	var buckets []bucketInfo
	if err := json.Unmarshal(data, &buckets); err != nil {
		return err
	}

	if len(buckets) == 0 {
		d.Out.Info("No buckets", "Create one with: storage bucket create <name>")
		return nil
	}

	fmt.Printf("%-16s %6s %10s  %-6s  %s\n", "NAME", "FILES", "SIZE", "PUBLIC", "CREATED")
	for _, b := range buckets {
		pub := "no"
		if b.IsPublic() {
			pub = "yes"
		}
		fmt.Printf("%-16s %6d %10s  %-6s  %s\n",
			b.Name, b.ObjectCount, HumanSize(b.TotalSize), pub, RelativeTime(b.CreatedAt))
	}
	return nil
}

type objectInfo struct {
	ID          string `json:"id"`
	Path        string `json:"path"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

func listObjects(d *Deps, bucket, prefix string, limit int, search string) error {
	body := map[string]any{
		"limit":   limit,
		"sort_by": map[string]string{"column": "name", "order": "asc"},
	}
	if prefix != "" {
		body["prefix"] = prefix
	}
	if search != "" {
		body["search"] = search
	}

	data, err := d.Client.DoJSON("POST", "/object/list/"+bucket, body)
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

	var objects []objectInfo
	if err := json.Unmarshal(data, &objects); err != nil {
		return err
	}

	if len(objects) == 0 {
		label := bucket
		if prefix != "" {
			label += "/" + prefix
		}
		d.Out.Info("Empty", "No objects in "+label)
		return nil
	}

	fmt.Printf("%-30s %10s  %-24s  %s\n", "PATH", "SIZE", "TYPE", "MODIFIED")
	for _, o := range objects {
		fmt.Printf("%-30s %10s  %-24s  %s\n",
			o.Path, HumanSize(o.Size), o.ContentType, RelativeTime(o.UpdatedAt))
	}
	return nil
}
