package git

import (
	"context"
	"path/filepath"
	"strings"
)

// LanguageStats represents language breakdown for a repository
type LanguageStats struct {
	Name       string  `json:"name"`
	Color      string  `json:"color"`
	Percentage float64 `json:"percentage"`
	Bytes      int64   `json:"bytes"`
}

// languageColors maps languages to their GitHub colors
var languageColors = map[string]string{
	"Go":          "#00ADD8",
	"JavaScript":  "#f1e05a",
	"TypeScript":  "#3178c6",
	"Python":      "#3572A5",
	"Rust":        "#dea584",
	"Java":        "#b07219",
	"C":           "#555555",
	"C++":         "#f34b7d",
	"C#":          "#178600",
	"Ruby":        "#701516",
	"PHP":         "#4F5D95",
	"Swift":       "#F05138",
	"Kotlin":      "#A97BFF",
	"Scala":       "#c22d40",
	"Dart":        "#00B4AB",
	"HTML":        "#e34c26",
	"CSS":         "#563d7c",
	"SCSS":        "#c6538c",
	"Less":        "#1d365d",
	"Vue":         "#41b883",
	"Svelte":      "#ff3e00",
	"Shell":       "#89e051",
	"Bash":        "#89e051",
	"PowerShell":  "#012456",
	"Makefile":    "#427819",
	"Dockerfile":  "#384d54",
	"YAML":        "#cb171e",
	"JSON":        "#292929",
	"TOML":        "#9c4221",
	"XML":         "#0060ac",
	"Markdown":    "#083fa1",
	"SQL":         "#e38c00",
	"GraphQL":     "#e10098",
	"Lua":         "#000080",
	"Perl":        "#0298c3",
	"R":           "#198CE7",
	"Julia":       "#a270ba",
	"Haskell":     "#5e5086",
	"Elixir":      "#6e4a7e",
	"Erlang":      "#B83998",
	"Clojure":     "#db5855",
	"OCaml":       "#3be133",
	"F#":          "#b845fc",
	"Objective-C": "#438eff",
	"Assembly":    "#6E4C13",
	"Vim Script":  "#199f4b",
	"Zig":         "#ec915c",
	"Nix":         "#7e7eff",
	"Terraform":   "#7B42BC",
	"HCL":         "#844FBA",
	"Protobuf":    "#FEAD00",
}

// extensionToLanguage maps file extensions to language names
var extensionToLanguage = map[string]string{
	".go":          "Go",
	".js":          "JavaScript",
	".mjs":         "JavaScript",
	".cjs":         "JavaScript",
	".jsx":         "JavaScript",
	".ts":          "TypeScript",
	".tsx":         "TypeScript",
	".py":          "Python",
	".pyw":         "Python",
	".pyx":         "Python",
	".rs":          "Rust",
	".java":        "Java",
	".c":           "C",
	".h":           "C",
	".cpp":         "C++",
	".cc":          "C++",
	".cxx":         "C++",
	".hpp":         "C++",
	".hxx":         "C++",
	".cs":          "C#",
	".rb":          "Ruby",
	".erb":         "Ruby",
	".php":         "PHP",
	".swift":       "Swift",
	".kt":          "Kotlin",
	".kts":         "Kotlin",
	".scala":       "Scala",
	".dart":        "Dart",
	".html":        "HTML",
	".htm":         "HTML",
	".css":         "CSS",
	".scss":        "SCSS",
	".sass":        "SCSS",
	".less":        "Less",
	".vue":         "Vue",
	".svelte":      "Svelte",
	".sh":          "Shell",
	".bash":        "Bash",
	".zsh":         "Shell",
	".fish":        "Shell",
	".ps1":         "PowerShell",
	".psm1":        "PowerShell",
	".yaml":        "YAML",
	".yml":         "YAML",
	".json":        "JSON",
	".toml":        "TOML",
	".xml":         "XML",
	".md":          "Markdown",
	".markdown":    "Markdown",
	".sql":         "SQL",
	".graphql":     "GraphQL",
	".gql":         "GraphQL",
	".lua":         "Lua",
	".pl":          "Perl",
	".pm":          "Perl",
	".r":           "R",
	".R":           "R",
	".jl":          "Julia",
	".hs":          "Haskell",
	".lhs":         "Haskell",
	".ex":          "Elixir",
	".exs":         "Elixir",
	".erl":         "Erlang",
	".hrl":         "Erlang",
	".clj":         "Clojure",
	".cljs":        "Clojure",
	".cljc":        "Clojure",
	".ml":          "OCaml",
	".mli":         "OCaml",
	".fs":          "F#",
	".fsi":         "F#",
	".fsx":         "F#",
	".m":           "Objective-C",
	".mm":          "Objective-C",
	".s":           "Assembly",
	".asm":         "Assembly",
	".vim":         "Vim Script",
	".zig":         "Zig",
	".nix":         "Nix",
	".tf":          "Terraform",
	".hcl":         "HCL",
	".proto":       "Protobuf",
}

// filenameToLanguage maps specific filenames to languages
var filenameToLanguage = map[string]string{
	"Makefile":    "Makefile",
	"makefile":    "Makefile",
	"GNUmakefile": "Makefile",
	"Dockerfile":  "Dockerfile",
	"dockerfile":  "Dockerfile",
	".bashrc":     "Bash",
	".zshrc":      "Shell",
	".profile":    "Shell",
	"go.mod":      "Go",
	"go.sum":      "Go",
	"Cargo.toml":  "TOML",
	"Cargo.lock":  "TOML",
	"package.json": "JSON",
	"tsconfig.json": "JSON",
	".gitignore":   "Git Config",
	".gitattributes": "Git Config",
}

// DetectLanguage returns the programming language for a filename
func DetectLanguage(filename string) string {
	// Check exact filename first
	if lang, ok := filenameToLanguage[filename]; ok {
		return lang
	}

	// Check by extension
	ext := strings.ToLower(filepath.Ext(filename))
	if lang, ok := extensionToLanguage[ext]; ok {
		return lang
	}

	return ""
}

// LanguageColor returns the color associated with a language
func LanguageColor(language string) string {
	if color, ok := languageColors[language]; ok {
		return color
	}
	return "#808080" // Default gray
}

// GetLanguageStats calculates language statistics for a repository
func (r *Repository) GetLanguageStats(ctx context.Context, ref string) ([]*LanguageStats, error) {
	tree, err := r.GetTreeRecursive(ctx, ref)
	if err != nil {
		return nil, err
	}

	// Count bytes per language
	langBytes := make(map[string]int64)
	var totalBytes int64

	for _, entry := range tree.Entries {
		if entry.Type != "blob" {
			continue
		}

		lang := DetectLanguage(entry.Name)
		if lang == "" {
			continue
		}

		langBytes[lang] += entry.Size
		totalBytes += entry.Size
	}

	if totalBytes == 0 {
		return nil, nil
	}

	// Convert to stats
	stats := make([]*LanguageStats, 0, len(langBytes))
	for lang, bytes := range langBytes {
		stats = append(stats, &LanguageStats{
			Name:       lang,
			Color:      LanguageColor(lang),
			Bytes:      bytes,
			Percentage: float64(bytes) / float64(totalBytes) * 100,
		})
	}

	// Sort by bytes descending
	sortLanguageStats(stats)

	return stats, nil
}

// GetPrimaryLanguage returns the primary (most used) language in a repository
func (r *Repository) GetPrimaryLanguage(ctx context.Context, ref string) (string, string, error) {
	stats, err := r.GetLanguageStats(ctx, ref)
	if err != nil || len(stats) == 0 {
		return "", "", err
	}

	return stats[0].Name, stats[0].Color, nil
}

// sortLanguageStats sorts by bytes descending
func sortLanguageStats(stats []*LanguageStats) {
	n := len(stats)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if stats[j].Bytes < stats[j+1].Bytes {
				stats[j], stats[j+1] = stats[j+1], stats[j]
			}
		}
	}
}
