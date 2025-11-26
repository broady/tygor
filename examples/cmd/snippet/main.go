// Command snippet extracts marked code snippets from source files and outputs them as markdown.
//
// Usage:
//
//	snippet [flags] <file>...
//	  -name string    Extract specific snippet (default: all)
//	  -lang string    Override language detection
//	  -format string  Output format: simple|mdx (default: mdx)
//	  -out string     Output file (default: stdout)
//
// Snippet markers in source files:
//
//	// [snippet:example-name]
//	func Example() {
//	    // code here
//	}
//	// [/snippet:example-name]
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	nameFlag   = flag.String("name", "", "Extract specific snippet (default: all)")
	langFlag   = flag.String("lang", "", "Override language detection")
	formatFlag = flag.String("format", "mdx", "Output format: simple|mdx")
	outFlag    = flag.String("out", "", "Output file (default: stdout)")
)

// Snippet represents an extracted code snippet.
type Snippet struct {
	Name      string
	File      string
	StartLine int
	EndLine   int
	Content   string
	Lang      string
}

var (
	// Matches: // [snippet:name] or # [snippet:name]
	startPattern = regexp.MustCompile(`(?://|#)\s*\[snippet:([^\]]+)\]`)
	// Matches: // [/snippet:name] or # [/snippet:name]
	endPattern = regexp.MustCompile(`(?://|#)\s*\[/snippet:([^\]]+)\]`)
)

func main() {
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Usage: snippet [flags] <file>...")
		flag.PrintDefaults()
		os.Exit(1)
	}

	var allSnippets []Snippet

	for _, file := range flag.Args() {
		snippets, err := extractSnippets(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", file, err)
			os.Exit(1)
		}
		allSnippets = append(allSnippets, snippets...)
	}

	// Filter by name if specified
	if *nameFlag != "" {
		var filtered []Snippet
		for _, s := range allSnippets {
			if s.Name == *nameFlag {
				filtered = append(filtered, s)
			}
		}
		allSnippets = filtered
	}

	// Generate output
	output := formatSnippets(allSnippets, *formatFlag)

	// Write output
	if *outFlag != "" {
		if err := os.WriteFile(*outFlag, []byte(output), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Print(output)
	}
}

func extractSnippets(filename string) ([]Snippet, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var snippets []Snippet
	var current *Snippet
	var contentLines []string

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Check for start marker
		if matches := startPattern.FindStringSubmatch(line); matches != nil {
			if current != nil {
				return nil, fmt.Errorf("line %d: nested snippet %q inside %q", lineNum, matches[1], current.Name)
			}
			current = &Snippet{
				Name:      matches[1],
				File:      filepath.Base(filename),
				StartLine: lineNum + 1, // Content starts on next line
				Lang:      detectLang(filename),
			}
			contentLines = nil
			continue
		}

		// Check for end marker
		if matches := endPattern.FindStringSubmatch(line); matches != nil {
			if current == nil {
				return nil, fmt.Errorf("line %d: end marker for %q without start", lineNum, matches[1])
			}
			if matches[1] != current.Name {
				return nil, fmt.Errorf("line %d: end marker %q doesn't match start %q", lineNum, matches[1], current.Name)
			}
			current.EndLine = lineNum - 1
			current.Content = strings.Join(contentLines, "\n")
			snippets = append(snippets, *current)
			current = nil
			continue
		}

		// Collect content if inside a snippet
		if current != nil {
			contentLines = append(contentLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if current != nil {
		return nil, fmt.Errorf("unclosed snippet %q", current.Name)
	}

	return snippets, nil
}

func detectLang(filename string) string {
	if *langFlag != "" {
		return *langFlag
	}

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".go":
		return "go"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "tsx"
	case ".js":
		return "javascript"
	case ".jsx":
		return "jsx"
	case ".proto":
		return "protobuf"
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".md":
		return "markdown"
	case ".sh", ".bash":
		return "bash"
	default:
		return ""
	}
}

func formatSnippets(snippets []Snippet, format string) string {
	var sb strings.Builder

	for i, s := range snippets {
		if i > 0 {
			sb.WriteString("\n")
		}

		switch format {
		case "simple":
			// Simple format with comment header
			sb.WriteString(fmt.Sprintf("```%s\n", s.Lang))
			sb.WriteString(fmt.Sprintf("// %s:%d-%d\n", s.File, s.StartLine, s.EndLine))
			sb.WriteString(s.Content)
			sb.WriteString("\n```\n")

		case "mdx":
			// MDX-style with title attribute
			sb.WriteString(fmt.Sprintf("```%s title=\"%s:%d-%d\"\n", s.Lang, s.File, s.StartLine, s.EndLine))
			sb.WriteString(s.Content)
			sb.WriteString("\n```\n")

		default:
			// Default to mdx
			sb.WriteString(fmt.Sprintf("```%s title=\"%s:%d-%d\"\n", s.Lang, s.File, s.StartLine, s.EndLine))
			sb.WriteString(s.Content)
			sb.WriteString("\n```\n")
		}
	}

	return sb.String()
}
