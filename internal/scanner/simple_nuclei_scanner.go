package scanner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"wepoc/internal/models"
)

// ScanEvent represents a real-time scan event
type ScanEvent struct {
	TaskID     int64                  `json:"task_id"`
	EventType  string                 `json:"event_type"` // progress, log, vuln_found, completed, error
	Data       interface{}            `json:"data"`
	Timestamp  time.Time              `json:"timestamp"`
}

// ScanProgress represents real-time scan progress
type ScanProgress struct {
	TaskID            int64   `json:"task_id"`
	TotalRequests     int     `json:"total_requests"`
	CompletedRequests int     `json:"completed_requests"`
	FoundVulns        int     `json:"found_vulns"`
	Percentage        float64 `json:"percentage"`
	Status            string  `json:"status"`
}

// ScanLogEntry represents a log entry with request/response
type ScanLogEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Level       string    `json:"level"` // INFO, WARN, ERROR, DEBUG
	TemplateID  string    `json:"template_id,omitempty"`
	Target      string    `json:"target,omitempty"`
	Message     string    `json:"message"`
	Request     string    `json:"request,omitempty"`
	Response    string    `json:"response,omitempty"`
	IsVulnFound bool      `json:"is_vuln_found"`
}

// SimpleNucleiScanner is a simplified scanner that runs nuclei and saves results to JSON
type SimpleNucleiScanner struct {
	task              *TaskConfig
	manager           *JSONTaskManager
	timeout           time.Duration
	progress          *ScanProgress
	progressMu        sync.RWMutex
	logs              []*ScanLogEntry
	logsMu            sync.Mutex
	eventChannel      chan *ScanEvent
	ctx               context.Context
	lastProgressEmit  time.Time
	lastProgressMu    sync.Mutex
	nucleiPath        string // Add nuclei path configuration
}

// NewSimpleNucleiScanner creates a new simple nuclei scanner
func NewSimpleNucleiScanner(task *TaskConfig, manager *JSONTaskManager) *SimpleNucleiScanner {
	// Get nuclei path from configuration
	nucleiPath := "nuclei" // Default fallback
	if manager != nil && manager.config != nil {
		nucleiPath = manager.config.NucleiPath
		fmt.Printf("🔧 从配置获取 nuclei 路径: %s\n", nucleiPath)
	} else {
		fmt.Printf("⚠️  配置为空，使用默认 nuclei 路径: %s\n", nucleiPath)
	}
	
	return &SimpleNucleiScanner{
		task:    task,
		manager: manager,
		timeout: 30 * time.Minute, // 30 minutes timeout
		progress: &ScanProgress{
			TaskID:            task.ID,
			TotalRequests:     task.TotalRequests,
			CompletedRequests: 0,
			FoundVulns:        0,
			Percentage:        0.0,
			Status:            "pending",
		},
		logs:         make([]*ScanLogEntry, 0),
		eventChannel: make(chan *ScanEvent, 100),
		nucleiPath:   nucleiPath,
	}
}

// SetContext sets the context for event emitting
func (sns *SimpleNucleiScanner) SetContext(ctx context.Context) {
	sns.ctx = ctx
}

// GetEventChannel returns the event channel for frontend subscription
func (sns *SimpleNucleiScanner) GetEventChannel() <-chan *ScanEvent {
	return sns.eventChannel
}

// emitEvent emits an event to the channel
func (sns *SimpleNucleiScanner) emitEvent(eventType string, data interface{}) {
	event := &ScanEvent{
		TaskID:    sns.task.ID,
		EventType: eventType,
		Data:      data,
		Timestamp: time.Now(),
	}

	select {
	case sns.eventChannel <- event:
	default:
		// Channel full, skip event
		fmt.Printf("⚠️  Event channel full, skipping event: %s\n", eventType)
	}
}

// updateProgress updates the scan progress
func (sns *SimpleNucleiScanner) updateProgress(completed int, foundVulns int, status string) {
	sns.progressMu.Lock()
	defer sns.progressMu.Unlock()

	if completed > 0 {
		sns.progress.CompletedRequests = completed
	}
	if foundVulns >= 0 {
		sns.progress.FoundVulns = foundVulns
	}
	if status != "" {
		sns.progress.Status = status
	}

	if sns.progress.TotalRequests > 0 {
		sns.progress.Percentage = float64(sns.progress.CompletedRequests) / float64(sns.progress.TotalRequests) * 100
	}

	// Emit progress event
	sns.emitEvent("progress", sns.progress)
}

// addLog adds a log entry WITHOUT emitting an event (to avoid UI lag)
// Logs are saved to file and can be retrieved after scan completes
func (sns *SimpleNucleiScanner) addLog(level, templateID, target, message, request, response string, isVuln bool) {
	log := &ScanLogEntry{
		Timestamp:   time.Now(),
		Level:       level,
		TemplateID:  templateID,
		Target:      target,
		Message:     message,
		Request:     request,
		Response:    response,
		IsVulnFound: isVuln,
	}

	sns.logsMu.Lock()
	sns.logs = append(sns.logs, log)
	sns.logsMu.Unlock()

	// Don't emit log events during scan to avoid UI lag
	// Logs will be saved to file and retrieved when user views them
}

// Start begins the scanning process
func (sns *SimpleNucleiScanner) Start() error {
	fmt.Printf("\n=== 开始扫描任务 %d ===\n", sns.task.ID)
	fmt.Printf("任务名称: %s\n", sns.task.Name)
	fmt.Printf("目标数量: %d\n", len(sns.task.Targets))
	fmt.Printf("模板数量: %d\n", len(sns.task.POCs))
	fmt.Printf("预计请求数: %d\n", sns.task.TotalRequests)
	fmt.Printf("开始时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))

	// Update status to running
	sns.updateProgress(0, 0, "running")
	sns.addLog("INFO", "", "", fmt.Sprintf("开始扫描任务: %s", sns.task.Name), "", "", false)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), sns.timeout)
	defer cancel()

	// Prepare output file
	outputFile := sns.task.OutputFile
	fmt.Printf("输出文件: %s\n", outputFile)
	if err := sns.prepareOutputFile(outputFile); err != nil {
		fmt.Printf("❌ 准备输出文件失败: %v\n", err)
		sns.addLog("ERROR", "", "", fmt.Sprintf("准备输出文件失败: %v", err), "", "", false)
		return fmt.Errorf("failed to prepare output file: %w", err)
	}

	// Create targets file
	targetsFile, err := sns.createTargetsFile()
	if err != nil {
		fmt.Printf("❌ 创建目标文件失败: %v\n", err)
		sns.addLog("ERROR", "", "", fmt.Sprintf("创建目标文件失败: %v", err), "", "", false)
		return fmt.Errorf("failed to create targets file: %w", err)
	}
	defer os.Remove(targetsFile)
	fmt.Printf("目标文件: %s\n", targetsFile)

	// Build nuclei command with -debug flag
	cmd := sns.buildNucleiCommand(targetsFile, outputFile)
	cmd.Dir = filepath.Dir(outputFile)

	// Print command details
	fmt.Printf("\n=== Nuclei 命令信息 ===\n")
	fmt.Printf("工作目录: %s\n", cmd.Dir)
	fmt.Printf("执行命令: %s %s\n", cmd.Path, strings.Join(cmd.Args[1:], " "))
	fmt.Printf("超时时间: %v\n", sns.timeout)
	fmt.Printf("========================\n\n")

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start nuclei process
	fmt.Printf("🚀 启动 Nuclei 扫描进程...\n")
	sns.addLog("INFO", "", "", "启动 Nuclei 扫描进程", "", "", false)

	if err := cmd.Start(); err != nil {
		fmt.Printf("❌ 启动 Nuclei 失败: %v\n", err)
		fmt.Printf("🔧 使用的 nuclei 路径: %s\n", sns.nucleiPath)
		fmt.Printf("🔧 工作目录: %s\n", cmd.Dir)
		fmt.Printf("🔧 环境变量 PATH: %s\n", os.Getenv("PATH"))
		
		// Check if nuclei file exists and is executable
		if _, err := os.Stat(sns.nucleiPath); os.IsNotExist(err) {
			fmt.Printf("❌ Nuclei 文件不存在: %s\n", sns.nucleiPath)
		} else {
			fmt.Printf("✅ Nuclei 文件存在: %s\n", sns.nucleiPath)
		}
		
		// Save error to log file
		sns.logError("启动 Nuclei 失败", err, sns.nucleiPath, cmd.Dir)
		
		sns.addLog("ERROR", "", "", fmt.Sprintf("启动 Nuclei 失败: %v", err), "", "", false)
		return fmt.Errorf("failed to start nuclei: %w", err)
	}
	fmt.Printf("✅ Nuclei 进程已启动 (PID: %d)\n", cmd.Process.Pid)

	// Monitor output in goroutines
	var wg sync.WaitGroup

	// Monitor stdout for stats and debug output
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		sns.monitorStdout(scanner)
	}()

	// Monitor stderr for errors
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		sns.monitorStderr(scanner)
	}()

	// Wait for completion or timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		// Timeout reached, kill the process
		fmt.Printf("⏰ 扫描超时 (%v)，终止进程...\n", sns.timeout)
		sns.addLog("WARN", "", "", fmt.Sprintf("扫描超时 (%v)，终止进程", sns.timeout), "", "", false)
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		wg.Wait()
		return fmt.Errorf("scan timeout after %v", sns.timeout)
	case err := <-done:
		if err != nil {
			fmt.Printf("❌ Nuclei 命令执行失败: %v\n", err)
			sns.logError("Nuclei 命令执行失败", err, sns.nucleiPath, cmd.Dir)
			sns.addLog("ERROR", "", "", fmt.Sprintf("Nuclei 命令执行失败: %v", err), "", "", false)
			return fmt.Errorf("nuclei command failed: %w", err)
		}
		fmt.Printf("✅ Nuclei 进程正常结束\n")
	}

	// Process results
	fmt.Printf("\n=== 处理扫描结果 ===\n")
	sns.addLog("INFO", "", "", "开始处理扫描结果", "", "", false)

	if err := sns.processResults(outputFile); err != nil {
		fmt.Printf("❌ 处理结果失败: %v\n", err)
		sns.addLog("ERROR", "", "", fmt.Sprintf("处理结果失败: %v", err), "", "", false)
		return err
	}

	// Save logs to file
	if err := sns.saveLogs(); err != nil {
		fmt.Printf("⚠️  保存日志失败: %v\n", err)
	}

	// Update final status
	sns.updateProgress(sns.progress.TotalRequests, sns.progress.FoundVulns, "completed")
	sns.emitEvent("completed", sns.progress)

	fmt.Printf("🎉 扫描任务 %d 完成！\n", sns.task.ID)
	sns.addLog("INFO", "", "", "扫描任务完成", "", "", false)

	// Close event channel
	close(sns.eventChannel)

	return nil
}

// stripAnsiCodes removes ANSI color codes from a string
func stripAnsiCodes(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(s, "")
}

// monitorStdout monitors the nuclei stdout for stats, debug logs, and progress
func (sns *SimpleNucleiScanner) monitorStdout(stdout *bufio.Scanner) {
	var currentRequest, currentResponse strings.Builder
	var currentTemplate, currentTarget string
	inRequest, inResponse := false, false

	for stdout.Scan() {
		rawLine := stdout.Text()
		line := stripAnsiCodes(rawLine) // Remove ANSI color codes

		// Parse JSON output (vulnerability findings and stats)
		if strings.HasPrefix(line, "{") {
			var jsonData map[string]interface{}
			if err := json.Unmarshal([]byte(line), &jsonData); err == nil {
				// Check if this is a stats JSON
				if _, hasRequests := jsonData["requests"]; hasRequests {
					sns.parseStatsLine(line)
					continue
				}

				// Check if this is a vulnerability finding
				if templateID, ok := jsonData["template-id"].(string); ok {
					// This is a vulnerability finding!
					sns.progressMu.Lock()
					sns.progress.FoundVulns++
					currentVulns := sns.progress.FoundVulns
					sns.progressMu.Unlock()

					// Get vulnerability details
					vulnHost := ""
					vulnName := ""
					vulnSeverity := "unknown"

					if host, ok := jsonData["host"].(string); ok {
						vulnHost = host
					}
					if info, ok := jsonData["info"].(map[string]interface{}); ok {
						if name, ok := info["name"].(string); ok {
							vulnName = name
						}
						if severity, ok := info["severity"].(string); ok {
							vulnSeverity = severity
						}
					}

					fmt.Printf("🐛 发现漏洞 #%d: [%s] %s - %s (目标: %s)\n",
						currentVulns, vulnSeverity, templateID, vulnName, vulnHost)

					// Immediately emit progress update to show vuln count
					sns.emitEvent("progress", sns.progress)

					// Emit vulnerability found event for real-time notification
					sns.emitEvent("vuln_found", map[string]interface{}{
						"vuln_number": currentVulns,
						"template_id": templateID,
						"name":        vulnName,
						"severity":    vulnSeverity,
						"host":        vulnHost,
						"timestamp":   time.Now().Format("15:04:05"),
					})

					// Log the vulnerability
					sns.addLog("VULN", templateID, vulnHost,
						fmt.Sprintf("[%s] %s - %s", vulnSeverity, templateID, vulnName), "", "", true)
				}
			}
			continue
		}

		// Parse request/response from debug output
		if strings.Contains(line, "Dumped HTTP request for") {
			// Extract template and target
			re := regexp.MustCompile(`\[([^\]]+)\] Dumped HTTP request for (https?://[^\s]+)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 2 {
				currentTemplate = matches[1]
				currentTarget = matches[2]
				inRequest = true
				currentRequest.Reset()
			}
			continue
		}

		// Detect HTTP request start
		if strings.HasPrefix(line, "GET ") || strings.HasPrefix(line, "POST ") ||
		   strings.HasPrefix(line, "PUT ") || strings.HasPrefix(line, "DELETE ") ||
		   strings.HasPrefix(line, "PATCH ") || strings.HasPrefix(line, "HEAD ") ||
		   strings.HasPrefix(line, "OPTIONS ") {
			inRequest = true
			currentRequest.Reset()
			currentRequest.WriteString(line + "\n")
			continue
		}

		// Detect HTTP response start
		if strings.HasPrefix(line, "HTTP/") {
			inRequest = false
			inResponse = true
			currentResponse.Reset()
			currentResponse.WriteString(line + "\n")
			continue
		}

		// Collect request/response lines
		if inRequest {
			currentRequest.WriteString(line + "\n")
			// Request ends at empty line
			if strings.TrimSpace(line) == "" {
				inRequest = false
			}
		} else if inResponse {
			currentResponse.WriteString(line + "\n")
			// Response ends at empty line
			if strings.TrimSpace(line) == "" && currentResponse.Len() > 50 {
				inResponse = false
				// Only log complete HTTP request/response pairs
				if currentRequest.Len() > 0 && currentResponse.Len() > 0 {
					// Save to logs for later viewing
					sns.addLog("DEBUG", currentTemplate, currentTarget,
						fmt.Sprintf("%s -> %s", currentTemplate, currentTarget),
						currentRequest.String(), currentResponse.String(), false)
				}
				currentRequest.Reset()
				currentResponse.Reset()
			}
		}
	}
}

// monitorStderr monitors the nuclei stderr for errors
func (sns *SimpleNucleiScanner) monitorStderr(stderr *bufio.Scanner) {
	for stderr.Scan() {
		line := stderr.Text()
		fmt.Printf("[Nuclei Error] %s\n", line)
		sns.addLog("ERROR", "", "", line, "", "", false)
	}
}

// parseStatsLine parses the JSON stats output from nuclei
func (sns *SimpleNucleiScanner) parseStatsLine(line string) {
	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(line), &stats); err != nil {
		return
	}

	completed := 0
	total := 0
	matched := 0

	if requests, ok := stats["requests"]; ok {
		if val, err := parseNumericValue(requests); err == nil {
			completed = val
		}
	}
	if totalVal, ok := stats["total"]; ok {
		if val, err := parseNumericValue(totalVal); err == nil {
			total = val
		}
	}
	if matchedVal, ok := stats["matched"]; ok {
		if val, err := parseNumericValue(matchedVal); err == nil {
			matched = val
		}
	}

	// Update progress
	sns.progressMu.Lock()
	if total > 0 {
		sns.progress.TotalRequests = total
	}
	sns.progress.CompletedRequests = completed
	sns.progress.FoundVulns = matched
	if sns.progress.TotalRequests > 0 {
		sns.progress.Percentage = float64(completed) / float64(sns.progress.TotalRequests) * 100
	}
	sns.progressMu.Unlock()

	// Emit progress event more frequently - every 0.02 seconds or every request
	// This ensures frontend gets real-time updates even for very fast scans
	sns.lastProgressMu.Lock()
	now := time.Now()
	timeSinceLastEmit := now.Sub(sns.lastProgressEmit)
	shouldEmit := timeSinceLastEmit >= 20*time.Millisecond || completed%1 == 0 // Emit every 0.02 seconds OR every request

	if shouldEmit {
		sns.lastProgressEmit = now
		sns.lastProgressMu.Unlock()

		// Emit progress event
		sns.emitEvent("progress", sns.progress)

		fmt.Printf("📊 进度: %d/%d (%.1f%%), 发现漏洞: %d\n",
			completed, sns.progress.TotalRequests, sns.progress.Percentage, matched)
	} else {
		sns.lastProgressMu.Unlock()
	}
}

// prepareOutputFile creates the output directory if it doesn't exist
func (sns *SimpleNucleiScanner) prepareOutputFile(outputFile string) error {
	dir := filepath.Dir(outputFile)
	return os.MkdirAll(dir, 0755)
}

// createTargetsFile creates a temporary file with target URLs
func (sns *SimpleNucleiScanner) createTargetsFile() (string, error) {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "wepoc-targets-*.txt")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// Write targets to file
	for _, target := range sns.task.Targets {
		if _, err := tmpFile.WriteString(target + "\n"); err != nil {
			os.Remove(tmpFile.Name())
			return "", err
		}
	}

	return tmpFile.Name(), nil
}

// buildNucleiCommand builds the nuclei command with -debug flag
func (sns *SimpleNucleiScanner) buildNucleiCommand(targetsFile, outputFile string) *exec.Cmd {
	// Build command arguments - following user's specification
	args := []string{
		"-l", targetsFile,                    // Target list file
		"-jle", outputFile,                   // JSONL export to file (matches user's spec)
		"-jsonl",                             // Also output JSONL to stdout for real-time parsing
		"-stats",                             // Show statistics
		"-stats-interval", "2",               // Stats interval (as specified)
		"-debug",                             // Debug mode to get request/response (KEY FEATURE)
		"-timeout", "30",                     // HTTP timeout per request (30 seconds) - CRITICAL for preventing hangs
		"-retries", "1",                      // Retry failed requests once
		"-nc",                                // No color output
		"-v",                                 // Verbose
	}

	// Add template files (skip existence check for performance)
	fmt.Printf("使用的模板文件:\n")
	for _, poc := range sns.task.POCs {
		// Check if poc is already an absolute path
		var templateFile string
		if filepath.IsAbs(poc) {
			// It's already an absolute path from frontend
			templateFile = poc
		} else {
			// It's a relative path, add base directory
			homeDir, _ := os.UserHomeDir()
			templatesDir := filepath.Join(homeDir, ".wepoc", "nuclei-templates")

			// Check if it has .yaml extension
			if strings.HasSuffix(poc, ".yaml") || strings.HasSuffix(poc, ".yml") {
				templateFile = filepath.Join(templatesDir, poc)
			} else {
				templateFile = filepath.Join(templatesDir, poc+".yaml")
			}
		}

		// Add template file directly without checking existence (already validated during import)
		args = append(args, "-t", templateFile)
		fmt.Printf("  📄 %s\n", templateFile)
	}
	fmt.Printf("模板数量: %d\n", len(sns.task.POCs))

	// Log the command being executed for debugging
	fmt.Printf("🔧 执行命令: %s %v\n", sns.nucleiPath, args)
	
	// Save debug info to log file
	sns.logDebugInfo(sns.nucleiPath, args, outputFile)
	
	cmd := exec.Command(sns.nucleiPath, args...)
	
	// Set working directory for Windows compatibility
	cmd.Dir = filepath.Dir(outputFile)
	
	// Set environment variables for Windows
	if runtime.GOOS == "windows" {
		// Add nuclei directory to PATH
		nucleiDir := filepath.Dir(sns.nucleiPath)
		currentPath := os.Getenv("PATH")
		newPath := nucleiDir + ";" + currentPath
		cmd.Env = append(os.Environ(), "PATH="+newPath)
		
		// Also set working directory to nuclei directory for better compatibility
		cmd.Dir = nucleiDir
		
		// Hide the command window on Windows
		hideWindowOnWindows(cmd)
	}
	
	return cmd
}

// logDebugInfo saves debug information to log file
func (sns *SimpleNucleiScanner) logDebugInfo(nucleiPath string, args []string, outputFile string) {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("❌ 无法获取用户目录: %v\n", err)
		return
	}
	
	// Create logs directory
	logsDir := filepath.Join(homeDir, ".wepoc", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		fmt.Printf("❌ 无法创建日志目录: %v\n", err)
		return
	}
	
	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFile := filepath.Join(logsDir, fmt.Sprintf("scan_debug_%d_%s.log", sns.task.ID, timestamp))
	
	// Open log file
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("❌ 无法创建日志文件: %v\n", err)
		return
	}
	defer file.Close()
	
	// Write debug information
	fmt.Fprintf(file, "=== Nuclei 扫描调试信息 ===\n")
	fmt.Fprintf(file, "时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "任务ID: %d\n", sns.task.ID)
	fmt.Fprintf(file, "任务名称: %s\n", sns.task.Name)
	fmt.Fprintf(file, "操作系统: %s\n", runtime.GOOS)
	fmt.Fprintf(file, "架构: %s\n", runtime.GOARCH)
	fmt.Fprintf(file, "\n")
	
	fmt.Fprintf(file, "=== Nuclei 配置 ===\n")
	fmt.Fprintf(file, "Nuclei 路径: %s\n", nucleiPath)
	fmt.Fprintf(file, "输出文件: %s\n", outputFile)
	fmt.Fprintf(file, "工作目录: %s\n", filepath.Dir(outputFile))
	fmt.Fprintf(file, "\n")
	
	fmt.Fprintf(file, "=== 执行命令 ===\n")
	fmt.Fprintf(file, "命令: %s %v\n", nucleiPath, args)
	fmt.Fprintf(file, "\n")
	
	fmt.Fprintf(file, "=== 环境变量 ===\n")
	fmt.Fprintf(file, "PATH: %s\n", os.Getenv("PATH"))
	fmt.Fprintf(file, "HOME: %s\n", os.Getenv("HOME"))
	fmt.Fprintf(file, "USERPROFILE: %s\n", os.Getenv("USERPROFILE"))
	fmt.Fprintf(file, "\n")
	
	fmt.Fprintf(file, "=== 文件检查 ===\n")
	if _, err := os.Stat(nucleiPath); os.IsNotExist(err) {
		fmt.Fprintf(file, "❌ Nuclei 文件不存在: %s\n", nucleiPath)
	} else {
		fmt.Fprintf(file, "✅ Nuclei 文件存在: %s\n", nucleiPath)
		// Check if it's executable
		if info, err := os.Stat(nucleiPath); err == nil {
			fmt.Fprintf(file, "文件大小: %d 字节\n", info.Size())
			fmt.Fprintf(file, "文件权限: %s\n", info.Mode().String())
		}
	}
	fmt.Fprintf(file, "\n")
	
	fmt.Fprintf(file, "=== 模板文件列表 ===\n")
	for i, poc := range sns.task.POCs {
		var templateFile string
		if filepath.IsAbs(poc) {
			templateFile = poc
		} else {
			templatesDir := filepath.Join(homeDir, ".wepoc", "nuclei-templates")
			if strings.HasSuffix(poc, ".yaml") {
				templateFile = filepath.Join(templatesDir, poc)
			} else {
				templateFile = filepath.Join(templatesDir, poc+".yaml")
			}
		}
		
		// 只记录模板文件路径，不检查存在性（提升性能）
		fmt.Fprintf(file, "📄 模板 %d: %s\n", i+1, templateFile)
	}
	fmt.Fprintf(file, "\n")
	
	fmt.Printf("📝 调试信息已保存到: %s\n", logFile)
}

// logError saves error information to log file
func (sns *SimpleNucleiScanner) logError(message string, err error, nucleiPath, workDir string) {
	// Get home directory
	homeDir, err2 := os.UserHomeDir()
	if err2 != nil {
		fmt.Printf("❌ 无法获取用户目录: %v\n", err2)
		return
	}
	
	// Create logs directory
	logsDir := filepath.Join(homeDir, ".wepoc", "logs")
	if err2 := os.MkdirAll(logsDir, 0755); err2 != nil {
		fmt.Printf("❌ 无法创建日志目录: %v\n", err2)
		return
	}
	
	// Create error log file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFile := filepath.Join(logsDir, fmt.Sprintf("scan_error_%d_%s.log", sns.task.ID, timestamp))
	
	// Open log file
	file, err2 := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err2 != nil {
		fmt.Printf("❌ 无法创建错误日志文件: %v\n", err2)
		return
	}
	defer file.Close()
	
	// Write error information
	fmt.Fprintf(file, "=== Nuclei 扫描错误信息 ===\n")
	fmt.Fprintf(file, "时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "任务ID: %d\n", sns.task.ID)
	fmt.Fprintf(file, "任务名称: %s\n", sns.task.Name)
	fmt.Fprintf(file, "错误消息: %s\n", message)
	fmt.Fprintf(file, "错误详情: %v\n", err)
	fmt.Fprintf(file, "\n")
	
	fmt.Fprintf(file, "=== 环境信息 ===\n")
	fmt.Fprintf(file, "操作系统: %s\n", runtime.GOOS)
	fmt.Fprintf(file, "架构: %s\n", runtime.GOARCH)
	fmt.Fprintf(file, "Nuclei 路径: %s\n", nucleiPath)
	fmt.Fprintf(file, "工作目录: %s\n", workDir)
	fmt.Fprintf(file, "PATH: %s\n", os.Getenv("PATH"))
	fmt.Fprintf(file, "\n")
	
	fmt.Fprintf(file, "=== 文件检查 ===\n")
	if _, err := os.Stat(nucleiPath); os.IsNotExist(err) {
		fmt.Fprintf(file, "❌ Nuclei 文件不存在: %s\n", nucleiPath)
	} else {
		fmt.Fprintf(file, "✅ Nuclei 文件存在: %s\n", nucleiPath)
		if info, err := os.Stat(nucleiPath); err == nil {
			fmt.Fprintf(file, "文件大小: %d 字节\n", info.Size())
			fmt.Fprintf(file, "文件权限: %s\n", info.Mode().String())
		}
	}
	fmt.Fprintf(file, "\n")
	
	fmt.Printf("📝 错误信息已保存到: %s\n", logFile)
}

// processResults processes the nuclei output and creates a result file
func (sns *SimpleNucleiScanner) processResults(outputFile string) error {
	// Check if output file exists
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		fmt.Printf("📄 输出文件不存在，创建空结果...\n")
		// No output file means no vulnerabilities found
		return sns.createEmptyResult()
	}

	fmt.Printf("📄 读取输出文件: %s\n", outputFile)
	
	// Read and parse the JSONL output
	vulnerabilities, err := sns.parseJSONLOutput(outputFile)
	if err != nil {
		fmt.Printf("❌ 解析输出文件失败: %v\n", err)
		return fmt.Errorf("failed to parse output: %w", err)
	}

	fmt.Printf("🔍 发现漏洞数量: %d\n", len(vulnerabilities))
	
	// Print vulnerability details
	for i, vuln := range vulnerabilities {
		fmt.Printf("  %d. %s - %s\n", i+1, vuln.TemplateID, vuln.Info.Name)
		fmt.Printf("     目标: %s\n", vuln.MatchedAt)
		fmt.Printf("     严重程度: %s\n", vuln.Info.Severity)
	}

	// Create result object
	result := &TaskResult{
		TaskID:           sns.task.ID,
		TaskName:         sns.task.Name,
		Status:           "completed",
		StartTime:        sns.task.StartTime,
		EndTime:          time.Now(),
		Duration:         time.Since(sns.task.StartTime).String(),
		Targets:          sns.task.Targets,
		Templates:        sns.task.POCs,
		TemplateCount:    len(sns.task.POCs),
		TargetCount:      len(sns.task.Targets),
		TotalRequests:    sns.task.TotalRequests,
		CompletedRequests: sns.task.TotalRequests, // Assume all completed
		FoundVulns:       len(vulnerabilities),
		SuccessRate:      100.0, // Simplified
		Vulnerabilities:  vulnerabilities,
		Summary: map[string]interface{}{
			"total_requests":    sns.task.TotalRequests,
			"completed_requests": sns.task.TotalRequests,
			"found_vulns":       len(vulnerabilities),
			"duration":          time.Since(sns.task.StartTime).String(),
			"success_rate":      100.0,
		},
		CreatedAt: time.Now(),
	}

	fmt.Printf("💾 保存结果到文件...\n")
	// Save result to JSON file
	if err := sns.saveResult(result); err != nil {
		fmt.Printf("❌ 保存结果失败: %v\n", err)
		return err
	}
	
	fmt.Printf("✅ 结果已保存到: %s\n", filepath.Join(sns.manager.resultsDir, fmt.Sprintf("task_%d_result.json", result.TaskID)))
	return nil
}

// parseJSONLOutput parses the JSONL output file
func (sns *SimpleNucleiScanner) parseJSONLOutput(outputFile string) ([]*models.NucleiResult, error) {
	file, err := os.Open(outputFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var vulnerabilities []*models.NucleiResult
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var result models.NucleiResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			// Skip invalid JSON lines
			continue
		}

		// Only include results with vulnerabilities
		if result.MatchedAt != "" {
			vulnerabilities = append(vulnerabilities, &result)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return vulnerabilities, nil
}

// createEmptyResult creates an empty result when no vulnerabilities are found
func (sns *SimpleNucleiScanner) createEmptyResult() error {
	fmt.Printf("🔍 未发现漏洞，创建空结果...\n")
	
	result := &TaskResult{
		TaskID:           sns.task.ID,
		TaskName:         sns.task.Name,
		Status:           "completed",
		StartTime:        sns.task.StartTime,
		EndTime:          time.Now(),
		Duration:         time.Since(sns.task.StartTime).String(),
		Targets:          sns.task.Targets,
		Templates:        sns.task.POCs,
		TemplateCount:    len(sns.task.POCs),
		TargetCount:      len(sns.task.Targets),
		TotalRequests:    sns.task.TotalRequests,
		CompletedRequests: sns.task.TotalRequests,
		FoundVulns:       0,
		SuccessRate:      100.0,
		Vulnerabilities:  []*models.NucleiResult{},
		Summary: map[string]interface{}{
			"total_requests":    sns.task.TotalRequests,
			"completed_requests": sns.task.TotalRequests,
			"found_vulns":       0,
			"duration":          time.Since(sns.task.StartTime).String(),
			"success_rate":      100.0,
		},
		CreatedAt: time.Now(),
	}

	fmt.Printf("💾 保存空结果到文件...\n")
	if err := sns.saveResult(result); err != nil {
		fmt.Printf("❌ 保存空结果失败: %v\n", err)
		return err
	}
	
	fmt.Printf("✅ 空结果已保存到: %s\n", filepath.Join(sns.manager.resultsDir, fmt.Sprintf("task_%d_result.json", result.TaskID)))
	return nil
}

// saveResult saves the result to a JSON file
func (sns *SimpleNucleiScanner) saveResult(result *TaskResult) error {
	// Create result file path
	resultFile := filepath.Join(sns.manager.resultsDir, fmt.Sprintf("task_%d_result.json", result.TaskID))

	// Marshal to JSON
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(resultFile, data, 0644)
}

// saveLogs saves the logs to a JSON file
func (sns *SimpleNucleiScanner) saveLogs() error {
	sns.logsMu.Lock()
	defer sns.logsMu.Unlock()

	// Create log file path
	logFile := filepath.Join(sns.manager.logsDir, fmt.Sprintf("task_%d.json", sns.task.ID))

	// Marshal logs to JSON
	data, err := json.MarshalIndent(sns.logs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal logs: %w", err)
	}

	// Write to file
	if err := os.WriteFile(logFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write log file: %w", err)
	}

	fmt.Printf("💾 日志已保存到: %s (%d 条记录)\n", logFile, len(sns.logs))
	return nil
}

