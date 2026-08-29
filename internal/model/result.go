// Package model contains the data exchanged by the scanner and report writers.
package model

import "time"

// Technology describes a technology detected on a website.
type Technology struct {
	Name        string   `json:"name"`
	Version     string   `json:"version,omitempty"`
	Categories  []string `json:"categories,omitempty"`
	Website     string   `json:"website,omitempty"`
	Description string   `json:"description,omitempty"`
	CPE         string   `json:"cpe,omitempty"`
}

// Result is the complete report for one input target.
type Result struct {
	InputURL     string       `json:"input_url"`
	URL          string       `json:"url,omitempty"`
	FinalURL     string       `json:"final_url,omitempty"`
	StatusCode   int          `json:"status_code,omitempty"`
	Title        string       `json:"title,omitempty"`
	Technologies []Technology `json:"technologies"`
	Server       string       `json:"server,omitempty"`
	ContentType  string       `json:"content_type,omitempty"`
	BodyBytes    int64        `json:"body_bytes,omitempty"`
	Truncated    bool         `json:"truncated,omitempty"`
	DurationMS   int64        `json:"duration_ms"`
	ScannedAt    time.Time    `json:"scanned_at"`
	Error        string       `json:"error,omitempty"`
}

// CMSGroup contains all scan results associated with one detected CMS.
type CMSGroup struct {
	CMS       string   `json:"cms"`
	SiteCount int      `json:"site_count"`
	Sites     []Result `json:"sites"`
}
