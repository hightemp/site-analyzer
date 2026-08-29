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

// GroupMode selects an optional report aggregation.
type GroupMode string

// Supported report grouping modes.
const (
	GroupNone GroupMode = ""
	GroupCMS  GroupMode = "cms"
)

// Options configures report rendering.
type Options struct {
	GroupBy GroupMode
}

// ParseGroupMode validates and normalizes a CLI grouping value.
func ParseGroupMode(value string) (GroupMode, error) {
	mode := GroupMode(strings.ToLower(strings.TrimSpace(value)))
	switch mode {
	case GroupNone, GroupCMS:
		return mode, nil
	default:
		return GroupNone, fmt.Errorf("unsupported group mode %q (supported: cms)", value)
	}
}

// Write serializes all results in the requested format.
func Write(writer io.Writer, format string, results []model.Result, options Options) error {
	if options.GroupBy == GroupCMS {
		return writeCMSGroups(writer, format, GroupByCMS(results))
	}
	if options.GroupBy != GroupNone {
		return fmt.Errorf("unsupported group mode %q (supported: cms)", options.GroupBy)
	}

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
	if err := csvWriter.Write(csvHeader()); err != nil {
		return fmt.Errorf("write CSV header: %w", err)
	}
	for _, result := range results {
		if err := csvWriter.Write(csvRecord(result)); err != nil {
			return fmt.Errorf("write CSV record for %q: %w", result.InputURL, err)
		}
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("flush CSV report: %w", err)
	}
	return nil
}

func writeCMSGroups(writer io.Writer, format string, groups []model.CMSGroup) error {
	switch strings.ToLower(format) {
	case "jsonl", "ndjson":
		return writeCMSJSONL(writer, groups)
	case "json":
		return writeCMSJSON(writer, groups)
	case "csv":
		return writeCMSCSV(writer, groups)
	case "table":
		return writeCMSTable(writer, groups)
	case "markdown", "md":
		return writeCMSMarkdown(writer, groups)
	default:
		return fmt.Errorf("unsupported report format %q (supported: %s)", format, Formats)
	}
}

func writeCMSJSONL(writer io.Writer, groups []model.CMSGroup) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	for _, group := range groups {
		if err := encoder.Encode(group); err != nil {
			return fmt.Errorf("encode grouped JSONL result for %q: %w", group.CMS, err)
		}
	}
	return nil
}

func writeCMSJSON(writer io.Writer, groups []model.CMSGroup) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(groups); err != nil {
		return fmt.Errorf("encode grouped JSON report: %w", err)
	}
	return nil
}

func writeCMSCSV(writer io.Writer, groups []model.CMSGroup) error {
	csvWriter := csv.NewWriter(writer)
	header := append([]string{"cms", "site_count"}, csvHeader()...)
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("write grouped CSV header: %w", err)
	}
	for _, group := range groups {
		for _, result := range group.Sites {
			record := append([]string{group.CMS, strconv.Itoa(group.SiteCount)}, csvRecord(result)...)
			if err := csvWriter.Write(record); err != nil {
				return fmt.Errorf("write grouped CSV record for %q in %q: %w", result.InputURL, group.CMS, err)
			}
		}
	}
	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return fmt.Errorf("flush grouped CSV report: %w", err)
	}
	return nil
}

func writeCMSTable(writer io.Writer, groups []model.CMSGroup) error {
	table := tabwriter.NewWriter(writer, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "CMS\tCOUNT\tSTATUS\tURL\tTITLE\tTECHNOLOGIES\tERROR"); err != nil {
		return fmt.Errorf("write grouped table header: %w", err)
	}
	for _, group := range groups {
		for _, result := range group.Sites {
			if _, err := fmt.Fprintf(table, "%s\t%d\t%s\t%s\t%s\t%s\t%s\n",
				oneLine(group.CMS),
				group.SiteCount,
				statusCode(result, "-"),
				oneLine(displayURL(result)),
				oneLine(result.Title),
				oneLine(technologyNames(result.Technologies)),
				oneLine(result.Error),
			); err != nil {
				return fmt.Errorf("write grouped table row for %q in %q: %w", result.InputURL, group.CMS, err)
			}
		}
	}
	if err := table.Flush(); err != nil {
		return fmt.Errorf("flush grouped table report: %w", err)
	}
	return nil
}

func writeCMSMarkdown(writer io.Writer, groups []model.CMSGroup) error {
	if _, err := fmt.Fprintln(writer, "| CMS | Count | Status | URL | Title | Technologies | Error |\n|---|---:|---:|---|---|---|---|"); err != nil {
		return fmt.Errorf("write grouped Markdown header: %w", err)
	}
	for _, group := range groups {
		for _, result := range group.Sites {
			if _, err := fmt.Fprintf(writer, "| %s | %d | %s | %s | %s | %s | %s |\n",
				markdownCell(group.CMS),
				group.SiteCount,
				markdownCell(statusCode(result, "—")),
				markdownCell(displayURL(result)),
				markdownCell(result.Title),
				markdownCell(technologyNames(result.Technologies)),
				markdownCell(result.Error),
			); err != nil {
				return fmt.Errorf("write grouped Markdown row for %q in %q: %w", result.InputURL, group.CMS, err)
			}
		}
	}
	return nil
}

func csvHeader() []string {
	return []string{"input_url", "url", "final_url", "status_code", "title", "technologies", "server", "content_type", "body_bytes", "truncated", "duration_ms", "scanned_at", "error"}
}

func csvRecord(result model.Result) []string {
	return []string{
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

func statusCode(result model.Result, empty string) string {
	if result.StatusCode == 0 {
		return empty
	}
	return strconv.Itoa(result.StatusCode)
}

func oneLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func markdownCell(value string) string {
	return strings.ReplaceAll(oneLine(value), "|", "\\|")
}
