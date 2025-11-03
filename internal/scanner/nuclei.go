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
	Task          *models.ScanTask
	DB            *database.Database
	Progress      *models.TaskProgress
	ProgressMu    sync.RWMutex
	OutputFile    string
	VulnResults   []*models.NucleiResult
	VulnResultsMu sync.Mutex
	TaskManager   *TaskManager
	TaskID        int64
	LogParser     *LogParser
	ResultManager *ResultManager
	RawOutput     strings.Builder
	NucleiPath    string          // Add nuclei path configuration
	TempDir       string          // Temporary directory for templates
	Logger        *EnhancedLogger // Enhanced logger for detailed logging
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

	// Initialize enhanced logger
	logger, err := NewEnhancedLogger(taskID, fmt.Sprintf("nuclei_scanner_%d", taskID))
	if err != nil {
		fmt.Printf("Warning: failed to create enhanced logger: %v\n", err)
	}

	scanner := &NucleiScanner{
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
		Logger:        logger,
	}

	if logger != nil {
		logger.Info("NucleiScanner initialized", map[string]interface{}{
			"task_id":        taskID,
			"task_name":      task.Name,
			"nuclei_path":    nucleiPath,
			"total_requests": task.TotalRequests,
		})
	}

	return scanner
}

// Start begins the scanning process
func (ns *NucleiScanner) Start(ctx context.Context, progressChan chan<- *models.TaskProgress) error {
	scanStartTime := time.Now()

	if ns.Logger != nil {
		ns.Logger.Info("Starting nuclei scan", map[string]interface{}{
			"task_id":   ns.TaskID,
			"task_name": ns.Task.Name,
		})
	}

	// Prepare output file
	outputFile, err := ns.prepareOutputFile()
	if err != nil {
		if ns.Logger != nil {
			ns.Logger.Error("Failed to prepare output file", err, map[string]interface{}{
				"task_id": ns.TaskID,
			})
		}
		return fmt.Errorf("failed to prepare output file: %w", err)
	}
	ns.OutputFile = outputFile

	// Parse POCs and targets from JSON
	var pocs []string
	var targets []string

	if err := json.Unmarshal([]byte(ns.Task.POCs), &pocs); err != nil {
		if ns.Logger != nil {
			ns.Logger.Error("Failed to parse POCs from JSON", err, map[string]interface{}{
				"task_id":  ns.TaskID,
				"pocs_raw": ns.Task.POCs,
			})
		}
		return fmt.Errorf("failed to parse POCs: %w", err)
	}
	if err := json.Unmarshal([]byte(ns.Task.Targets), &targets); err != nil {
		if ns.Logger != nil {
			ns.Logger.Error("Failed to parse targets from JSON", err, map[string]interface{}{
				"task_id":     ns.TaskID,
				"targets_raw": ns.Task.Targets,
			})
		}
		return fmt.Errorf("failed to parse targets: %w", err)
	}

	if ns.Logger != nil {
		ns.Logger.Info("Parsed scan parameters", map[string]interface{}{
			"task_id":      ns.TaskID,
			"poc_count":    len(pocs),
			"target_count": len(targets),
			"output_file":  outputFile,
		})
	}

	// Create targets file
	targetsFile, err := ns.createTargetsFile(targets)
	if err != nil {
		if ns.Logger != nil {
			ns.Logger.Error("Failed to create targets file", err, map[string]interface{}{
				"task_id":      ns.TaskID,
				"target_count": len(targets),
			})
		}
		return fmt.Errorf("failed to create targets file: %w", err)
	}
	defer os.Remove(targetsFile)

	// Build nuclei command
	cmd := ns.buildNucleiCommand(pocs, targetsFile, outputFile)
	cmd.Dir = filepath.Dir(outputFile)

	// Create pipes for stdout/stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		if ns.Logger != nil {
			ns.Logger.Error("Failed to create stdout pipe", err, map[string]interface{}{
				"task_id": ns.TaskID,
			})
		}
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		if ns.Logger != nil {
			ns.Logger.Error("Failed to create stderr pipe", err, map[string]interface{}{
				"task_id": ns.TaskID,
			})
		}
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	cmdStartTime := time.Now()
	if err := cmd.Start(); err != nil {
		if ns.Logger != nil {
			ns.Logger.Error("Failed to start nuclei command", err, map[string]interface{}{
				"task_id":     ns.TaskID,
				"nuclei_path": ns.NucleiPath,
				"working_dir": cmd.Dir,
			})
		}
		return fmt.Errorf("failed to start nuclei: %w", err)
	}

	if ns.Logger != nil {
		ns.Logger.Info("Nuclei command started successfully", map[string]interface{}{
			"task_id":           ns.TaskID,
			"process_id":        cmd.Process.Pid,
			"start_duration_ms": time.Since(cmdStartTime).Milliseconds(),
		})
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

	var cmdErr error
	var exitReason string

	select {
	case <-ctx.Done():
		// Context cancelled, kill the process
		exitReason = "cancelled"
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		wg.Wait()
		cmdErr = fmt.Errorf("scan cancelled")
	case <-timeout:
		// Timeout reached, kill the process
		exitReason = "timeout"
		ns.logOutput("WARN", "Scan timeout reached, terminating process", false)
		if ns.Logger != nil {
			ns.Logger.Warn("Scan timeout reached, terminating process", map[string]interface{}{
				"task_id":    ns.TaskID,
				"timeout":    "30 minutes",
				"process_id": cmd.Process.Pid,
			})
		}
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		wg.Wait()
		cmdErr = fmt.Errorf("scan timeout after 30 minutes")
	case err := <-cmdDone:
		// Command completed
		exitReason = "completed"
		wg.Wait()
		if err != nil {
			cmdErr = fmt.Errorf("nuclei command failed: %w", err)
		}
	}

	// Log command execution results
	executionDuration := time.Since(cmdStartTime)
	totalDuration := time.Since(scanStartTime)

	if ns.Logger != nil {
		cmdInfo := &CommandInfo{
			Executable: ns.NucleiPath,
			Arguments:  cmd.Args[1:], // Exclude executable name
			WorkingDir: cmd.Dir,
			Duration:   executionDuration,
		}

		if cmd.ProcessState != nil {
			cmdInfo.ExitCode = cmd.ProcessState.ExitCode()
		}

		// Get output file size if it exists
		if stat, err := os.Stat(outputFile); err == nil {
			cmdInfo.OutputSize = stat.Size()
		}

		contextData := map[string]interface{}{
			"task_id":            ns.TaskID,
			"exit_reason":        exitReason,
			"execution_duration": executionDuration.String(),
			"total_duration":     totalDuration.String(),
			"process_id":         cmd.Process.Pid,
			"output_file_size":   cmdInfo.OutputSize,
		}

		if cmdErr != nil {
			ns.Logger.Error("Nuclei command execution completed with error", cmdErr, contextData)
		} else {
			ns.Logger.Info("Nuclei command execution completed successfully", contextData)
		}

		ns.Logger.LogCommand(cmdInfo, fmt.Sprintf("Nuclei command %s", exitReason), contextData)
	}

	if cmdErr != nil {
		return cmdErr
	}

	// Save vulnerabilities to output file
	if err := ns.saveVulnerabilities(); err != nil {
		if ns.Logger != nil {
			ns.Logger.Error("Failed to save vulnerabilities", err, map[string]interface{}{
				"task_id":     ns.TaskID,
				"output_file": outputFile,
			})
		}
		return fmt.Errorf("failed to save vulnerabilities: %w", err)
	}

	// Save comprehensive scan result
	if err := ns.saveComprehensiveResult(); err != nil {
		// Log error but don't fail the scan
		if ns.Logger != nil {
			ns.Logger.Error("Failed to save comprehensive result", err, map[string]interface{}{
				"task_id": ns.TaskID,
			})
		}
		fmt.Printf("Warning: failed to save comprehensive result: %v\n", err)
	}

	// Save log summary
	if err := ns.saveLogSummary(); err != nil {
		if ns.Logger != nil {
			ns.Logger.Error("Failed to save log summary", err, map[string]interface{}{
				"task_id": ns.TaskID,
			})
		}
		fmt.Printf("Warning: failed to save log summary: %v\n", err)
	}

	// Clean up temporary directory if it was created
	if ns.TempDir != "" {
		tempManager, err := NewTempManager()
		if err != nil {
			if ns.Logger != nil {
				ns.Logger.Error("Failed to create temp manager for cleanup", err, map[string]interface{}{
					"task_id":  ns.TaskID,
					"temp_dir": ns.TempDir,
				})
			}
			fmt.Printf("Warning: failed to create temp manager for cleanup: %v\n", err)
		} else {
			if err := tempManager.CleanupTempDir(ns.TempDir); err != nil {
				if ns.Logger != nil {
					ns.Logger.Error("Failed to cleanup temporary directory", err, map[string]interface{}{
						"task_id":  ns.TaskID,
						"temp_dir": ns.TempDir,
					})
				}
				fmt.Printf("Warning: failed to cleanup temp directory: %v\n", err)
			}
		}
	}

	if ns.Logger != nil {
		ns.Logger.Info("Nuclei scan completed successfully", map[string]interface{}{
			"task_id":               ns.TaskID,
			"total_duration":        totalDuration.String(),
			"vulnerabilities_found": ns.Progress.FoundVulns,
			"requests_completed":    ns.Progress.CompletedRequests,
		})
	}

	ns.updateProgress("completed", ns.Progress.TotalRequests, ns.Progress.FoundVulns)
	if progressChan != nil {
		progressChan <- ns.GetProgress()
	}

	return nil
}

// buildNucleiCommand builds the nuclei command with all parameters
func (ns *NucleiScanner) buildNucleiCommand(pocs []string, targetsFile, outputFile string) *exec.Cmd {
	startTime := time.Now()

	if ns.Logger != nil {
		ns.Logger.Info("Building nuclei command", map[string]interface{}{
			"poc_count":    len(pocs),
			"targets_file": targetsFile,
			"output_file":  outputFile,
			"nuclei_path":  ns.NucleiPath,
		})
	}

	args := []string{
		"-l", targetsFile, // List of targets
		"-jsonl",         // JSON Lines output
		"-o", outputFile, // Output file
		"-stats",               // Enable stats
		"-stats-interval", "2", // Stats interval
		"-debug",                                  // Debug mode for better logging
		"-nc",                                     // No color
		"-v",                                      // Verbose
		"-project",                                // Enable project mode for better organization
		"-project-path", filepath.Dir(outputFile), // Project path
		"-no-meta",      // Disable metadata in output
		"-include-rr",   // Include request/response in output
		"-include-tags", // Include tags in output
	}

	// Get configuration from TaskManager
	var config *models.Config
	if ns.TaskManager != nil && ns.TaskManager.config != nil {
		config = ns.TaskManager.config
	}

	// Apply configuration parameters
	if config != nil {
		nucleiConfig := config.NucleiConfig

		if ns.Logger != nil {
			ns.Logger.Debug("Applying nuclei configuration", map[string]interface{}{
				"concurrency":        nucleiConfig.Concurrency,
				"bulk_size":          nucleiConfig.BulkSize,
				"rate_limit":         nucleiConfig.RateLimit,
				"proxy_enabled":      nucleiConfig.ProxyEnabled,
				"interactsh_enabled": nucleiConfig.InteractshEnabled,
			})
		}

		// Threading Configuration
		if nucleiConfig.Concurrency > 0 {
			args = append(args, "-c", fmt.Sprintf("%d", nucleiConfig.Concurrency))
		} else {
			args = append(args, "-c", "25") // Default
		}

		if nucleiConfig.BulkSize > 0 {
			args = append(args, "-bulk-size", fmt.Sprintf("%d", nucleiConfig.BulkSize))
		} else {
			args = append(args, "-bulk-size", "25") // Default
		}

		if nucleiConfig.RateLimit > 0 {
			args = append(args, "-rate-limit", fmt.Sprintf("%d", nucleiConfig.RateLimit))
		} else {
			args = append(args, "-rate-limit", "100") // Default
		}

		if nucleiConfig.RateLimitMinute > 0 {
			args = append(args, "-rate-limit-minute", fmt.Sprintf("%d", nucleiConfig.RateLimitMinute))
		}

		// Proxy Configuration
		if nucleiConfig.ProxyEnabled {
			if nucleiConfig.ProxyURL != "" {
				args = append(args, "-proxy-url", nucleiConfig.ProxyURL)
			}
			if len(nucleiConfig.ProxyList) > 0 {
				// Join proxy list with commas
				proxyList := strings.Join(nucleiConfig.ProxyList, ",")
				args = append(args, "-proxy-url", proxyList)
			}
			if nucleiConfig.ProxyInternal {
				args = append(args, "-proxy-internal")
			}
		}

		// DNS/OAST Configuration
		if nucleiConfig.InteractshEnabled {
			if nucleiConfig.InteractshServer != "" {
				args = append(args, "-interactsh-server", nucleiConfig.InteractshServer)
			}
			if nucleiConfig.InteractshToken != "" {
				args = append(args, "-interactsh-token", nucleiConfig.InteractshToken)
			}
		} else if nucleiConfig.InteractshDisable {
			args = append(args, "-no-interactsh")
		}

		// Additional Options
		if nucleiConfig.Retries > 0 {
			args = append(args, "-retries", fmt.Sprintf("%d", nucleiConfig.Retries))
		} else {
			args = append(args, "-retries", "1") // Default
		}

		if nucleiConfig.MaxHostError > 0 {
			args = append(args, "-max-host-error", fmt.Sprintf("%d", nucleiConfig.MaxHostError))
		}

		if nucleiConfig.DisableUpdateCheck {
			args = append(args, "-disable-update-check")
		}

		if nucleiConfig.FollowRedirects {
			args = append(args, "-follow-redirects")
			if nucleiConfig.MaxRedirects > 0 {
				args = append(args, "-max-redirects", fmt.Sprintf("%d", nucleiConfig.MaxRedirects))
			}
		}

		// Use timeout from main config
		if config.Timeout > 0 {
			args = append(args, "-timeout", fmt.Sprintf("%d", config.Timeout))
		} else {
			args = append(args, "-timeout", "10") // Default
		}
	} else {
		// Use default values if no config available
		args = append(args, "-c", "25")
		args = append(args, "-timeout", "10")
		args = append(args, "-retries", "1")
		args = append(args, "-rate-limit", "100")
		args = append(args, "-bulk-size", "25")
		args = append(args, "-disable-update-check")

		if ns.Logger != nil {
			ns.Logger.Warn("No configuration available, using default values", nil)
		}
	}

	// Use temporary directory approach to avoid Windows command line length limits
	if len(pocs) > 100 { // Use temp directory for large template sets
		if ns.Logger != nil {
			ns.Logger.Info("Using temporary directory approach for large POC set", map[string]interface{}{
				"poc_count": len(pocs),
				"threshold": 100,
			})
		}

		tempManager, err := NewTempManager()
		if err != nil {
			if ns.Logger != nil {
				ns.Logger.Error("Failed to create temp manager, falling back to individual templates", err, map[string]interface{}{
					"poc_count": len(pocs),
				})
			}
			fmt.Printf("⚠️  创建临时目录管理器失败，回退到单个模板模式: %v\n", err)
			// Fallback to individual templates
			homeDir, err := os.UserHomeDir()
			if err != nil {
				for _, poc := range pocs {
					args = append(args, "-t", poc)
				}
			} else {
				templatesDir := filepath.Join(homeDir, ".wepoc", "nuclei-templates")
				for _, poc := range pocs {
					templateFile := filepath.Join(templatesDir, poc+".yaml")
					args = append(args, "-t", templateFile)
				}
			}
		} else {
			// Create temporary directory with selected templates
			tempDir, err := tempManager.CreateTempTemplateDir(ns.TaskID, pocs)
			if err != nil {
				if ns.Logger != nil {
					ns.Logger.Error("Failed to create temp template directory, falling back to individual templates", err, map[string]interface{}{
						"task_id":   ns.TaskID,
						"poc_count": len(pocs),
					})
				}
				fmt.Printf("⚠️  创建临时模板目录失败，回退到单个模板模式: %v\n", err)
				// Fallback to individual templates
				homeDir, err := os.UserHomeDir()
				if err != nil {
					for _, poc := range pocs {
						args = append(args, "-t", poc)
					}
				} else {
					templatesDir := filepath.Join(homeDir, ".wepoc", "nuclei-templates")
					for _, poc := range pocs {
						templateFile := filepath.Join(templatesDir, poc+".yaml")
						args = append(args, "-t", templateFile)
					}
				}
			} else {
				// Use directory parameter instead of individual -t parameters
				args = append(args, "-t", tempDir)
				fmt.Printf("🚀 使用临时目录模式: %s (包含 %d 个模板)\n", tempDir, len(pocs))

				if ns.Logger != nil {
					ns.Logger.Info("Successfully created temporary template directory", map[string]interface{}{
						"temp_dir":  tempDir,
						"poc_count": len(pocs),
						"task_id":   ns.TaskID,
					})
				}

				// Store temp directory for cleanup
				ns.TempDir = tempDir
			}
		}
	} else {
		// Use individual templates for smaller sets (< 100 templates)
		if ns.Logger != nil {
			ns.Logger.Debug("Using individual template approach for small POC set", map[string]interface{}{
				"poc_count": len(pocs),
				"threshold": 100,
			})
		}

		homeDir, err := os.UserHomeDir()
		if err != nil {
			for _, poc := range pocs {
				args = append(args, "-t", poc)
			}
		} else {
			templatesDir := filepath.Join(homeDir, ".wepoc", "nuclei-templates")
			for _, poc := range pocs {
				templateFile := filepath.Join(templatesDir, poc+".yaml")
				args = append(args, "-t", templateFile)
			}
		}
	}

	cmd := exec.Command(ns.NucleiPath, args...)

	// Hide the command window on Windows
	if runtime.GOOS == "windows" {
		hideWindowOnWindows(cmd)
	}

	buildDuration := time.Since(startTime)

	if ns.Logger != nil {
		// Log the complete command that will be executed
		cmdInfo := &CommandInfo{
			Executable:  ns.NucleiPath,
			Arguments:   args,
			WorkingDir:  cmd.Dir,
			Environment: make(map[string]string),
		}

		// Capture relevant environment variables
		for _, env := range []string{"PATH", "HOME", "USER", "NUCLEI_CONFIG_DIR"} {
			if value := os.Getenv(env); value != "" {
				cmdInfo.Environment[env] = value
			}
		}

		ns.Logger.LogCommand(cmdInfo, "Nuclei command built successfully", map[string]interface{}{
			"build_duration_ms": buildDuration.Milliseconds(),
			"total_args":        len(args),
			"command_length":    len(strings.Join(args, " ")),
		})

		// Also log the full command string for debugging (but truncate if too long)
		fullCmd := ns.NucleiPath + " " + strings.Join(args, " ")
		if len(fullCmd) > 1000 {
			fullCmd = fullCmd[:1000] + "... (truncated)"
		}

		ns.Logger.Debug("Full nuclei command", map[string]interface{}{
			"command": fullCmd,
		})
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
			TaskID:     int(ns.TaskID),
			Timestamp:  time.Now(),
			Level:      "SUMMARY",
			TemplateID: "",
			Target:     ns.Task.Targets,
			Message: fmt.Sprintf("Scan completed: %d/%d requests, %d vulnerabilities found",
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

	// ... existing code ...
	// Save comprehensive result
	if err := ns.saveComprehensiveResult(); err != nil {
		// Log error but don't fail the scan
		fmt.Printf("Warning: failed to save comprehensive result: %v\n", err)
	}

	// Cleanup temporary directory if used
	if ns.TempDir != "" {
		tempManager, err := NewTempManager()
		if err == nil {
			if err := tempManager.CleanupTempDir(ns.TempDir); err != nil {
				fmt.Printf("⚠️  清理临时目录失败: %v\n", err)
			}
		}
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
