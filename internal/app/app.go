// Package app implements the site-analyzer command-line application.
package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"site-analyzer/internal/model"
	"site-analyzer/internal/report"
	"site-analyzer/internal/runner"
	"site-analyzer/internal/scanner"
	"site-analyzer/internal/target"
)

const version = "dev"

type stringList []string

func (values *stringList) String() string {
	return strings.Join(*values, ",")
}

func (values *stringList) Set(value string) error {
	*values = append(*values, value)
	return nil
}

type headerList http.Header

func (headers *headerList) String() string {
	return fmt.Sprint(http.Header(*headers))
}

func (headers *headerList) Set(value string) error {
	name, headerValue, ok := strings.Cut(value, ":")
	if !ok || strings.TrimSpace(name) == "" {
		return fmt.Errorf("header must have the form 'Name: value'")
	}
	if *headers == nil {
		*headers = headerList(make(http.Header))
	}
	http.Header(*headers).Add(strings.TrimSpace(name), strings.TrimSpace(headerValue))
	return nil
}

// Run parses CLI options, scans targets, writes a report, and returns an exit code.
func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("site-analyzer", flag.ContinueOnError)
	flags.SetOutput(stderr)

	var urls stringList
	var headers headerList
	var listPath, outputPath, format, userAgent string
	var concurrency int
	var timeout time.Duration
	var maxBody int64
	var insecure, noRedirect, quiet, showVersion bool

	flags.Var(&urls, "u", "target URL (repeatable)")
	flags.Var(&urls, "url", "target URL (repeatable)")
	flags.StringVar(&listPath, "l", "", "TXT file with one target per line; - reads stdin")
	flags.StringVar(&listPath, "list", "", "TXT file with one target per line; - reads stdin")
	flags.StringVar(&format, "f", "jsonl", "report format: "+report.Formats)
	flags.StringVar(&format, "format", "jsonl", "report format: "+report.Formats)
	flags.StringVar(&outputPath, "o", "", "output file (default: stdout)")
	flags.StringVar(&outputPath, "output", "", "output file (default: stdout)")
	flags.IntVar(&concurrency, "c", 10, "number of concurrent requests")
	flags.IntVar(&concurrency, "concurrency", 10, "number of concurrent requests")
	flags.DurationVar(&timeout, "timeout", 15*time.Second, "timeout per target")
	flags.Int64Var(&maxBody, "max-body", 5*1024*1024, "maximum response body bytes")
	flags.StringVar(&userAgent, "user-agent", "site-analyzer/1.0", "HTTP User-Agent")
	flags.Var(&headers, "H", "additional HTTP header, 'Name: value' (repeatable)")
	flags.BoolVar(&insecure, "insecure", false, "skip TLS certificate verification")
	flags.BoolVar(&noRedirect, "no-redirect", false, "do not follow HTTP redirects")
	flags.BoolVar(&quiet, "quiet", false, "do not print the scan summary to stderr")
	flags.BoolVar(&showVersion, "version", false, "print version and exit")
	flags.Usage = func() {
		writef(stderr, "Usage: site-analyzer [options] [URL ...]\n\n")
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if showVersion {
		writef(stdout, "site-analyzer %s\n", version)
		return 0
	}
	if concurrency <= 0 {
		writef(stderr, "error: concurrency must be greater than zero\n")
		return 2
	}
	if timeout <= 0 {
		writef(stderr, "error: timeout must be greater than zero\n")
		return 2
	}
	if maxBody <= 0 {
		writef(stderr, "error: max-body must be greater than zero\n")
		return 2
	}
	if err := report.Write(io.Discard, format, nil); err != nil {
		writef(stderr, "error: %v\n", err)
		return 2
	}

	targets, err := target.Load(urls, flags.Args(), listPath, stdin)
	if err != nil {
		writef(stderr, "error: %v\n", err)
		return 1
	}
	if len(targets) == 0 {
		writef(stderr, "error: no targets; use -u URL, -l targets.txt, or positional URLs\n")
		return 2
	}

	siteScanner, err := scanner.New(scanner.Options{
		Timeout:         timeout,
		MaxBodyBytes:    maxBody,
		UserAgent:       userAgent,
		Insecure:        insecure,
		FollowRedirects: !noRedirect,
		Headers:         http.Header(headers),
	})
	if err != nil {
		writef(stderr, "error: %v\n", err)
		return 1
	}
	batch, err := runner.New(siteScanner, concurrency)
	if err != nil {
		writef(stderr, "error: %v\n", err)
		return 1
	}

	started := time.Now()
	results, runErr := batch.Run(ctx, targets)
	if runErr != nil {
		writef(stderr, "error: %v\n", runErr)
	}

	if err := writeReport(outputPath, format, results, stdout); err != nil {
		writef(stderr, "error: %v\n", err)
		return 1
	}

	failures, technologies := 0, 0
	for _, result := range results {
		if result.Error != "" {
			failures++
		}
		technologies += len(result.Technologies)
	}
	if !quiet {
		writef(stderr, "scanned=%d failed=%d technologies=%d duration=%s\n",
			len(results), failures, technologies, time.Since(started).Round(time.Millisecond))
	}
	if runErr != nil || failures > 0 {
		return 1
	}
	return 0
}

func writef(writer io.Writer, format string, values ...any) {
	if _, err := fmt.Fprintf(writer, format, values...); err != nil {
		return
	}
}

func writeReport(path, format string, results []model.Result, stdout io.Writer) error {
	if path == "" || path == "-" {
		if err := report.Write(stdout, format, results); err != nil {
			return fmt.Errorf("write report: %w", err)
		}
		return nil
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create report %q: %w", path, err)
	}
	writeErr := report.Write(file, format, results)
	closeErr := file.Close()
	if writeErr != nil {
		return fmt.Errorf("write report %q: %w", path, writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close report %q: %w", path, closeErr)
	}
	return nil
}
