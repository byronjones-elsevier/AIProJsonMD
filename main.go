// Command AIProJsonMD converts an AI Pro chat export JSON file into Markdown
// files grouped by export folder, with attachments extracted alongside.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// ErrMissingInput is returned when the required -in flag is not provided.
var ErrMissingInput = errors.New("missing required -in flag")

// ErrNoHistory is returned when the export contains no conversations.
var ErrNoHistory = errors.New("no history entries found")

// run parses args, converts the export, and writes a summary to stdout.
func run(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("AIProJsonMD", flag.ContinueOnError)
	inPath := flags.String("in", "", "Path to input JSON file")
	outDir := flags.String("out", "out_markdown", "Output directory for generated files")
	assetsDir := flags.String("assets", "assets", "Per-folder assets directory name")
	formatFlag := flags.String("format", string(formatMarkdown), "Output format: md, html, or both")
	overwrite := flags.Bool("overwrite", false, "Overwrite existing output and attachment files")
	if err := flags.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(*inPath) == "" {
		flags.Usage()
		return ErrMissingInput
	}
	if err := validateAssetsDir(*assetsDir); err != nil {
		return err
	}
	format, err := parseFormat(*formatFlag)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(*inPath)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	var export Export
	if err := json.Unmarshal(data, &export); err != nil {
		return fmt.Errorf("parse JSON: %w", err)
	}
	if len(export.History) == 0 {
		return ErrNoHistory
	}

	if err := os.MkdirAll(*outDir, dirPerm); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	cv, err := newConverter(export, *outDir, *assetsDir, *overwrite, format)
	if err != nil {
		return err
	}

	var markdownWritten, htmlWritten int
	for i, c := range export.History {
		res, err := cv.convertConversation(i, c)
		if err != nil {
			return fmt.Errorf("conversation %d: %w", i, err)
		}
		if res.markdown {
			markdownWritten++
		}
		if res.html {
			htmlWritten++
		}
	}

	indexes, err := cv.writeIndexes()
	if err != nil {
		return err
	}

	total := len(export.History)
	lines := []string{}
	if format.writesMarkdown() {
		lines = append(lines, fmt.Sprintf("Wrote %d of %d markdown files to %s", markdownWritten, total, *outDir))
	}
	if format.writesHTML() {
		index := "kept"
		if indexes.main {
			index = "written"
		}
		lines = append(lines, fmt.Sprintf("Wrote %d of %d HTML files to %s (index.html %s, %d of %d folder index pages written)",
			htmlWritten, total, *outDir, index, indexes.foldersWritten, indexes.foldersTotal))
	}
	lines[0] = "Done. " + lines[0]
	lines = append(lines, fmt.Sprintf("Attachments written under per-folder %s directories", *assetsDir))

	_, err = fmt.Fprintln(stdout, strings.Join(lines, "\n"))
	return err
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		slog.Error("conversion failed", "error", err)
		os.Exit(1)
	}
}
