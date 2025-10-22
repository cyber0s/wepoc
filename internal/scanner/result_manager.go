package scanner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"wepoc/internal/models"
)

// ResultManager manages scan results storage and retrieval
type ResultManager struct {
	resultsDir string
}

// NewResultManager creates a new result manager
func NewResultManager() (*ResultManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	
	resultsDir := filepath.Join(homeDir, ".wepoc", "results")
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create results directory: %w", err)
	}
	
	return &ResultManager{
		resultsDir: resultsDir,
	}, nil
}

// SaveScanResult saves a comprehensive scan result to JSON file
func (rm *ResultManager) SaveScanResult(result *models.ScanResult) error {
	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("scan_result_%d_%s.json", result.TaskID, timestamp)
	filepath := filepath.Join(rm.resultsDir, filename)
	
	// Update file paths in result
	result.OutputFile = filepath
	result.LogFile = filepath + ".log"
	result.UpdatedAt = time.Now()
	
	// Marshal to JSON with indentation
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal scan result: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write scan result file: %w", err)
	}
	
	// Also save a log file with raw output if available
	if result.RawOutput != "" {
		logFilepath := filepath + ".log"
		if err := os.WriteFile(logFilepath, []byte(result.RawOutput), 0644); err != nil {
			// Log error but don't fail the main operation
			fmt.Printf("Warning: failed to save log file: %v\n", err)
		}
	}
	
	return nil
}

// LoadScanResult loads a scan result from JSON file
func (rm *ResultManager) LoadScanResult(filepath string) (*models.ScanResult, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read scan result file: %w", err)
	}
	
	var result models.ScanResult
	if err := json.Unmarshal(content, &result); err != nil {
		return nil, fmt.Errorf("failed to parse scan result file: %w", err)
	}
	
	return &result, nil
}

// ListScanResults lists all scan result files
func (rm *ResultManager) ListScanResults() ([]string, error) {
	pattern := filepath.Join(rm.resultsDir, "scan_result_*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search for scan result files: %w", err)
	}
	
	return matches, nil
}

// GetScanResultByTaskID finds a scan result by task ID
func (rm *ResultManager) GetScanResultByTaskID(taskID int64) (*models.ScanResult, error) {
	pattern := filepath.Join(rm.resultsDir, fmt.Sprintf("scan_result_%d_*.json", taskID))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search for scan result files: %w", err)
	}
	
	if len(matches) == 0 {
		return nil, fmt.Errorf("no scan result found for task ID %d", taskID)
	}
	
	// Return the most recent result
	latestFile := matches[len(matches)-1]
	return rm.LoadScanResult(latestFile)
}

// GetAllScanResults loads all scan results
func (rm *ResultManager) GetAllScanResults() ([]*models.ScanResult, error) {
	files, err := rm.ListScanResults()
	if err != nil {
		return nil, err
	}
	
	var results []*models.ScanResult
	for _, file := range files {
		result, err := rm.LoadScanResult(file)
		if err != nil {
			// Log error but continue with other files
			fmt.Printf("Warning: failed to load scan result from %s: %v\n", file, err)
			continue
		}
		results = append(results, result)
	}
	
	return results, nil
}

// DeleteScanResult deletes a scan result file
func (rm *ResultManager) DeleteScanResult(filepath string) error {
	// Delete main JSON file
	if err := os.Remove(filepath); err != nil {
		return fmt.Errorf("failed to delete scan result file: %w", err)
	}
	
	// Delete associated log file if it exists
	logFilepath := filepath + ".log"
	if _, err := os.Stat(logFilepath); err == nil {
		if err := os.Remove(logFilepath); err != nil {
			fmt.Printf("Warning: failed to delete log file %s: %v\n", logFilepath, err)
		}
	}
	
	return nil
}

// CreateScanResultFromTask creates a scan result from a completed task
func (rm *ResultManager) CreateScanResultFromTask(task *models.ScanTask, logParser *LogParser, vulnResults []*models.NucleiResult, rawOutput string) *models.ScanResult {
	// Parse task data
	var targets []string
	var templates []string
	
	if task.Targets != "" {
		json.Unmarshal([]byte(task.Targets), &targets)
	}
	if task.POCs != "" {
		json.Unmarshal([]byte(task.POCs), &templates)
	}
	
	// Create vulnerability details
	var vulnDetails []models.VulnerabilityDetail
	for _, vuln := range vulnResults {
		// Convert reference to string slice
		var references []string
		if vuln.Info.Reference != nil {
			switch ref := vuln.Info.Reference.(type) {
			case []string:
				references = ref
			case string:
				references = []string{ref}
			case []interface{}:
				for _, r := range ref {
					if str, ok := r.(string); ok {
						references = append(references, str)
					}
				}
			}
		}
		
		vulnDetail := models.VulnerabilityDetail{
			TemplateID:      vuln.TemplateID,
			Name:            vuln.Info.Name,
			Severity:        vuln.Info.Severity,
			Description:     vuln.Info.Description,
			Reference:       references,
			Tags:            vuln.Info.Tags,
			Author:          vuln.Info.Author,
			Host:            vuln.Host,
			MatchedAt:       vuln.MatchedAt,
			Timestamp:       time.Now(),
			Request: models.HTTPRequest{
				Raw: vuln.Request, // Store raw request string
			},
			Response: models.HTTPResponse{
				Raw: vuln.Response, // Store raw response string
			},
			ExtractedResults: vuln.ExtractedResults,
		}
		vulnDetails = append(vulnDetails, vulnDetail)
	}
	
	// Create log summary
	var logSummary models.LogSummary
	if logParser != nil {
		summary := logParser.GetSummary()
		logSummary = models.LogSummary{
			KeyEvents:         convertKeyEvents(summary.KeyEvents),
			Errors:            summary.Errors,
			Warnings:          summary.Warnings,
			TemplateExecutions: []models.TemplateExecution{}, // TODO: Extract from log parser
			Statistics:        []models.StatisticEntry{},     // TODO: Extract from log parser
		}
	}
	
	// Calculate duration
	var duration string
	var endTime time.Time
	if task.EndTime != nil {
		endTime = *task.EndTime
		if !task.StartTime.IsZero() && !endTime.IsZero() {
			duration = endTime.Sub(task.StartTime).String()
		}
	}
	
	// Create comprehensive scan result
	result := &models.ScanResult{
		ID:                task.ID,
		TaskID:            task.ID,
		TaskName:          task.Name,
		Status:            task.Status,
		StartTime:         task.StartTime,
		EndTime:           endTime,
		Duration:          duration,
		Targets:           targets,
		Templates:         templates,
		TemplateCount:     len(templates),
		TargetCount:       len(targets),
		TotalRequests:     task.TotalRequests,
		CompletedRequests: task.CompletedRequests,
		FoundVulns:        task.FoundVulns,
		SuccessRate:       calculateSuccessRate(task.CompletedRequests, task.TotalRequests),
		Vulnerabilities:   vulnDetails,
		LogSummary:        logSummary,
		RawOutput:         rawOutput,
		CreatedAt:         task.CreatedAt,
		UpdatedAt:         time.Now(),
	}
	
	return result
}

// Helper functions
func convertKeyEvents(events []KeyEvent) []models.KeyEvent {
	var result []models.KeyEvent
	for _, event := range events {
		result = append(result, models.KeyEvent{
			Timestamp: event.Timestamp,
			Type:      event.Type,
			Message:   event.Message,
			Template:  event.Template,
			Target:    event.Target,
		})
	}
	return result
}

func calculateSuccessRate(completed, total int) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(completed) / float64(total) * 100
}
