package scanner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"wepoc/internal/database"
	"wepoc/internal/models"
)

// NucleiScanner manages a single Nuclei scanning process
type NucleiScanner struct {
	Task           *models.ScanTask
	DB             *database.Database
	Progress       *models.TaskProgress
	ProgressMu     sync.RWMutex
	OutputFile     string
	VulnResults    []*models.NucleiResult
	VulnResultsMu  sync.Mutex
	TaskManager    *TaskManager
	TaskID         int64
	LogParser      *LogParser
	ResultManager  *ResultManager
	RawOutput      strings.Builder
	NucleiPath     string // Add nuclei path configuration
}

// NewNucleiScanner creates a new Nuclei scanner instance
func NewNucleiScanner(task *models.ScanTask, db *database.Database, taskManager *TaskManager, taskID int64) *NucleiScanner {
	// Create result manager
	resultManager, err := NewResultManager()
	if err != nil {
		fmt.Printf("Warning: failed to create result manager: %v\n", err)
	}
	
	// Get nuclei path from configuration
	nucleiPath := "nuclei" // Default fallback
	if taskManager != nil && taskManager.config != nil {
		nucleiPath = taskManager.config.NucleiPath
	}
	
	return &NucleiScanner{
		Task: task,
		DB:   db,
		Progress: &models.TaskProgress{
			TaskID:            int(task.ID),
			TotalRequests:     task.TotalRequests,
			CompletedRequests: 0,
			FoundVulns:        0,
			Percentage:        0.0,
			Status:            "pending",
		},
		VulnResults:   make([]*models.NucleiResult, 0),
		TaskManager:   taskManager,
		TaskID:        taskID,
		LogParser:     NewLogParser(taskID),
		ResultManager: resultManager,
		RawOutput:     strings.Builder{},
		NucleiPath:    nucleiPath,
	}
}

// Start begins the scanning process
func (ns *NucleiScanner) Start(ctx context.Context, progressChan chan<- *models.TaskProgress) error {
	// Prepare output file
	outputFile, err := ns.prepareOutputFile()
	if err != nil {
		return fmt.Errorf("failed to prepare output file: %w", err)
	}
	ns.OutputFile = outputFile

	// Parse POCs and targets from JSON
	var pocs []string
	var targets []string

	if err := json.Unmarshal([]byte(ns.Task.POCs), &pocs); err != nil {
		return fmt.Errorf("failed to parse POCs: %w", err)
	}
	if err := json.Unmarshal([]byte(ns.Task.Targets), &targets); err != nil {
		return fmt.Errorf("failed to parse targets: %w", err)
	}

	// Create targets file
	targetsFile, err := ns.createTargetsFile(targets)
	if err != nil {
		return fmt.Errorf("failed to create targets file: %w", err)
	}
	defer os.Remove(targetsFile)

	// Build nuclei command
	cmd := ns.buildNucleiCommand(pocs, targetsFile, outputFile)
	cmd.Dir = filepath.Dir(outputFile)

	// Create pipes for stdout/stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start nuclei: %w", err)
	}

	// Update status
	ns.updateProgress("running", 0, 0)
	if progressChan != nil {
		progressChan <- ns.GetProgress()
	}

	// Monitor output in goroutines
	var wg sync.WaitGroup

	// Monitor stdout (Nuclei stats and info)
	wg.Add(1)
	go func() {
		defer wg.Done()
		ns.monitorStdout(stdout, progressChan)
	}()

	// Monitor stderr (errors)
	wg.Add(1)
	go func() {
		defer wg.Done()
		ns.monitorStderr(stderr)
	}()

	// Monitor output file for vulnerabilities
	wg.Add(1)
	go func() {
		defer wg.Done()
		ns.monitorOutputFile(ctx, outputFile, progressChan)
	}()

	// Wait for command to complete or context cancellation
	cmdDone := make(chan error, 1)
	go func() {
		cmdDone <- cmd.Wait()
	}()

	// Add timeout to prevent hanging
	timeout := time.After(30 * time.Minute) // 30 minutes timeout

	select {
	case <-ctx.Done():
		// Context cancelled, kill the process
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		wg.Wait()
		return fmt.Errorf("scan cancelled")
	case <-timeout:
		// Timeout reached, kill the process
		ns.logOutput("WARN", "Scan timeout reached, terminating process", false)
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		wg.Wait()
		return fmt.Errorf("scan timeout after 30 minutes")
	case err := <-cmdDone:
		// Command completed
		wg.Wait()
		if err != nil {
			return fmt.Errorf("nuclei command failed: %w", err)
		}
	}

	// Save vulnerabilities to output file
	if err := ns.saveVulnerabilities(); err != nil {
		return fmt.Errorf("failed to save vulnerabilities: %w", err)
	}

	// Save comprehensive scan result
	if err := ns.saveComprehensiveResult(); err != nil {
		// Log error but don't fail the scan
		fmt.Printf("Warning: failed to save comprehensive result: %v\n", err)
	}

	// Save log summary to results directory
	if err := ns.saveLogSummary(); err != nil {
		// Log error but don't fail the scan
		fmt.Printf("Warning: failed to save log summary: %v\n", err)
	}

	ns.updateProgress("completed", ns.Progress.TotalRequests, ns.Progress.FoundVulns)
	if progressChan != nil {
		progressChan <- ns.GetProgress()
	}

	return nil
}

// buildNucleiCommand builds the nuclei command with all parameters
func (ns *NucleiScanner) buildNucleiCommand(pocs []string, targetsFile, outputFile string) *exec.Cmd {
	args := []string{
		"-l", targetsFile, // List of targets
		"-jsonl", // JSON Lines output
		"-o", outputFile, // Output file
		"-stats", // Enable stats
		"-stats-interval", "2", // Stats interval
		"-debug", // Debug mode for better logging
		"-nc", // No color
		"-v", // Verbose
		"-c", "25", // Concurrency
		"-timeout", "10", // Request timeout
		"-retries", "1", // Number of retries
		"-rate-limit", "100", // Rate limit per second
		"-bulk-size", "25", // Bulk size for processing
		"-project", // Enable project mode for better organization
		"-project-path", filepath.Dir(outputFile), // Project path
		"-disable-update-check", // Disable update check
		"-no-meta", // Disable metadata in output
		"-include-rr", // Include request/response in output
		"-include-tags", // Include tags in output
	}

	// Use selected templates from the validated templates directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to individual templates if home dir not found
		for _, poc := range pocs {
			args = append(args, "-t", poc)
		}
	} else {
		// Convert POC IDs to template file paths in ~/.wepoc/nuclei-templates/
		templatesDir := filepath.Join(homeDir, ".wepoc", "nuclei-templates")
		for _, poc := range pocs {
			// poc is the template ID, find the corresponding file
			templateFile := filepath.Join(templatesDir, poc+".yaml")
			// Skip existence check for performance (already validated during import)
			args = append(args, "-t", templateFile)
		}
	}

	cmd := exec.Command(ns.NucleiPath, args...)
	
	// Hide the command window on Windows
	if runtime.GOOS == "windows" {
		hideWindowOnWindows(cmd)
	}
	
	return cmd
}

// createTargetsFile creates a temporary file with targets
func (ns *NucleiScanner) createTargetsFile(targets []string) (string, error) {
	tmpFile, err := os.CreateTemp("", "wepoc-targets-*.txt")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	for _, target := range targets {
		if _, err := tmpFile.WriteString(target + "\n"); err != nil {
			return "", err
		}
	}

	return tmpFile.Name(), nil
}

// prepareOutputFile prepares the output file path
func (ns *NucleiScanner) prepareOutputFile() (string, error) {
	// Expand home directory
	outputFile := strings.Replace(ns.Task.OutputFile, "~", os.Getenv("HOME"), 1)

	// Ensure directory exists
	dir := filepath.Dir(outputFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	// Change extension to .jsonl for nuclei output
	outputFile = strings.TrimSuffix(outputFile, ".json") + ".jsonl"

	return outputFile, nil
}

// monitorStdout monitors nuclei stdout for progress information
func (ns *NucleiScanner) monitorStdout(stdout io.Reader, progressChan chan<- *models.TaskProgress) {
	scanner := bufio.NewScanner(stdout)
	
	// Create a ticker for periodic progress updates
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	// Buffer for last line to avoid duplicate processing
	lastLine := ""
	
	for scanner.Scan() {
		line := scanner.Text()
		
		// Skip if line is empty or duplicate
		if line == "" || line == lastLine {
			continue
		}
		lastLine = line
		
		// Collect raw output
		ns.RawOutput.WriteString(line + "\n")
		
		// Use the new log parser to extract key information
		ns.LogParser.ParseLine(line)

		// Parse nuclei stats output (JSON format) for progress updates
		if strings.HasPrefix(line, "{") && strings.Contains(line, "requests") {
			ns.parseStatsLine(line)
			if progressChan != nil {
				progressChan <- ns.GetProgress()
			}
		}

		// Parse scan completion info
		if strings.Contains(line, "Scan completed") {
			ns.parseScanCompletion(line)
			if progressChan != nil {
				progressChan <- ns.GetProgress()
			}
		}
		
		// Send periodic updates even if no new stats
		select {
		case <-ticker.C:
			if progressChan != nil {
				progressChan <- ns.GetProgress()
			}
		default:
			// Don't block if ticker channel is not ready
		}
	}
	
	// Send final update when scanner finishes
	if progressChan != nil {
		progressChan <- ns.GetProgress()
	}
}

// monitorStderr monitors nuclei stderr for errors
func (ns *NucleiScanner) monitorStderr(stderr io.Reader) {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		ns.logOutput("ERROR", line, false)
	}
}

// monitorOutputFile monitors the output JSONL file for new vulnerabilities
func (ns *NucleiScanner) monitorOutputFile(ctx context.Context, outputFile string, progressChan chan<- *models.TaskProgress) {
	// Wait for file to be created
	for {
		if _, err := os.Stat(outputFile); err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return
		default:
		}

		time.Sleep(100 * time.Millisecond)
	}

	file, err := os.Open(outputFile)
	if err != nil {
		return
	}
	defer file.Close()

	// Keep track of file position to avoid re-reading
	var lastPos int64 = 0
	
	// Create a ticker for periodic checks
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Continue with file checking
		}

		// Seek to last position
		file.Seek(lastPos, 0)

		// Read new content
		scanner := bufio.NewScanner(file)
		newVulnsFound := false
		
		for scanner.Scan() {
			line := scanner.Text()
			lastPos += int64(len(line)) + 1 // +1 for newline

			var result models.NucleiResult
			if err := json.Unmarshal([]byte(line), &result); err == nil {
				// Only process if there's a match (vulnerability found)
				if result.MatchedAt != "" {
					ns.VulnResultsMu.Lock()
					ns.VulnResults = append(ns.VulnResults, &result)
					ns.VulnResultsMu.Unlock()

					// Update found vulns count
					ns.ProgressMu.Lock()
					ns.Progress.FoundVulns++
					ns.ProgressMu.Unlock()
					newVulnsFound = true

					// Log vulnerability with more details
					vulnMsg := fmt.Sprintf("🔴 VULNERABILITY FOUND: %s (%s) at %s", 
						result.Info.Name, result.Info.Severity, result.MatchedAt)
					ns.logOutput("VULN", vulnMsg, true)

					// Log additional details
					if result.Info.Description != "" {
						ns.logOutput("VULN", fmt.Sprintf("Description: %s", result.Info.Description), true)
					}
					if len(result.Info.Tags) > 0 {
						ns.logOutput("VULN", fmt.Sprintf("Tags: %s", strings.Join(result.Info.Tags, ", ")), true)
					}
				} else {
					// Log non-vulnerability requests for debugging
					ns.logOutput("INFO", fmt.Sprintf("Testing %s - No vulnerability found", result.TemplateID), false)
				}
			} else {
				// Log JSON parsing errors
				ns.logOutput("ERROR", fmt.Sprintf("Failed to parse JSON line: %s", line), false)
			}
		}

		// Send progress update if new vulnerabilities were found or periodically
		if (newVulnsFound || time.Now().Unix()%5 == 0) && progressChan != nil {
			// Get fresh progress data
			currentProgress := ns.GetProgress()
			progressChan <- currentProgress
		}
	}
}

// parseStatsLine parses nuclei stats output to update progress
func (ns *NucleiScanner) parseStatsLine(line string) {
	// Try to parse JSON stats output from nuclei
	// Example: {"duration":"0:00:02","errors":"0","hosts":"1","matched":"0","percent":"0","requests":"7","rps":"3","startedAt":"2025-10-17T22:57:05.635618+08:00","templates":"8642","total":"15696"}
	if strings.HasPrefix(line, "{") && strings.Contains(line, "requests") {
		var stats map[string]interface{}
		if err := json.Unmarshal([]byte(line), &stats); err == nil {
			ns.ProgressMu.Lock()
			
			// Update progress from stats - handle both string and number types
			if requests, ok := stats["requests"]; ok {
				var completed int
				switch v := requests.(type) {
				case string:
					if parsed, err := strconv.Atoi(v); err == nil {
						completed = parsed
					}
				case float64:
					completed = int(v)
				case int:
					completed = v
				}
				ns.Progress.CompletedRequests = completed
			}
			
			if total, ok := stats["total"]; ok {
				var totalInt int
				switch v := total.(type) {
				case string:
					if parsed, err := strconv.Atoi(v); err == nil {
						totalInt = parsed
					}
				case float64:
					totalInt = int(v)
				case int:
					totalInt = v
				}
				ns.Progress.TotalRequests = totalInt
			}
			
			// Don't override FoundVulns from stats, as it may be 0 even when vulnerabilities are found
			// The FoundVulns count is updated by monitorOutputFile when actual vulnerabilities are detected
			// Only use stats for requests and total counts
			if matched, ok := stats["matched"]; ok {
				var matchedInt int
				switch v := matched.(type) {
				case string:
					if parsed, err := strconv.Atoi(v); err == nil {
						matchedInt = parsed
					}
				case float64:
					matchedInt = int(v)
				case int:
					matchedInt = v
				}
				// Only update if the stats show more vulnerabilities than we currently have
				// This handles cases where stats are more accurate than our counting
				if matchedInt > ns.Progress.FoundVulns {
					ns.Progress.FoundVulns = matchedInt
				}
			}
			
			// Calculate percentage
			if ns.Progress.TotalRequests > 0 {
				ns.Progress.Percentage = float64(ns.Progress.CompletedRequests) / float64(ns.Progress.TotalRequests) * 100
			}
			
			// Log the progress update
			ns.logOutput("INFO", fmt.Sprintf("Progress update: %d/%d requests completed, %d vulnerabilities found (%.1f%%)", 
				ns.Progress.CompletedRequests, ns.Progress.TotalRequests, ns.Progress.FoundVulns, ns.Progress.Percentage), false)
			
			ns.ProgressMu.Unlock()
		}
	}
	
	// Send progress update immediately after parsing stats
	// This ensures frontend gets updates as soon as they're available
	ns.ProgressMu.RLock()
	defer ns.ProgressMu.RUnlock()
}

// parseTemplateInfo parses current template information
func (ns *NucleiScanner) parseTemplateInfo(line string) {
	// Extract template name from line like "Current template: CVE-2021-1234"
	if strings.Contains(line, "Current template:") {
		parts := strings.Split(line, "Current template:")
		if len(parts) > 1 {
			templateName := strings.TrimSpace(parts[1])
			// Extract just the template ID (before any brackets)
			if bracketIndex := strings.Index(templateName, "["); bracketIndex != -1 {
				templateName = templateName[:bracketIndex]
			}
			
			ns.ProgressMu.Lock()
			ns.Progress.CurrentTemplate = templateName
			ns.ProgressMu.Unlock()
		}
	}
}

// parseScanCompletion parses scan completion information
func (ns *NucleiScanner) parseScanCompletion(line string) {
	// Extract completion stats from line like "Scan completed in 212.727375ms. 1 matches found."
	if strings.Contains(line, "Scan completed") {
		ns.ProgressMu.Lock()
		ns.Progress.Status = "completed"
		ns.Progress.CompletedRequests = ns.Progress.TotalRequests
		ns.Progress.Percentage = 100.0
		ns.ProgressMu.Unlock()
	}
}

// parseHttpRequestDump parses HTTP request dump information
func (ns *NucleiScanner) parseHttpRequestDump(line string) {
	// Extract template name and URL from line like "[INF] [CVE-2017-12615] Dumped HTTP request for http://172.20.10.10:8080/34CR9mqTbmFHbnpqEjxu1SAW2jo.txt/"
	if strings.Contains(line, "Dumped HTTP request for") {
		// Extract template name (e.g., CVE-2017-12615)
		parts := strings.Split(line, "]")
		if len(parts) >= 2 {
			templateName := strings.TrimSpace(parts[1])
			templateName = strings.Trim(templateName, "[")
			
			// Extract URL
			urlStart := strings.Index(line, "http://")
			if urlStart != -1 {
				urlEnd := strings.Index(line[urlStart:], " ")
				if urlEnd == -1 {
					urlEnd = len(line) - urlStart
				}
				url := line[urlStart : urlStart+urlEnd]
				
				ns.logOutput("INFO", fmt.Sprintf("Testing %s against %s", templateName, url), false)
			}
		}
	}
}

// updateProgress updates the progress status
func (ns *NucleiScanner) updateProgress(status string, completed int, foundVulns int) {
	ns.ProgressMu.Lock()
	defer ns.ProgressMu.Unlock()

	ns.Progress.Status = status
	if completed > 0 {
		ns.Progress.CompletedRequests = completed
	}
	if foundVulns > 0 {
		ns.Progress.FoundVulns = foundVulns
	}
	if ns.Progress.TotalRequests > 0 {
		ns.Progress.Percentage = float64(ns.Progress.CompletedRequests) / float64(ns.Progress.TotalRequests) * 100
	}
}

// GetProgress returns current progress (thread-safe)
func (ns *NucleiScanner) GetProgress() *models.TaskProgress {
	ns.ProgressMu.RLock()
	defer ns.ProgressMu.RUnlock()

	// Return a copy
	return &models.TaskProgress{
		TaskID:            ns.Progress.TaskID,
		TotalRequests:     ns.Progress.TotalRequests,
		CompletedRequests: ns.Progress.CompletedRequests,
		FoundVulns:        ns.Progress.FoundVulns,
		Percentage:        ns.Progress.Percentage,
		CurrentTemplate:   ns.Progress.CurrentTemplate,
		Status:            ns.Progress.Status,
	}
}

// saveVulnerabilities saves found vulnerabilities to JSON file
func (ns *NucleiScanner) saveVulnerabilities() error {
	ns.VulnResultsMu.Lock()
	defer ns.VulnResultsMu.Unlock()

	// If no vulnerabilities found, don't create file
	if len(ns.VulnResults) == 0 {
		return nil
	}

	// Convert .jsonl to .json for final output
	outputFile := strings.Replace(ns.OutputFile, ".jsonl", ".json", 1)

	// Marshal to JSON
	data, err := json.MarshalIndent(ns.VulnResults, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write results: %w", err)
	}

	// Remove the .jsonl file
	os.Remove(ns.OutputFile)

	// Update output file in task
	ns.Task.OutputFile = outputFile

	return nil
}

// saveLogSummary saves the parsed log summary to the results directory
func (ns *NucleiScanner) saveLogSummary() error {
	// Get the log summary from the parser
	summary := ns.LogParser.GetSummary()
	
	// Create results directory if it doesn't exist
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	
	resultsDir := filepath.Join(homeDir, ".wepoc", "results")
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		return fmt.Errorf("failed to create results directory: %w", err)
	}
	
	// Create log summary filename
	timestamp := time.Now().Format("20060102_150405")
	logSummaryFile := filepath.Join(resultsDir, fmt.Sprintf("task_%d_log_summary_%s.json", ns.TaskID, timestamp))
	
	// Marshal summary to JSON
	summaryJSON, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal log summary: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(logSummaryFile, summaryJSON, 0644); err != nil {
		return fmt.Errorf("failed to write log summary file: %w", err)
	}
	
	// Also save to TaskManager for frontend access
	if ns.TaskManager != nil {
		// Convert summary to ScanLog format for compatibility
		summaryLog := &models.ScanLog{
			TaskID:      int(ns.TaskID),
			Timestamp:   time.Now(),
			Level:       "SUMMARY",
			TemplateID:  "",
			Target:      ns.Task.Targets,
			Message:     fmt.Sprintf("Scan completed: %d/%d requests, %d vulnerabilities found", 
				summary.CompletedRequests, summary.TotalRequests, summary.FoundVulns),
			IsVulnFound: summary.FoundVulns > 0,
		}
		ns.TaskManager.AddTaskLog(ns.TaskID, summaryLog)
	}
	
	return nil
}

// saveComprehensiveResult saves a comprehensive scan result with all details
func (ns *NucleiScanner) saveComprehensiveResult() error {
	if ns.ResultManager == nil {
		return fmt.Errorf("result manager not initialized")
	}
	
	// Create comprehensive scan result
	result := ns.ResultManager.CreateScanResultFromTask(
		ns.Task,
		ns.LogParser,
		ns.VulnResults,
		ns.RawOutput.String(),
	)
	
	// Save to file
	if err := ns.ResultManager.SaveScanResult(result); err != nil {
		return fmt.Errorf("failed to save comprehensive result: %w", err)
	}
	
	return nil
}

// logOutput logs scanner output
func (ns *NucleiScanner) logOutput(level, message string, isVuln bool) {
	log := &models.ScanLog{
		TaskID:      int(ns.Task.ID),
		Timestamp:   time.Now(),
		Level:       level,
		TemplateID:  ns.Progress.CurrentTemplate,
		Target:      ns.Task.Targets, // Use targets string as default
		Message:     message,
		IsVulnFound: isVuln,
	}

	// Add log to task-specific log storage
	if ns.TaskManager != nil {
		ns.TaskManager.AddTaskLog(ns.TaskID, log)
	}
}
