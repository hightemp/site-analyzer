# site-analyzer

[![Release](https://github.com/hightemp/site-analyzer/actions/workflows/release.yml/badge.svg)](https://github.com/hightemp/site-analyzer/actions/workflows/release.yml)
[![Latest release](https://img.shields.io/github/v/release/hightemp/site-analyzer)](https://github.com/hightemp/site-analyzer/releases/latest)
[![Downloads](https://img.shields.io/github/downloads/hightemp/site-analyzer/total)](https://github.com/hightemp/site-analyzer/releases)
[![Go version](https://img.shields.io/github/go-mod/go-version/hightemp/site-analyzer)](go.mod)

Website technology scanner written in Go. It fetches pages over HTTP(S), detects
technologies with
[`github.com/projectdiscovery/wappalyzergo`](https://github.com/projectdiscovery/wappalyzergo)
and produces machine-readable or human-readable reports.

## Build

Go 1.25 or newer is required.

```bash
make build
./bin/site-analyzer --help
```

## Usage

Scan one website:

```bash
./bin/site-analyzer -u https://example.com
```

Multiple URLs can be supplied with repeated `-u` options, positional arguments,
or a TXT file. Empty lines and lines beginning with `#` are ignored.

```bash
./bin/site-analyzer -l targets.txt -c 20 -f jsonl -o report.jsonl
./bin/site-analyzer example.com https://projectdiscovery.io -f table
cat targets.txt | ./bin/site-analyzer -l - -f csv -o report.csv
```

When the URL scheme is omitted, `https://` is used. A failed target is recorded
in the report without stopping the remaining scans. The process exits with code
`1` when one or more targets fail.

### Report formats

- `jsonl` / `ndjson` — one complete JSON object per website, suitable for streaming;
- `json` — an indented JSON array;
- `csv` — one row per website;
- `table` — a compact terminal table;
- `markdown` / `md` — a Markdown table.

JSON and JSONL reports include the input and final URLs, HTTP status, page title,
server, content type, duration, processed body size, error, and detailed
technology information: name, version, categories, website, description, and CPE.

### Grouping by CMS

Use `--group-by cms` to aggregate sites by technologies in Wappalyzer's `CMS`
category:

```bash
./bin/site-analyzer -l targets.txt --group-by cms -f json -o cms-report.json
```

Grouped JSON reports contain the CMS name, site count, and the full scan results
for every site in that group. A site detected with multiple CMS technologies is
included in every matching group. Sites without a detected CMS, including failed
scans, are placed in the `Unknown` group.

```json
[
  {
    "cms": "WordPress",
    "site_count": 2,
    "sites": [
      {
        "input_url": "https://example.com",
        "technologies": []
      }
    ]
  }
]
```

CMS grouping is supported by all report formats. JSONL emits one group per line;
CSV, table, and Markdown reports emit one CMS-site row.

### Useful options

```text
-c, --concurrency N  number of concurrent requests (default: 10)
--timeout 15s        timeout per target
--max-body N         maximum processed response body (default: 5 MiB)
--group-by cms       group report results by detected CMS
-H 'Name: value'     additional HTTP header; may be repeated
--insecure           allow invalid or self-signed TLS certificates
--no-redirect        do not follow HTTP redirects
--quiet              do not print the summary to stderr
```

## Development

```bash
make           # list available commands
make check     # tests, vet, and lint
make snapshot  # build local release archives in dist/
make test
make vet
make lint
```

## Publishing a release

Publish from a clean, up-to-date `main` branch using a semantic version tag:

```bash
make release VERSION=v1.2.3
```

The command runs all checks, creates an annotated tag, and pushes it to `origin`.
The tag triggers the [release workflow](.github/workflows/release.yml), which
publishes Linux, macOS, and Windows archives for AMD64 and ARM64 together with a
`checksums.txt` file. Use `make publish VERSION=v1.2.3` as an equivalent alias.

The code is split into the `target`, `scanner`, `runner`, `report`, and `app`
packages under `internal/`. The entry point is located in `cmd/site-analyzer`.
