// Package target loads website targets from command-line arguments and files.
package target

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Load combines explicit targets, positional arguments, and an optional list.
// Empty lines and lines beginning with # are ignored. Duplicates are removed
// while preserving the first occurrence.
func Load(explicit, positional []string, listPath string, stdin io.Reader) ([]string, error) {
	all := make([]string, 0, len(explicit)+len(positional))
	all = append(all, explicit...)

	if listPath != "" {
		var reader io.Reader
		var file *os.File

		if listPath == "-" {
			reader = stdin
		} else {
			opened, err := os.Open(listPath)
			if err != nil {
				return nil, fmt.Errorf("open target list %q: %w", listPath, err)
			}
			file = opened
			reader = opened
		}

		fromFile, err := readLines(reader)
		if file != nil {
			closeErr := file.Close()
			if err == nil && closeErr != nil {
				err = fmt.Errorf("close target list %q: %w", listPath, closeErr)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("read target list %q: %w", listPath, err)
		}
		all = append(all, fromFile...)
	}

	all = append(all, positional...)
	return uniqueTargets(all), nil
}

func readLines(reader io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(reader)
	buffer := make([]byte, 64*1024)
	scanner.Buffer(buffer, 1024*1024)

	var targets []string
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		targets = append(targets, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan lines: %w", err)
	}
	return targets, nil
}

func uniqueTargets(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
