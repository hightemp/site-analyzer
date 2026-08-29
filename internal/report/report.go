// Package report renders scan results in machine- and human-readable formats.
package report

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"site-analyzer/internal/model"
)

// Formats lists supported output formats for help and validation messages.
const Formats = "jsonl, json, csv, table, markdown"

// Write serializes all results in the requested format.
func Write(writer io.Writer, format string, results []model.Result) error {
	switch strings.ToLower(format) {
	case "jsonl", "ndjson":
		return writeJSONL(writer, results)
	case "json":
		return writeJSON(writer, results)
	case "csv":
		return writeCSV(writer, results)
	case "table":
		return writeTable(writer, results)
	case "markdown", "md":
		return writeMarkdown(writer, results)
	default:
		return fmt.Errorf("unsupported report format %q (supported: %s)", format, Formats)
	}
}

func writeJSONL(writer io.Writer, results []model.Result) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	for _, result := range results {
		if err := encoder.Encode(result); err != nil {
			return fmt.Errorf("encode JSONL result: %w", err)
		}
	}
	return nil
}

func writeJSON(writer io.Writer, results []model.Result) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		return fmt.Errorf("encode JSON report: %w", err)
	}
	return nil
}

func writeCSV(writer io.Writer, results []model.Result) error {
	csvWriter := csv.NewWriter(writer)
	header := []string{"input_url", "url", "final_url", "status_code", "title", "technologies", "server", "content_type", "body_bytes", "truncated", "duration_ms", "scanned_at", "error"}
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("write CSV header: %w", err)
	}
	for _, result := range results {
		record := []string{
			result.InputURL,
			result.URL,
			result.FinalURL,
			strconv.Itoa(result.StatusCode),
			result.Title,
			technologyNames(result.Technologies),
			result.Server,
			result.ContentType,
			strconv.FormatInt(result.BodyBytes, 10),
			strconv.FormatBool(result.Truncated),
			strconv.FormatInt(result.DurationMS, 10),
			result.ScannedAt.Format("2006-01-02T15:04:05.999999999Z07:00"),
			result.Error,
		}
		if err := csvWriter.Write(record); err != nil {
			return fmt.Errorf("write CSV record for %q: %w", result.InputURL, err)
		}
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("flush CSV report: %w", err)
	}
	return nil
}

func writeTable(writer io.Writer, results []model.Result) error {
	table := tabwriter.NewWriter(writer, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "STATUS\tURL\tTITLE\tTECHNOLOGIES\tERROR"); err != nil {
		return fmt.Errorf("write table header: %w", err)
	}
	for _, result := range results {
		status := "-"
		if result.StatusCode != 0 {
			status = strconv.Itoa(result.StatusCode)
		}
		if _, err := fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n",
			status,
			oneLine(displayURL(result)),
			oneLine(result.Title),
			oneLine(technologyNames(result.Technologies)),
			oneLine(result.Error),
		); err != nil {
			return fmt.Errorf("write table row for %q: %w", result.InputURL, err)
		}
	}
	if err := table.Flush(); err != nil {
		return fmt.Errorf("flush table report: %w", err)
	}
	return nil
}

func writeMarkdown(writer io.Writer, results []model.Result) error {
	if _, err := fmt.Fprintln(writer, "| Status | URL | Title | Technologies | Error |\n|---:|---|---|---|---|"); err != nil {
		return fmt.Errorf("write Markdown header: %w", err)
	}
	for _, result := range results {
		status := "—"
		if result.StatusCode != 0 {
			status = strconv.Itoa(result.StatusCode)
		}
		if _, err := fmt.Fprintf(writer, "| %s | %s | %s | %s | %s |\n",
			markdownCell(status),
			markdownCell(displayURL(result)),
			markdownCell(result.Title),
			markdownCell(technologyNames(result.Technologies)),
			markdownCell(result.Error),
		); err != nil {
			return fmt.Errorf("write Markdown row for %q: %w", result.InputURL, err)
		}
	}
	return nil
}

func technologyNames(technologies []model.Technology) string {
	values := make([]string, 0, len(technologies))
	for _, technology := range technologies {
		name := technology.Name
		if technology.Version != "" {
			name += " " + technology.Version
		}
		values = append(values, name)
	}
	return strings.Join(values, ", ")
}

func displayURL(result model.Result) string {
	if result.FinalURL != "" {
		return result.FinalURL
	}
	if result.URL != "" {
		return result.URL
	}
	return result.InputURL
}

func oneLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func markdownCell(value string) string {
	return strings.ReplaceAll(oneLine(value), "|", "\\|")
}
