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
	outDir := flags.String("out", "out_markdown", "Output directory for markdown files")
	assetsDir := flags.String("assets", "assets", "Per-folder assets directory name")
	overwrite := flags.Bool("overwrite", false, "Overwrite existing markdown and attachment files")
	if err := flags.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(*inPath) == "" {
		flags.Usage()
		return ErrMissingInput
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

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	cv := newConverter(export, *outDir, *assetsDir, *overwrite)
	written := 0
	for i, c := range export.History {
		ok, err := cv.convertConversation(i, c)
		if err != nil {
			return fmt.Errorf("conversation %d: %w", i, err)
		}
		if ok {
			written++
		}
	}

	_, err = fmt.Fprintf(stdout, "Done. Wrote %d of %d markdown files to %s\nAttachments written under per-folder %s directories\n",
		written, len(export.History), *outDir, *assetsDir)
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
