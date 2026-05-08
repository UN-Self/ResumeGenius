package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/UN-Self/ResumeGenius/backend/internal/modules/designskill"
)

type options struct {
	Domain     string
	Stack      string
	MaxResults int
	JSONOut    bool
	Query      string
}

func main() {
	opts := parseArgs(os.Args[1:])
	if opts.Query == "" {
		fail("query is required")
	}

	result, err := designskill.SearchSkill(opts.Query, opts.Domain, opts.Stack, opts.MaxResults)
	if err != nil {
		fail(err.Error())
	}

	if opts.JSONOut {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(result)
		return
	}
	printMarkdown(result)
}

func parseArgs(args []string) options {
	opts := options{MaxResults: 3}
	queryParts := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		key, value, hasInlineValue := strings.Cut(arg, "=")
		switch key {
		case "--domain":
			opts.Domain = readFlagValue(args, &i, value, hasInlineValue, key)
		case "--stack":
			opts.Stack = readFlagValue(args, &i, value, hasInlineValue, key)
		case "--max":
			raw := readFlagValue(args, &i, value, hasInlineValue, key)
			var parsed int
			if _, err := fmt.Sscanf(raw, "%d", &parsed); err != nil {
				fail("--max must be a number")
			}
			opts.MaxResults = parsed
		case "--json":
			opts.JSONOut = true
		case "-h", "--help":
			printUsage()
			os.Exit(0)
		default:
			queryParts = append(queryParts, arg)
		}
	}

	opts.Query = strings.TrimSpace(strings.Join(queryParts, " "))
	return opts
}

func readFlagValue(args []string, index *int, inlineValue string, hasInlineValue bool, key string) string {
	if hasInlineValue {
		return inlineValue
	}
	if *index+1 >= len(args) || strings.HasPrefix(args[*index+1], "--") {
		fail(key + " requires a value")
	}
	(*index)++
	return args[*index]
}

func printUsage() {
	fmt.Println("Usage: go run ./cmd/uxsearch [query] [--domain style|prompt|color|chart|landing|product|ux|typography] [--stack react|vue|nextjs|...] [--max 3] [--json]")
}

func printMarkdown(res designskill.SkillSearchResult) {
	if res.Stack != "" {
		fmt.Printf("## UI/UX Pro Max Stack Guidelines\n\n**Stack:** %s | **Query:** %s\n", res.Stack, res.Query)
	} else {
		fmt.Printf("## UI/UX Pro Max Search Results\n\n**Domain:** %s | **Query:** %s\n", res.Domain, res.Query)
	}
	fmt.Printf("**Source:** %s | **Found:** %d results\n\n", res.File, res.Count)
	for i, row := range res.Results {
		fmt.Printf("### Result %d\n", i+1)
		for key, value := range row {
			if len([]rune(value)) > 300 {
				value = string([]rune(value)[:300]) + "..."
			}
			fmt.Printf("- **%s:** %s\n", key, value)
		}
		fmt.Println()
	}
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, "Error:", message)
	os.Exit(1)
}
