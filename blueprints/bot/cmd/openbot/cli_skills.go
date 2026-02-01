package main

import (
	"fmt"
	"os"
	"os/exec"
	"text/tabwriter"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/skill"
)

// runSkills dispatches skills subcommands: list, info, check
func runSkills() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: openbot skills <list|info|check>")
	}

	switch os.Args[2] {
	case "list":
		return runSkillsList()
	case "info":
		return runSkillsInfo()
	case "check":
		return runSkillsCheck()
	default:
		return fmt.Errorf("unknown skills subcommand: %s", os.Args[2])
	}
}

func runSkillsList() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	skills, err := skill.LoadAllSkills(cfg.Workspace)
	if err != nil {
		return fmt.Errorf("load skills: %w", err)
	}

	if len(skills) == 0 {
		fmt.Println("No skills found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSOURCE\tREADY\tDESCRIPTION")
	for _, s := range skills {
		ready := "yes"
		if !s.Ready {
			ready = "no"
		}
		desc := s.Description
		if len(desc) > 60 {
			desc = desc[:60] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, s.Source, ready, desc)
	}
	w.Flush()
	return nil
}

func runSkillsInfo() error {
	if len(os.Args) < 4 {
		return fmt.Errorf("usage: openbot skills info <name>")
	}
	name := os.Args[3]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	skills, err := skill.LoadAllSkills(cfg.Workspace)
	if err != nil {
		return fmt.Errorf("load skills: %w", err)
	}

	for _, s := range skills {
		if s.Name == name {
			fmt.Printf("Name:        %s\n", s.Name)
			fmt.Printf("Source:      %s\n", s.Source)
			fmt.Printf("Dir:         %s\n", s.Dir)
			ready := "yes"
			if !s.Ready {
				ready = "no"
			}
			fmt.Printf("Ready:       %s\n", ready)
			fmt.Printf("Description: %s\n", s.Description)
			if len(s.Requires.Binaries) > 0 {
				fmt.Printf("Requires binaries: %v\n", s.Requires.Binaries)
			}
			if len(s.Requires.Config) > 0 {
				fmt.Printf("Requires config:   %v\n", s.Requires.Config)
			}
			return nil
		}
	}

	return fmt.Errorf("skill not found: %s", name)
}

func runSkillsCheck() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	skills, err := skill.LoadAllSkills(cfg.Workspace)
	if err != nil {
		return fmt.Errorf("load skills: %w", err)
	}

	if len(skills) == 0 {
		fmt.Println("No skills found.")
		return nil
	}

	for _, s := range skills {
		status := "ready"
		if !s.Ready {
			status = "not-ready"
		}
		fmt.Printf("%s: %s\n", s.Name, status)

		// Show which requirements are missing.
		for _, bin := range s.Requires.Binaries {
			if _, err := exec.LookPath(bin); err != nil {
				fmt.Printf("  missing binary: %s\n", bin)
			}
		}
		for _, key := range s.Requires.Config {
			if os.Getenv(key) == "" {
				fmt.Printf("  missing config: %s\n", key)
			}
		}
	}
	return nil
}
