package models

import (
	"time"
)

// ScanResult represents a comprehensive scan result with all details
type ScanResult struct {
	ID              int64                 `json:"id"`
	TaskID          int64                 `json:"task_id"`
	TaskName        string                `json:"task_name"`
	Status          string                `json:"status"` // completed, failed, running
	StartTime       time.Time             `json:"start_time"`
	EndTime         time.Time             `json:"end_time"`
	Duration        string                `json:"duration"`
	
	// Scan Configuration
	Targets         []string              `json:"targets"`
	Templates       []string              `json:"templates"`
	TemplateCount   int                   `json:"template_count"`
	TargetCount     int                   `json:"target_count"`
	
	// Scan Statistics
	TotalRequests   int                   `json:"total_requests"`
	CompletedRequests int                 `json:"completed_requests"`
	FoundVulns      int                   `json:"found_vulns"`
	SuccessRate     float64               `json:"success_rate"`
	
	// Nuclei Output
	Vulnerabilities []VulnerabilityDetail `json:"vulnerabilities"`
	LogSummary      LogSummary            `json:"log_summary"`
	RawOutput       string                `json:"raw_output,omitempty"` // Optional raw output
	
	// File Information
	OutputFile      string                `json:"output_file"`
	LogFile         string                `json:"log_file"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

// VulnerabilityDetail represents detailed vulnerability information
type VulnerabilityDetail struct {
	TemplateID      string            `json:"template_id"`
	Name            string            `json:"name"`
	Severity        string            `json:"severity"`
	Description     string            `json:"description"`
	Reference       []string          `json:"reference"`
	Tags            []string          `json:"tags"`
	Author          []string          `json:"author"`
	
	// Target Information
	Host            string            `json:"host"`
	URL             string            `json:"url"`
	MatchedAt       string            `json:"matched_at"`
	Timestamp       time.Time         `json:"timestamp"`
	
	// Request/Response Details
	Request         HTTPRequest       `json:"request"`
	Response        HTTPResponse      `json:"response"`
	
	// Additional Information
	ExtractedResults []string         `json:"extracted_results"`
	MatcherStatus   string            `json:"matcher_status"`
	MatcherName     string            `json:"matcher_name"`
	MatchedLine     string            `json:"matched_line"`
}

// HTTPRequest represents HTTP request details
type HTTPRequest struct {
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers"`
	Body        string            `json:"body"`
	Raw         string            `json:"raw"`
}

// HTTPResponse represents HTTP response details
type HTTPResponse struct {
	StatusCode  int               `json:"status_code"`
	Headers     map[string]string `json:"headers"`
	Body        string            `json:"body"`
	Raw         string            `json:"raw"`
	Size        int               `json:"size"`
}

// LogSummary represents a summary of scan logs
type LogSummary struct {
	KeyEvents        []KeyEvent               `json:"key_events"`
	Errors           []string                 `json:"errors"`
	Warnings         []string                 `json:"warnings"`
	TemplateExecutions []TemplateExecution    `json:"template_executions"`
	Statistics       []StatisticEntry         `json:"statistics"`
}

// KeyEvent represents important events during scanning
type KeyEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // start, template, vuln_found, error, complete
	Message   string    `json:"message"`
	Template  string    `json:"template,omitempty"`
	Target    string    `json:"target,omitempty"`
}

// TemplateExecution represents template execution information
type TemplateExecution struct {
	TemplateID   string    `json:"template_id"`
	Target       string    `json:"target"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Duration     string    `json:"duration"`
	Status       string    `json:"status"` // success, failed, timeout
	Requests     int       `json:"requests"`
	VulnsFound   int       `json:"vulns_found"`
}

// StatisticEntry represents a statistics entry
type StatisticEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Requests    int                    `json:"requests"`
	Total       int                    `json:"total"`
	Matched     int                    `json:"matched"`
	Errors      int                    `json:"errors"`
	Duration    string                 `json:"duration"`
	RPS         float64                `json:"rps"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}
