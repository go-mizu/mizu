package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "milestonebot:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return fmt.Errorf("no command")
	}
	cmd, rest := args[0], args[1:]

	fs := flag.NewFlagSet("milestonebot "+cmd, flag.ContinueOnError)
	repo := fs.String("repo", os.Getenv("GITHUB_REPOSITORY"), "repository in owner/name form")
	dry := fs.Bool("n", false, "print what would change without changing it")
	var (
		file    *string
		pr      *int
		strict  *bool
		tracker *string
	)
	switch cmd {
	case "sync-labels":
		file = fs.String("file", ".github/labels.yml", "the label file")
	case "label-pr":
		pr = fs.Int("pr", 0, "pull request number")
		strict = fs.Bool("strict", false, "fail if the title has no recognised prefix")
	case "on-merge":
		pr = fs.Int("pr", 0, "pull request number")
		tracker = fs.String("tracking-repo", "", "where the tracking issues live, if not -repo")
	case "status":
		tracker = fs.String("tracking-repo", "", "where the tracking issues live, if not -repo")
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("unknown command %q", cmd)
	}
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if *repo == "" {
		return fmt.Errorf("no repository: pass -repo owner/name or set GITHUB_REPOSITORY")
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" && !*dry {
		return fmt.Errorf("no GITHUB_TOKEN: every command reads the API, and the write commands need a token that can write")
	}
	c := NewClient(*repo, token)
	c.DryRun = *dry

	// The tracking issues are usually in the same repository as the pull
	// request. They are not when the work happens somewhere else, which is
	// true of the site: its checklist lives in go-mizu/mizu and its pull
	// requests merge in go-mizu/docs.
	//
	// That case gets its own token if one is set. A workflow token only ever
	// reaches its own repository, and one token holding both grants is a
	// bigger key than either job needs.
	t := c
	if tracker != nil && *tracker != "" && *tracker != *repo {
		tt := os.Getenv("TRACKING_TOKEN")
		if tt == "" {
			tt = token
		}
		t = NewClient(*tracker, tt)
		t.DryRun = *dry
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	switch cmd {
	case "sync-labels":
		return SyncLabels(ctx, c, *file)
	case "label-pr":
		if *pr == 0 {
			return fmt.Errorf("label-pr needs -pr")
		}
		return LabelPR(ctx, c, *pr, *strict)
	case "on-merge":
		if *pr == 0 {
			return fmt.Errorf("on-merge needs -pr")
		}
		return OnMerge(ctx, c, t, *pr)
	case "status":
		return Status(ctx, t)
	}
	panic("unreachable: the command was validated above")
}

func usage() {
	fmt.Fprint(os.Stderr, `milestonebot keeps labels, milestone checklists, and pull requests in agreement.

Usage:
  milestonebot sync-labels [-file .github/labels.yml]
  milestonebot label-pr    -pr N [-strict]
  milestonebot on-merge    -pr N [-tracking-repo owner/name]
  milestonebot status      [-tracking-repo owner/name]

Common flags:
  -repo owner/name   defaults to GITHUB_REPOSITORY
  -n                 print what would change without changing it

Authentication is GITHUB_TOKEN.

Pass -tracking-repo when the checklist lives somewhere other than the
repository the pull request merged in. The tracking repository is then read and
written with TRACKING_TOKEN if that is set, and with GITHUB_TOKEN if it is not.
Two tokens is the point: a workflow token only reaches its own repository, and
one token holding both grants is a bigger key than either job needs.

A pull request names the checklist item it finishes with a trailer in its
description:

  Checklist: M0-03

on-merge finds the open issue carrying both `+"`tracking`"+` and `+"`milestone/M0`"+`,
ticks that box, and comments with what is left.
`)
}
