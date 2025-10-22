package scanner

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"wepoc/internal/models"
)

// LogSummary represents a summary of scan logs with key information only
type LogSummary struct {
	TaskID           int64                    `json:"task_id"`
	StartTime        time.Time                `json:"start_time"`
	EndTime          time.Time                `json:"end_time"`
	Duration         string                   `json:"duration"`
	TotalRequests    int                      `json:"total_requests"`
	CompletedRequests int                     `json:"completed_requests"`
	FoundVulns       int                      `json:"found_vulns"`
	SuccessRate      float64                  `json:"success_rate"`
	KeyEvents        []KeyEvent               `json:"key_events"`
	Vulnerabilities  []VulnerabilitySummary   `json:"vulnerabilities"`
	Errors           []string                 `json:"errors"`
	Warnings         []string                 `json:"warnings"`
}

// KeyEvent represents important events during scanning
type KeyEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // start, template, vuln_found, error, complete
	Message   string    `json:"message"`
	Template  string    `json:"template,omitempty"`
	Target    string    `json:"target,omitempty"`
}

// VulnerabilitySummary represents a summary of found vulnerabilities
type VulnerabilitySummary struct {
	TemplateID   string    `json:"template_id"`
	Name         string    `json:"name"`
	Severity     string    `json:"severity"`
	Target       string    `json:"target"`
	MatchedAt    string    `json:"matched_at"`
	Timestamp    time.Time `json:"timestamp"`
	Description  string    `json:"description,omitempty"`
}

// LogParser parses and filters Nuclei output to extract key information
type LogParser struct {
	summary     *LogSummary
	startTime   time.Time
	templates   map[string]string // template ID -> name mapping
}

// NewLogParser creates a new log parser
func NewLogParser(taskID int64) *LogParser {
	return &LogParser{
		summary: &LogSummary{
			TaskID:      taskID,
			StartTime:   time.Now(),
			KeyEvents:   make([]KeyEvent, 0),
			Vulnerabilities: make([]VulnerabilitySummary, 0),
			Errors:      make([]string, 0),
			Warnings:    make([]string, 0),
		},
		startTime: time.Now(),
		templates: make(map[string]string),
	}
}

// ParseLine parses a single line of Nuclei output and extracts key information
func (lp *LogParser) ParseLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	// Parse JSON stats
	if strings.HasPrefix(line, "{") && strings.Contains(line, "requests") {
		lp.parseStatsLine(line)
		return
	}

	// Parse template execution
	if strings.Contains(line, "Current template:") {
		lp.parseTemplateLine(line)
		return
	}

	// Parse scan completion
	if strings.Contains(line, "Scan completed") {
		lp.parseCompletionLine(line)
		return
	}

	// Parse vulnerability found (JSON output)
	if strings.HasPrefix(line, "{") && strings.Contains(line, "template-id") {
		lp.parseVulnerabilityLine(line)
		return
	}

	// Parse HTTP request dump (only log template name and target)
	if strings.Contains(line, "Dumped HTTP request for") {
		lp.parseHttpRequestDump(line)
		return
	}

	// Parse errors and warnings
	if strings.Contains(line, "[ERR]") {
		lp.parseErrorLine(line)
	} else if strings.Contains(line, "[WRN]") {
		lp.parseWarningLine(line)
	}
}

// parseStatsLine parses JSON statistics
func (lp *LogParser) parseStatsLine(line string) {
	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(line), &stats); err == nil {
		if requests, ok := stats["requests"]; ok {
			if completed, err := parseNumericValue(requests); err == nil {
				lp.summary.CompletedRequests = completed
			}
		}
		if total, ok := stats["total"]; ok {
			if totalInt, err := parseNumericValue(total); err == nil {
				lp.summary.TotalRequests = totalInt
			}
		}
		if matched, ok := stats["matched"]; ok {
			if matchedInt, err := parseNumericValue(matched); err == nil {
				lp.summary.FoundVulns = matchedInt
			}
		}
		if duration, ok := stats["duration"]; ok {
			if durationStr, ok := duration.(string); ok {
				lp.summary.Duration = durationStr
			}
		}
	}
}

// parseTemplateLine parses template execution information
func (lp *LogParser) parseTemplateLine(line string) {
	// Extract template name from line like "[INF] Current template: CVE-2017-12615"
	re := regexp.MustCompile(`Current template:\s*([^\s]+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		templateID := matches[1]
		lp.summary.KeyEvents = append(lp.summary.KeyEvents, KeyEvent{
			Timestamp: time.Now(),
			Type:      "template",
			Message:   fmt.Sprintf("Testing template: %s", templateID),
			Template:  templateID,
		})
	}
}

// parseCompletionLine parses scan completion information
func (lp *LogParser) parseCompletionLine(line string) {
	lp.summary.EndTime = time.Now()
	
	// Extract completion info from line like "[INF] Scan completed in 265.548708ms. 1 matches found."
	re := regexp.MustCompile(`Scan completed in ([\d.]+)ms\. (\d+) matches found\.`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 2 {
		duration := matches[1] + "ms"
		matches := matches[2]
		lp.summary.Duration = duration
		lp.summary.KeyEvents = append(lp.summary.KeyEvents, KeyEvent{
			Timestamp: time.Now(),
			Type:      "complete",
			Message:   fmt.Sprintf("Scan completed in %s, found %s vulnerabilities", duration, matches),
		})
	}
}

// parseVulnerabilityLine parses vulnerability JSON output
func (lp *LogParser) parseVulnerabilityLine(line string) {
	var vuln models.NucleiResult
	if err := json.Unmarshal([]byte(line), &vuln); err == nil {
		vulnSummary := VulnerabilitySummary{
			TemplateID:  vuln.TemplateID,
			Name:        vuln.Info.Name,
			Severity:    vuln.Info.Severity,
			Target:      vuln.Host,
			MatchedAt:   vuln.MatchedAt,
			Timestamp:   time.Now(),
			Description: vuln.Info.Description,
		}
		lp.summary.Vulnerabilities = append(lp.summary.Vulnerabilities, vulnSummary)
		
		lp.summary.KeyEvents = append(lp.summary.KeyEvents, KeyEvent{
			Timestamp: time.Now(),
			Type:      "vuln_found",
			Message:   fmt.Sprintf("Vulnerability found: %s (%s)", vuln.Info.Name, vuln.Info.Severity),
			Template:  vuln.TemplateID,
			Target:    vuln.Host,
		})
	}
}

// parseHttpRequestDump parses HTTP request dump (simplified)
func (lp *LogParser) parseHttpRequestDump(line string) {
	// Extract template name and URL from line like "[INF] [CVE-2017-12615] Dumped HTTP request for http://192.168.1.2:8080/..."
	re := regexp.MustCompile(`\[([^\]]+)\]\s+Dumped HTTP request for\s+(https?://[^\s]+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 2 {
		templateID := matches[1]
		target := matches[2]
		
		// Only log if we haven't seen this template-target combination recently
		key := fmt.Sprintf("%s-%s", templateID, target)
		if _, exists := lp.templates[key]; !exists {
			lp.templates[key] = target
			lp.summary.KeyEvents = append(lp.summary.KeyEvents, KeyEvent{
				Timestamp: time.Now(),
				Type:      "template",
				Message:   fmt.Sprintf("Testing %s against %s", templateID, target),
				Template:  templateID,
				Target:    target,
			})
		}
	}
}

// parseErrorLine parses error messages
func (lp *LogParser) parseErrorLine(line string) {
	// Extract error message after [ERR]
	parts := strings.SplitN(line, "[ERR]", 2)
	if len(parts) > 1 {
		errorMsg := strings.TrimSpace(parts[1])
		lp.summary.Errors = append(lp.summary.Errors, errorMsg)
		lp.summary.KeyEvents = append(lp.summary.KeyEvents, KeyEvent{
			Timestamp: time.Now(),
			Type:      "error",
			Message:   errorMsg,
		})
	}
}

// parseWarningLine parses warning messages
func (lp *LogParser) parseWarningLine(line string) {
	// Extract warning message after [WRN]
	parts := strings.SplitN(line, "[WRN]", 2)
	if len(parts) > 1 {
		warningMsg := strings.TrimSpace(parts[1])
		lp.summary.Warnings = append(lp.summary.Warnings, warningMsg)
	}
}

// GetSummary returns the parsed log summary
func (lp *LogParser) GetSummary() *LogSummary {
	// Calculate success rate
	if lp.summary.TotalRequests > 0 {
		lp.summary.SuccessRate = float64(lp.summary.CompletedRequests) / float64(lp.summary.TotalRequests) * 100
	}
	
	// Set end time if not set
	if lp.summary.EndTime.IsZero() {
		lp.summary.EndTime = time.Now()
	}
	
	return lp.summary
}

// parseNumericValue parses numeric values from various types
func parseNumericValue(value interface{}) (int, error) {
	switch v := value.(type) {
	case string:
		return strconv.Atoi(v)
	case float64:
		return int(v), nil
	case int:
		return v, nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", value)
	}
}
