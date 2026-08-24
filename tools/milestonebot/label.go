package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// LabelFile is .github/labels.yml.
//
// Labels are grouped so that a group can set the colour once. A label may
// override the group's colour, which the loose group at the end of the file
// does for every one of its labels, and a group with an empty prefix imposes
// no naming rule.
type LabelFile struct {
	Groups []LabelGroup `yaml:"groups"`
}

type LabelGroup struct {
	Prefix string     `yaml:"prefix"`
	Color  string     `yaml:"color"`
	Labels []LabelDef `yaml:"labels"`
}

type LabelDef struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color"`
	Description string `yaml:"description"`
}

// LoadLabels reads and validates the label file. Validation is part of loading
// because a label file with two definitions of the same name is a merge
// accident, and finding it here beats finding it as a confusing API error
// halfway through a sync.
func LoadLabels(path string) ([]Label, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f LabelFile
	dec := yaml.NewDecoder(strings.NewReader(string(b)))
	dec.KnownFields(true)
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	seen := map[string]bool{}
	var out []Label
	for _, g := range f.Groups {
		for _, d := range g.Labels {
			switch {
			case d.Name == "":
				return nil, fmt.Errorf("%s: a label in group %q has no name", path, g.Prefix)
			case seen[d.Name]:
				return nil, fmt.Errorf("%s: %s is defined twice", path, d.Name)
			case d.Description == "":
				return nil, fmt.Errorf("%s: %s has no description, and a label nobody can explain is a label nobody applies consistently", path, d.Name)
			case len(d.Description) > 100:
				return nil, fmt.Errorf("%s: %s has a %d character description, and GitHub truncates at 100", path, d.Name, len(d.Description))
			}
			if g.Prefix != "" && !strings.HasPrefix(d.Name, g.Prefix+"/") {
				return nil, fmt.Errorf("%s: %s is in the %q group so it must be named %s/something", path, d.Name, g.Prefix, g.Prefix)
			}
			color := d.Color
			if color == "" {
				color = g.Color
			}
			if len(color) != 6 {
				return nil, fmt.Errorf("%s: %s has colour %q, which is not six hex digits", path, d.Name, color)
			}
			seen[d.Name] = true
			out = append(out, Label{Name: d.Name, Color: color, Description: d.Description})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: no labels", path)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// SyncLabels reconciles the repository against the file.
//
// It creates and updates. It does not delete, and it reports what it found on
// the repository that the file does not describe, because a label somebody is
// using on 40 issues is not this tool's to remove on the strength of a diff.
// Removing one is a pull request against the file plus a deliberate `gh label
// delete`, and that friction is the point.
func SyncLabels(ctx context.Context, c *Client, path string) error {
	want, err := LoadLabels(path)
	if err != nil {
		return err
	}
	have, err := c.Labels(ctx)
	if err != nil {
		return err
	}
	byName := make(map[string]Label, len(have))
	for _, l := range have {
		byName[l.Name] = l
	}

	var created, updated int
	for _, w := range want {
		cur, ok := byName[w.Name]
		switch {
		case !ok:
			if err := c.CreateLabel(ctx, w); err != nil {
				return fmt.Errorf("create %s: %w", w.Name, err)
			}
			fmt.Printf("created  %-24s #%s  %s\n", w.Name, w.Color, w.Description)
			created++
		case !strings.EqualFold(cur.Color, w.Color) || cur.Description != w.Description:
			if err := c.UpdateLabel(ctx, w); err != nil {
				return fmt.Errorf("update %s: %w", w.Name, err)
			}
			fmt.Printf("updated  %-24s #%s  %s\n", w.Name, w.Color, w.Description)
			updated++
		}
	}

	wanted := make(map[string]bool, len(want))
	for _, w := range want {
		wanted[w.Name] = true
	}
	var extra []string
	for _, h := range have {
		if !wanted[h.Name] {
			extra = append(extra, h.Name)
		}
	}
	sort.Strings(extra)

	fmt.Printf("\n%d labels in %s, %d created, %d updated\n", len(want), path, created, updated)
	if len(extra) > 0 {
		fmt.Printf("\nOn the repository but not in the file: %s\n", strings.Join(extra, ", "))
		fmt.Println("Nothing was deleted. Add each one to the file, or delete it deliberately with gh label delete.")
	}
	return nil
}
