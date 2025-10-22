package models

import (
	"time"
)

// Template represents a Nuclei template/POC
type Template struct {
	ID         int64     `json:"id"`
	TemplateID string    `json:"template_id"` // Unique template ID from YAML
	Name       string    `json:"name"`
	Severity   string    `json:"severity"`
	Tags       string    `json:"tags"`      // JSON array string
	Author     string    `json:"author"`
	FilePath   string    `json:"file_path"` // Path to template file
	CreatedAt  time.Time `json:"created_at"`
}

// ScanTask represents a scanning task
type ScanTask struct {
	ID                 int64     `json:"id"`
	Name               string    `json:"name"`              // Task-1, Task-2...
	Status             string    `json:"status"`            // running/paused/stopped/completed
	POCs               string    `json:"pocs"`              // JSON array of template IDs
	Targets            string    `json:"targets"`           // JSON array of target URLs
	TotalRequests      int       `json:"total_requests"`    // Total number of requests
	CompletedRequests  int       `json:"completed_requests"` // Completed requests
	FoundVulns         int       `json:"found_vulns"`       // Number of vulnerabilities found
	OutputFile         string    `json:"output_file"`       // Path to output JSON file
	StartTime          time.Time `json:"start_time"`
	EndTime            *time.Time `json:"end_time,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

// TaskProgress represents real-time progress of a task
type TaskProgress struct {
	TaskID            int     `json:"task_id"`
	TotalRequests     int     `json:"total_requests"`
	CompletedRequests int     `json:"completed_requests"`
	FoundVulns        int     `json:"found_vulns"`
	Percentage        float64 `json:"percentage"`
	CurrentTemplate   string  `json:"current_template"`
	Status            string  `json:"status"`
}

// NucleiResult represents a single vulnerability finding
type NucleiResult struct {
	TemplateID   string                 `json:"template-id"`
	TemplatePath string                 `json:"template-path"`
	Info         NucleiInfo             `json:"info"`
	Type         string                 `json:"type"`
	Host         string                 `json:"host"`
	MatchedAt    string                 `json:"matched-at"`
	ExtractedResults []string           `json:"extracted-results,omitempty"`
	Request      string                 `json:"request,omitempty"`
	Response     string                 `json:"response,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	CurlCommand  string                 `json:"curl-command,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// NucleiInfo contains template metadata
type NucleiInfo struct {
	Name        string                 `json:"name"`
	Author      []string               `json:"author"`
	Tags        []string               `json:"tags"`
	Description string                 `json:"description,omitempty"`
	Reference   interface{}            `json:"reference,omitempty"`
	Severity    string                 `json:"severity"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Config represents application configuration
type Config struct {
	POCDirectory   string `json:"poc_directory"`   // Path to POC templates
	ResultsDir     string `json:"results_dir"`     // Path to results directory
	DatabasePath   string `json:"database_path"`   // Path to SQLite database
	NucleiPath     string `json:"nuclei_path"`     // Path to nuclei binary
	MaxConcurrency int    `json:"max_concurrency"` // Max concurrent tasks
	Timeout        int    `json:"timeout"`         // Request timeout in seconds
}

// ScanLog represents a log entry during scanning
type ScanLog struct {
	TaskID      int       `json:"task_id"`
	Timestamp   time.Time `json:"timestamp"`
	Level       string    `json:"level"` // INFO, WARNING, ERROR
	TemplateID  string    `json:"template_id"`
	Target      string    `json:"target"`
	Message     string    `json:"message"`
	IsVulnFound bool      `json:"is_vuln_found"`
}
