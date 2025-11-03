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
	"strconv"
	"strings"
	"sync"
	"time"

	"wepoc/internal/models"
)

// ScanEvent represents a real-time scan event
type ScanEvent struct {
	TaskID    int64       `json:"task_id"`
	EventType string      `json:"event_type"` // progress, log, vuln_found, completed, error
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// ScanProgress represents real-time scan progress
type ScanProgress struct {
	TaskID            int64   `json:"task_id"`
	TotalRequests     int     `json:"total_requests"`
	CompletedRequests int     `json:"completed_requests"`
	FoundVulns        int     `json:"found_vulns"`
	Percentage        float64 `json:"percentage"`
	Status            string  `json:"status"`
	CurrentTemplate   string  `json:"current_template"`   // 当前扫描的POC模板
	CurrentTarget     string  `json:"current_target"`     // 当前扫描的目标
	TotalTemplates    int     `json:"total_templates"`    // 任务中的POC总数量（用户选择的）
	CompletedTemplates int    `json:"completed_templates"` // 已扫描的POC数量（成功+失败）
	ScannedTemplates  int     `json:"scanned_templates"`   // 实际扫描过的模板数量（包括失败的）
	FailedTemplates   int     `json:"failed_templates"`    // 扫描失败的模板数量
	FilteredTemplates int     `json:"filtered_templates"`  // 被Nuclei过滤的模板数量（code/headless等）
	SkippedTemplates  int     `json:"skipped_templates"`   // 被跳过的模板数量（条件不符）
	CurrentIndex      int     `json:"current_index"`       // 当前模板在选择列表中的序号（1-based）
	SelectedTemplates []string `json:"selected_templates"`  // 用户选择的所有模板ID
	ScannedTemplateIDs []string `json:"scanned_template_ids"` // 已扫描模板的ID集合（按首次出现顺序）
	FailedTemplateIDs  []string `json:"failed_template_ids"`  // 扫描失败模板的ID集合
	FilteredTemplateIDs []string `json:"filtered_template_ids"` // 被过滤模板的ID集合
	SkippedTemplateIDs  []string `json:"skipped_template_ids"`  // 被跳过模板的ID集合
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

// HTTPRequestLog represents a single HTTP request/response in scan task
// 用于在前端以列表形式展示每个HTTP请求
type HTTPRequestLog struct {
	ID          int64     `json:"id"`           // 请求序号
	TaskID      int64     `json:"task_id"`      // 所属任务ID
	Timestamp   time.Time `json:"timestamp"`    // 请求时间
	TemplateID  string    `json:"template_id"`  // POC模板ID
	TemplateName string   `json:"template_name"` // POC模板名称
	Severity    string    `json:"severity"`     // 严重程度
	Target      string    `json:"target"`       // 目标URL
	Method      string    `json:"method"`       // HTTP方法（GET/POST等）
	StatusCode  int       `json:"status_code"`  // HTTP状态码
	IsVulnFound bool      `json:"is_vuln_found"` // 是否发现漏洞
	Request     string    `json:"request"`      // 完整请求包
	Response    string    `json:"response"`     // 完整响应包
	Duration    int64     `json:"duration_ms"`  // 请求耗时（毫秒）
}

// SimpleNucleiScanner is a simplified scanner that runs nuclei and saves results to JSON
type SimpleNucleiScanner struct {
	task             *TaskConfig
	manager          *JSONTaskManager
	timeout          time.Duration
	progress         *ScanProgress
	progressMu       sync.RWMutex
	logs             []*ScanLogEntry
	logsMu           sync.Mutex
	httpRequestLogs  []*HTTPRequestLog // 新增：HTTP请求日志列表
	httpLogsMu       sync.Mutex        // 新增：HTTP请求日志互斥锁
	requestCounter   int64             // 新增：请求计数器
	eventChannel     chan *ScanEvent
	ctx              context.Context
	lastProgressEmit time.Time
	lastProgressMu   sync.Mutex
	nucleiPath       string          // Add nuclei path configuration
	tempDir          string          // Temporary directory for templates
	logger           *EnhancedLogger // Enhanced logger for detailed logging
	templateSet       map[string]bool   // 用于跟踪已扫描的模板（成功+失败）
	templateSetMu     sync.Mutex        // 保护templateSet的互斥锁
	failedTemplates   map[string]bool   // 用于跟踪扫描失败的模板
	failedTemplatesMu sync.Mutex        // 保护failedTemplates的互斥锁
	templateIndex     map[string]int    // 模板ID到选择顺序索引的映射（0-based）
	templateSeverity  map[string]string // 模板ID到严重性的映射
	templateSevMu     sync.Mutex        // 保护templateSeverity的互斥锁
	debugLogFile      string            // Debug log file path for nuclei output
}

// NewSimpleNucleiScanner creates a new simple nuclei scanner
func NewSimpleNucleiScanner(task *TaskConfig, manager *JSONTaskManager) *SimpleNucleiScanner {
	// Initialize enhanced logger
	logger, err := NewEnhancedLogger(task.ID, "SimpleNucleiScanner")
	if err != nil {
		fmt.Printf("⚠️ Failed to create enhanced logger: %v\n", err)
		logger = nil
	}

	// Get nuclei path from configuration
	nucleiPath := "nuclei" // Default fallback
	if manager != nil && manager.config != nil {
		nucleiPath = manager.config.NucleiPath
	}

	// 构建模板索引映射
	idx := make(map[string]int)
	for i, tid := range task.POCs {
		idx[tid] = i
	}

	scanner := &SimpleNucleiScanner{
		task:             task,
		manager:          manager,
		timeout:          30 * time.Minute, // Default timeout
		progress:         &ScanProgress{TaskID: task.ID, Status: "pending", TotalTemplates: len(task.POCs), SelectedTemplates: append([]string{}, task.POCs...)},
		logs:             make([]*ScanLogEntry, 0),
		eventChannel:     make(chan *ScanEvent, 100),
		nucleiPath:       nucleiPath, // Use nuclei path from configuration
		logger:           logger,
		templateSet:      make(map[string]bool),   // 初始化模板跟踪集合
		failedTemplates:  make(map[string]bool),   // 初始化失败模板跟踪集合
		templateIndex:    idx,
		templateSeverity: make(map[string]string), // 初始化模板严重性映射
	}

	// Log scanner initialization
	if logger != nil {
		logger.Info("SimpleNucleiScanner initialized", map[string]interface{}{
			"task_id":      task.ID,
			"task_name":    task.Name,
			"nuclei_path":  scanner.nucleiPath,
			"timeout":      scanner.timeout.String(),
			"poc_count":    len(task.POCs),
			"target_count": len(task.Targets),
		})
	}

	return scanner
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

	// 对于完成状态的事件，使用阻塞发送确保一定被接收
	if eventType == "progress" {
		if progress, ok := data.(*ScanProgress); ok && progress.Status == "completed" {
			fmt.Printf("📤 强制发送完成事件（阻塞模式）\n")
			sns.eventChannel <- event // 阻塞发送，确保completed事件一定被接收
			return
		}
	}

	// 其他事件使用非阻塞发送
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
		if status == "completed" {
			fmt.Printf("🎯 updateProgress设置状态为completed\n")
		}
	}

	if sns.progress.TotalRequests > 0 {
		sns.progress.Percentage = float64(sns.progress.CompletedRequests) / float64(sns.progress.TotalRequests) * 100
	}

	// Emit progress event
	if status == "completed" {
		fmt.Printf("🎯 updateProgress发送completed事件\n")
	}
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

// getTemplateSeverity retrieves the severity for a template ID from cache
func (sns *SimpleNucleiScanner) getTemplateSeverity(templateID string) string {
	sns.templateSevMu.Lock()
	defer sns.templateSevMu.Unlock()

	if sev, ok := sns.templateSeverity[templateID]; ok {
		return sev
	}
	return "info" // 默认返回info
}

// setTemplateSeverity caches the severity for a template ID
func (sns *SimpleNucleiScanner) setTemplateSeverity(templateID, severity string) {
	sns.templateSevMu.Lock()
	defer sns.templateSevMu.Unlock()
	sns.templateSeverity[templateID] = severity
}

// addHTTPRequestLog records a single HTTP request/response for display in frontend table
func (sns *SimpleNucleiScanner) addHTTPRequestLog(templateID, templateName, severity, target, method string, statusCode int, request, response string, isVuln bool, duration int64) {
	sns.httpLogsMu.Lock()
	defer sns.httpLogsMu.Unlock()

	sns.requestCounter++

	httpLog := &HTTPRequestLog{
		ID:           sns.requestCounter,
		TaskID:       sns.task.ID,
		Timestamp:    time.Now(),
		TemplateID:   templateID,
		TemplateName: templateName,
		Severity:     severity,
		Target:       target,
		Method:       method,
		StatusCode:   statusCode,
		IsVulnFound:  isVuln,
		Request:      request,
		Response:     response,
		Duration:     duration,
	}

	sns.httpRequestLogs = append(sns.httpRequestLogs, httpLog)

	// 实时发送到前端（用于实时列表更新）- 但不发送完整请求/响应包以节省带宽
	sns.emitEvent("http_request", map[string]interface{}{
		"id":            httpLog.ID,
		"task_id":       httpLog.TaskID,
		"timestamp":     httpLog.Timestamp.Format("15:04:05"),
		"template_id":   httpLog.TemplateID,
		"template_name": httpLog.TemplateName,
		"severity":      httpLog.Severity,
		"target":        httpLog.Target,
		"method":        httpLog.Method,
		"status_code":   httpLog.StatusCode,
		"is_vuln_found": httpLog.IsVulnFound,
		"duration_ms":   httpLog.Duration,
	})
}

// Start begins the scanning process
func (sns *SimpleNucleiScanner) Start() error {
	startTime := time.Now()

	// Log scan start
	if sns.logger != nil {
		sns.logger.Info("Starting nuclei scan", map[string]interface{}{
			"task_id":      sns.task.ID,
			"task_name":    sns.task.Name,
			"poc_count":    len(sns.task.POCs),
			"target_count": len(sns.task.Targets),
			"timeout":      sns.timeout.String(),
		})
	}

	// Update progress to running
	sns.updateProgress(0, 0, "running")

	// Create output directory with absolute path
	// Use the manager's results directory as base
	baseDir := sns.manager.resultsDir
	outputDir := filepath.Join(baseDir, fmt.Sprintf("task_%d", sns.task.ID))
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		if sns.logger != nil {
			sns.logger.Error("Failed to create output directory", err, map[string]interface{}{
				"task_id":    sns.task.ID,
				"output_dir": outputDir,
			})
		}
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	outputFile := filepath.Join(outputDir, "nuclei_output.jsonl")

	// Log output file preparation
	if sns.logger != nil {
		sns.logger.Debug("Output file prepared", map[string]interface{}{
			"task_id":     sns.task.ID,
			"output_file": outputFile,
			"output_dir":  outputDir,
		})
	}

	// Prepare output file
	if err := sns.prepareOutputFile(outputFile); err != nil {
		if sns.logger != nil {
			sns.logger.Error("Failed to prepare output file", err, map[string]interface{}{
				"task_id":     sns.task.ID,
				"output_file": outputFile,
			})
		}
		return fmt.Errorf("failed to prepare output file: %v", err)
	}

	// Create targets file
	targetsFile, err := sns.createTargetsFile()
	if err != nil {
		if sns.logger != nil {
			sns.logger.Error("Failed to create targets file", err, map[string]interface{}{
				"task_id": sns.task.ID,
			})
		}
		return fmt.Errorf("failed to create targets file: %v", err)
	}
	defer os.Remove(targetsFile)

	// Log targets file creation
	if sns.logger != nil {
		sns.logger.Debug("Targets file created", map[string]interface{}{
			"task_id":      sns.task.ID,
			"targets_file": targetsFile,
			"target_count": len(sns.task.Targets),
		})
	}

	// Build nuclei command
	cmd := sns.buildNucleiCommand(targetsFile, outputFile)

	// Log command construction
	if sns.logger != nil {
		cmdInfo := &CommandInfo{
			Executable:  cmd.Path,
			Arguments:   cmd.Args[1:], // Skip the executable name
			WorkingDir:  cmd.Dir,
			Environment: make(map[string]string),
		}

		// Capture environment variables
		for _, env := range cmd.Env {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				cmdInfo.Environment[parts[0]] = parts[1]
			}
		}

		sns.logger.LogCommand(cmdInfo, "Nuclei command constructed", map[string]interface{}{
			"task_id":        sns.task.ID,
			"command_length": len(strings.Join(cmd.Args, " ")),
		})
	}

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		if sns.logger != nil {
			sns.logger.Error("Failed to create stdout pipe", err, map[string]interface{}{
				"task_id": sns.task.ID,
			})
		}
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		if sns.logger != nil {
			sns.logger.Error("Failed to create stderr pipe", err, map[string]interface{}{
				"task_id": sns.task.ID,
			})
		}
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		if sns.logger != nil {
			sns.logger.Error("Failed to start nuclei command", err, map[string]interface{}{
				"task_id": sns.task.ID,
				"command": strings.Join(cmd.Args, " "),
			})
		}
		return fmt.Errorf("failed to start nuclei command: %v", err)
	}

	// Log command start
	if sns.logger != nil {
		sns.logger.Info("Nuclei command started", map[string]interface{}{
			"task_id":    sns.task.ID,
			"process_id": cmd.Process.Pid,
		})
	}

	// Monitor stdout and stderr only
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		sns.monitorStdout(bufio.NewScanner(stdout))
	}()

	go func() {
		defer wg.Done()
		sns.monitorStderr(bufio.NewScanner(stderr))
	}()

	// Wait for command to complete or timeout
	done := make(chan error, 1)
	go func() {
		wg.Wait()
		done <- cmd.Wait()
	}()

	var cmdErr error
	select {
	case cmdErr = <-done:
		// Command completed
	case <-time.After(sns.timeout):
		// Timeout occurred
		if sns.logger != nil {
			sns.logger.Warn("Nuclei command timed out", map[string]interface{}{
				"task_id": sns.task.ID,
				"timeout": sns.timeout.String(),
			})
		}
		if err := cmd.Process.Kill(); err != nil {
			if sns.logger != nil {
				sns.logger.Error("Failed to kill timed out process", err, map[string]interface{}{
					"task_id":    sns.task.ID,
					"process_id": cmd.Process.Pid,
				})
			}
		}
		cmdErr = fmt.Errorf("nuclei command timed out after %v", sns.timeout)
	}

	executionDuration := time.Since(startTime)

	// Log command completion
	if sns.logger != nil {
		exitReason := "completed"
		if cmdErr != nil {
			exitReason = "error"
		}

		cmdInfo := &CommandInfo{
			Executable: cmd.Path,
			Arguments:  cmd.Args[1:],
			WorkingDir: cmd.Dir,
			Duration:   executionDuration,
		}

		if cmd.ProcessState != nil {
			cmdInfo.ExitCode = cmd.ProcessState.ExitCode()
		}

		// Get output file size
		if stat, err := os.Stat(outputFile); err == nil {
			cmdInfo.OutputSize = stat.Size()
		}

		contextData := map[string]interface{}{
			"task_id":            sns.task.ID,
			"exit_reason":        exitReason,
			"execution_duration": executionDuration.String(),
			"process_id":         cmd.Process.Pid,
			"output_file_size":   cmdInfo.OutputSize,
		}

		if cmdErr != nil {
			sns.logger.Error("Nuclei command execution completed with error", cmdErr, contextData)
		} else {
			sns.logger.Info("Nuclei command execution completed successfully", contextData)
		}

		sns.logger.LogCommand(cmdInfo, fmt.Sprintf("Nuclei command %s", exitReason), contextData)
	}

	// Process results even if there was an error
	if err := sns.processResults(outputFile); err != nil {
		if sns.logger != nil {
			sns.logger.Error("Failed to process scan results", err, map[string]interface{}{
				"task_id":     sns.task.ID,
				"output_file": outputFile,
			})
		}
		sns.addLog("ERROR", "", "", fmt.Sprintf("Failed to process results: %v", err), "", "", false)
	}

	// Clean up temporary directory if it exists
	if sns.tempDir != "" {
		if err := os.RemoveAll(sns.tempDir); err != nil {
			if sns.logger != nil {
				sns.logger.Warn("Failed to clean up temporary directory", map[string]interface{}{
					"task_id":  sns.task.ID,
					"temp_dir": sns.tempDir,
					"error":    err.Error(),
				})
			}
		} else if sns.logger != nil {
			sns.logger.Debug("Temporary directory cleaned up", map[string]interface{}{
				"task_id":  sns.task.ID,
				"temp_dir": sns.tempDir,
			})
		}
	}

	// Save logs
	if err := sns.saveLogs(); err != nil {
		if sns.logger != nil {
			sns.logger.Error("Failed to save scan logs", err, map[string]interface{}{
				"task_id": sns.task.ID,
			})
		}
	}

	// 扫描结束后的最终统计
	sns.progressMu.Lock()
	
	// 计算被跳过的模板数量
	// 跳过的模板 = 总模板 - 被过滤的模板 - 实际扫描的模板
	actualScanned := len(sns.templateSet)
	filteredCount := sns.progress.FilteredTemplates
	skippedCount := sns.progress.TotalTemplates - filteredCount - actualScanned
	
	if skippedCount > 0 {
		sns.progress.SkippedTemplates = skippedCount
		// 为跳过的模板生成ID列表（用于调试）
		scannedSet := make(map[string]bool)
		for templateID := range sns.templateSet {
			scannedSet[templateID] = true
		}
		
		// 找出被跳过的模板ID
		for _, templateID := range sns.progress.SelectedTemplates {
			if !scannedSet[templateID] {
				sns.progress.SkippedTemplateIDs = append(sns.progress.SkippedTemplateIDs, templateID)
			}
		}
		
		fmt.Printf("📋 最终统计: 总计%d个POC，过滤%d个，跳过%d个，实际扫描%d个\n", 
			sns.progress.TotalTemplates, filteredCount, skippedCount, actualScanned)
	}
	
	sns.progress.ScannedTemplates = actualScanned
	sns.progress.CompletedTemplates = actualScanned
	sns.progressMu.Unlock()

	// Update final progress - 打印统计信息
	fmt.Printf("\n✅ 扫描完成！统计信息：\n")
	fmt.Printf("   - 已扫描POC: %d/%d\n", actualScanned, sns.progress.TotalTemplates)

	sns.failedTemplatesMu.Lock()
	actualFailed := len(sns.failedTemplates)
	sns.failedTemplatesMu.Unlock()

	fmt.Printf("   - 失败POC: %d\n", actualFailed)
	fmt.Printf("   - 发现漏洞: %d\n", sns.progress.FoundVulns)
	fmt.Printf("   - 完成请求: %d/%d\n", sns.progress.CompletedRequests, sns.progress.TotalRequests)
	fmt.Printf("   - 设置状态: completed\n\n")

	// 发送最终的completed状态事件（只发送一次）
	fmt.Printf("🎯 发送最终completed状态事件到前端\n")
	sns.updateProgress(sns.progress.CompletedRequests, sns.progress.FoundVulns, "completed")

	// 等待确保事件被发送和处理
	time.Sleep(200 * time.Millisecond)

	// Log scan completion
	if sns.logger != nil {
		sns.logger.Info("Nuclei scan completed", map[string]interface{}{
			"task_id":               sns.task.ID,
			"total_duration":        executionDuration.String(),
			"vulnerabilities_found": sns.progress.FoundVulns,
			"requests_completed":    sns.progress.CompletedRequests,
			"scanned_templates":     sns.progress.ScannedTemplates,
			"total_templates":       sns.progress.TotalTemplates,
			"final_status":          sns.progress.Status,
		})
	}

	return cmdErr
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
	
	// 用于跟踪所有模板的扫描状态
	allTemplatesSet := make(map[string]bool) // 所有遇到的模板（成功+失败）

	// 辅助函数：统一处理模板计数
	updateTemplateCount := func(templateID string, reason string) {
		sns.templateSetMu.Lock()
		defer sns.templateSetMu.Unlock()
		
		// 只有当模板第一次遇到时才更新计数
		if !allTemplatesSet[templateID] {
			allTemplatesSet[templateID] = true
			sns.progressMu.Lock()
			sns.progress.ScannedTemplates = len(allTemplatesSet)
			// 记录已扫描模板ID
			sns.progress.ScannedTemplateIDs = append(sns.progress.ScannedTemplateIDs, templateID)
			// 更新当前序号（若可解析索引）
			if idx, ok := sns.templateIndex[templateID]; ok {
				sns.progress.CurrentIndex = idx + 1 // 1-based
			} else {
				// 回退为已扫描数量
				sns.progress.CurrentIndex = sns.progress.ScannedTemplates
			}
			sns.progressMu.Unlock()
			
			fmt.Printf("📋 POC扫描进度: %d/%d 个POC已扫描 - %s (%s)\n", 
				sns.progress.ScannedTemplates, sns.progress.TotalTemplates, templateID, reason)
		}
	}

	for stdout.Scan() {
		rawLine := stdout.Text()
		line := stripAnsiCodes(rawLine) // Remove ANSI color codes

		// Log nuclei output to debug file
		sns.logNucleiOutput(line, false)

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
					
					// 更新当前扫描的模板信息
					sns.progress.CurrentTemplate = templateID
					if host, ok := jsonData["host"].(string); ok {
						sns.progress.CurrentTarget = host
					}
					// 更新当前序号
					if idx, ok := sns.templateIndex[templateID]; ok {
						sns.progress.CurrentIndex = idx + 1
					} else {
						sns.progress.CurrentIndex = sns.progress.ScannedTemplates
					}
					
					// 标记模板为已扫描（成功）- 只更新成功计数，不重复更新总计数
					sns.templateSetMu.Lock()
					if !sns.templateSet[templateID] {
						sns.templateSet[templateID] = true
						sns.progress.CompletedTemplates = len(sns.templateSet)
					}
					sns.templateSetMu.Unlock()
					
					sns.progressMu.Unlock()
					
					// 统一更新模板计数（如果是第一次遇到这个模板）
					updateTemplateCount(templateID, "发现漏洞")

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
							// 缓存模板的严重性信息
							sns.setTemplateSeverity(templateID, severity)
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

	// 检测Nuclei模板加载和过滤信息
	if strings.Contains(line, "Templates loaded for current scan:") {
		// 解析实际加载的模板数量
		re := regexp.MustCompile(`Templates loaded for current scan: (\d+)`)
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			if loadedCount, err := strconv.Atoi(matches[1]); err == nil {
				sns.progressMu.Lock()
				// 计算被过滤的模板数量
				filteredCount := sns.progress.TotalTemplates - loadedCount
				if filteredCount > 0 {
					sns.progress.FilteredTemplates = filteredCount
					fmt.Printf("📋 模板过滤: %d/%d 个POC被Nuclei过滤（不适用当前扫描）\n", 
						filteredCount, sns.progress.TotalTemplates)
				}
				sns.progressMu.Unlock()
				
				// 发送进度更新
				sns.emitEvent("progress", sns.progress)
			}
		}
		continue
	}

	// 检测模板开始扫描的标志
		if strings.Contains(line, "Executing") && strings.Contains(line, "on") {
			// 匹配类似 "[2025-01-24 23:17:14] [CVE-2020-0760] Executing CVE-2020-0760 on http://192.168.1.3:8080"
			re := regexp.MustCompile(`\[([^\]]+)\] Executing ([^\s]+) on (.+)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 3 {
				templateID := matches[2]
				target := matches[3]
				
				// 更新当前扫描的模板和目标信息
				sns.progressMu.Lock()
				sns.progress.CurrentTemplate = templateID
				sns.progress.CurrentTarget = target
				// 更新当前序号
				if idx, ok := sns.templateIndex[templateID]; ok {
					sns.progress.CurrentIndex = idx + 1
				} else {
					sns.progress.CurrentIndex = sns.progress.ScannedTemplates
				}
				sns.progressMu.Unlock()
				
				// 使用统一的计数函数
				updateTemplateCount(templateID, fmt.Sprintf("目标: %s", target))
				
				// 发送进度更新
				sns.emitEvent("progress", sns.progress)
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
				
				// 更新当前扫描的模板和目标信息
				sns.progressMu.Lock()
				sns.progress.CurrentTemplate = currentTemplate
				sns.progress.CurrentTarget = currentTarget
				// 更新当前序号
				if idx, ok := sns.templateIndex[currentTemplate]; ok {
					sns.progress.CurrentIndex = idx + 1
				} else {
					sns.progress.CurrentIndex = sns.progress.ScannedTemplates
				}
				sns.progressMu.Unlock()
				
				// 使用统一的计数函数
				updateTemplateCount(currentTemplate, "")
				
				// 检查是否为成功扫描的模板
				sns.templateSetMu.Lock()
				if !sns.templateSet[currentTemplate] {
					sns.templateSet[currentTemplate] = true
					sns.progressMu.Lock()
					sns.progress.CompletedTemplates = len(sns.templateSet)
					sns.progressMu.Unlock()
				}
				sns.templateSetMu.Unlock()
				
				// 发送进度更新
				sns.emitEvent("progress", sns.progress)
				
				inRequest = true
				currentRequest.Reset()
			}
			continue
		}

		// 检测模板扫描失败或跳过的情况
		if strings.Contains(line, "Could not execute step") || 
		   strings.Contains(line, "template execution failed") ||
		   strings.Contains(line, "skipping template") ||
		   strings.Contains(line, "template not applicable") {
			
			// 尝试从错误信息中提取模板ID
			templateID := ""
			if strings.Contains(line, "[") && strings.Contains(line, "]") {
				re := regexp.MustCompile(`\[([^\]]+)\]`)
				matches := re.FindStringSubmatch(line)
				if len(matches) > 1 {
					templateID = matches[1]
				}
			}
			
			if templateID != "" {
				sns.failedTemplatesMu.Lock()
				if !sns.failedTemplates[templateID] {
					sns.failedTemplates[templateID] = true
					sns.progressMu.Lock()
					sns.progress.FailedTemplates = len(sns.failedTemplates)
					// 记录失败模板ID
					sns.progress.FailedTemplateIDs = append(sns.progress.FailedTemplateIDs, templateID)
					// 更新当前序号
					if idx, ok := sns.templateIndex[templateID]; ok {
						sns.progress.CurrentIndex = idx + 1
					}
					sns.progressMu.Unlock()
					
					fmt.Printf("❌ POC扫描失败: %s - %s\n", templateID, line)
				}
				sns.failedTemplatesMu.Unlock()
				
				// 使用统一的计数函数
				updateTemplateCount(templateID, "失败")
				
				// 发送进度更新
				sns.emitEvent("progress", sns.progress)
			}
			continue
		}

		// 检测模板完成但没有发现漏洞的情况
		// 匹配类似 "[CVE-2020-0760] Finished CVE-2020-0760 execution on http://192.168.1.3:8080"
		// 或者 "[template-id] No match found for template-id on target"
		if (strings.Contains(line, "Finished") && strings.Contains(line, "execution")) ||
		   (strings.Contains(line, "No match") && strings.Contains(line, "found")) ||
		   (strings.Contains(line, "completed") && strings.Contains(line, "template")) {
			
			// 尝试从日志中提取模板ID
			templateID := ""
			if strings.Contains(line, "[") && strings.Contains(line, "]") {
				re := regexp.MustCompile(`\[([^\]]+)\]`)
				matches := re.FindStringSubmatch(line)
				if len(matches) > 1 {
					templateID = matches[1]
				}
			}
			
			if templateID != "" {
				// 更新当前序号
				sns.progressMu.Lock()
				if idx, ok := sns.templateIndex[templateID]; ok {
					sns.progress.CurrentIndex = idx + 1
				}
				sns.progressMu.Unlock()

				// 使用统一的计数函数
				updateTemplateCount(templateID, "无漏洞")
				
				// 发送进度更新
				sns.emitEvent("progress", sns.progress)
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
				// Response ends; emit real-time HTTP request/response event
				if currentRequest.Len() > 0 && currentResponse.Len() > 0 {
					requestStr := currentRequest.String()
					responseStr := currentResponse.String()

					// 从请求/响应中提取真实的template ID（从Nuclei日志标记中提取）
					// 示例：[VER] [CVE-2017-12615] Sent HTTP request...
					realTemplateID := currentTemplate // 默认使用currentTemplate
					realTarget := currentTarget       // 默认使用currentTarget

					// 正则匹配：\[...\] \[template-id\] ...
					re := regexp.MustCompile(`\[(VER|INF|DBG)\] \[([^\]]+)\]`)
					matches := re.FindStringSubmatch(requestStr + responseStr)
					if len(matches) > 2 {
						realTemplateID = matches[2]
					}

					// 从请求中提取target URL（从Nuclei日志中）
					// 示例：Sent HTTP request to http://example.com/path
					targetRe := regexp.MustCompile(`(?:Sent HTTP request to|Dumped HTTP (?:request|response)) (https?://[^\s]+)`)
					targetMatches := targetRe.FindStringSubmatch(requestStr + responseStr)
					if len(targetMatches) > 1 {
						realTarget = targetMatches[1]
					}

					// 解析HTTP方法和状态码
					method := "GET" // 默认
					statusCode := 200 // 默认

					// 从请求中提取方法（第一行：GET /path HTTP/1.1）
					requestLines := strings.Split(requestStr, "\n")
					if len(requestLines) > 0 {
						firstLine := strings.Fields(requestLines[0])
						if len(firstLine) > 0 {
							method = firstLine[0]
						}
					}

					// 从响应中提取状态码（第一行：HTTP/1.1 200 OK）
					responseLines := strings.Split(responseStr, "\n")
					if len(responseLines) > 0 {
						firstLine := strings.Fields(responseLines[0])
						if len(firstLine) >= 2 {
							if code, err := strconv.Atoi(firstLine[1]); err == nil {
								statusCode = code
							}
						}
					}

					// Save to old format logs for later viewing
					sns.addLog("DEBUG", realTemplateID, realTarget,
						fmt.Sprintf("%s -> %s", realTemplateID, realTarget),
						requestStr, responseStr, false)

					// 记录到HTTP请求日志（用于前端表格展示）
					sns.addHTTPRequestLog(
						realTemplateID,                      // template_id (使用真实提取的ID)
						realTemplateID,                      // template_name
						sns.getTemplateSeverity(realTemplateID), // severity (从缓存获取)
						realTarget,                          // target (使用真实提取的URL)
						method,                              // method
						statusCode,                          // status_code
						requestStr,                          // request
						responseStr,                         // response
						false,                               // is_vuln_found
						0,                                   // duration_ms
					)

					// Emit to frontend in real-time (deprecated, 已由 addHTTPRequestLog 发送)
					// sns.emitEvent("http", ...)
				}
				currentRequest.Reset()
				currentResponse.Reset()
			}
		}
	}
}

// monitorStderr monitors the nuclei stderr for POC progress, HTTP requests/responses
func (sns *SimpleNucleiScanner) monitorStderr(stderr *bufio.Scanner) {
	var currentRequest, currentResponse strings.Builder
	var currentTemplate, currentTarget string
	inRequest, inResponse := false, false
	lastLine := ""

	// 用于跟踪实际扫描的POC（从stderr的[template-id]标记）
	scannedPOCs := make(map[string]bool)
	// 匹配所有template-id：[CVE-2020-1234]、[tomcat-default-login]等
	// 允许字母（大小写）、数字、连字符、下划线
	templateIDPattern := regexp.MustCompile(`\[([a-zA-Z][a-zA-Z0-9\-_]+)\]`)

	// 匹配Nuclei过滤信息：[WRN] Excluded X template[s]
	excludedPattern := regexp.MustCompile(`\[WRN\]\s+Excluded\s+(\d+)\s+(\w+)\s+template`)
	totalFiltered := 0

	for stderr.Scan() {
		rawLine := stderr.Text()
		line := stripAnsiCodes(rawLine)

		// Log nuclei stderr to debug file
		sns.logNucleiOutput(line, true)

		// Skip empty lines and duplicates
		if line == "" || line == lastLine {
			continue
		}
		lastLine = line

		// 解析Nuclei过滤信息：Excluded X template[s]
		if matches := excludedPattern.FindStringSubmatch(line); len(matches) >= 3 {
			count := 0
			if n, err := fmt.Sscanf(matches[1], "%d", &count); err == nil && n == 1 {
				templateType := matches[2]
				totalFiltered += count
				fmt.Printf("📝 Nuclei过滤: %d个%s模板\n", count, templateType)

				// 更新进度
				sns.progressMu.Lock()
				sns.progress.FilteredTemplates = totalFiltered
				sns.progressMu.Unlock()
			}
		}

		// 解析POC扫描进度：提取 [template-id] 标记
		// 格式: [INF] [CVE-2020-1234] ... 或 [VER] [CVE-2020-1234] ...
		if strings.Contains(line, "[INF]") || strings.Contains(line, "[VER]") || strings.Contains(line, "[DBG]") {
			matches := templateIDPattern.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				if len(match) > 1 {
					templateID := match[1]
					// 跳过日志级别标记（非POC的标记）
					logLevels := map[string]bool{
						"INF": true, "VER": true, "DBG": true, "WRN": true,
						"ERR": true, "FTL": true, "TRC": true, "SIL": true,
					}
					if logLevels[templateID] {
						continue
					}

					// 检查是否是新的POC
					if !scannedPOCs[templateID] {
						scannedPOCs[templateID] = true
						currentTemplate = templateID

						// 更新进度
						sns.templateSetMu.Lock()
						sns.templateSet[templateID] = true
						scannedCount := len(sns.templateSet)
						sns.templateSetMu.Unlock()

						sns.progressMu.Lock()
						sns.progress.ScannedTemplates = scannedCount
						sns.progress.CompletedTemplates = scannedCount
						sns.progress.CurrentTemplate = templateID
						sns.progress.ScannedTemplateIDs = append(sns.progress.ScannedTemplateIDs, templateID)

						// 更新当前序号
						if idx, ok := sns.templateIndex[templateID]; ok {
							sns.progress.CurrentIndex = idx + 1
						} else {
							sns.progress.CurrentIndex = scannedCount
						}
						sns.progressMu.Unlock()

						fmt.Printf("📋 POC扫描: %d/%d - %s\n",
							scannedCount, sns.progress.TotalTemplates, templateID)

						// 发送进度更新
						sns.emitEvent("progress", sns.progress)
					}
				}
			}
		}

		// 检测HTTP请求开始
		if strings.HasPrefix(line, "GET ") || strings.HasPrefix(line, "POST ") ||
			strings.HasPrefix(line, "PUT ") || strings.HasPrefix(line, "DELETE ") ||
			strings.HasPrefix(line, "PATCH ") || strings.HasPrefix(line, "HEAD ") ||
			strings.HasPrefix(line, "OPTIONS ") {
			// 如果之前有未完成的请求/响应对，先发送
			if currentRequest.Len() > 0 && currentResponse.Len() > 0 {
				sns.emitHTTPEvent(currentTemplate, currentTarget, currentRequest.String(), currentResponse.String())
			}

			inRequest = true
			inResponse = false
			currentRequest.Reset()
			currentResponse.Reset()
			currentRequest.WriteString(line + "\n")

			// 尝试从请求行提取目标URL
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentTarget = parts[1] // 路径部分
			}
			continue
		}

		// 检测HTTP响应开始
		if strings.HasPrefix(line, "HTTP/1.") || strings.HasPrefix(line, "HTTP/2") {
			inRequest = false
			inResponse = true
			currentResponse.Reset()
			currentResponse.WriteString(line + "\n")
			continue
		}

		// 收集请求或响应的内容
		if inRequest {
			currentRequest.WriteString(line + "\n")

			// 请求在空行结束
			if strings.TrimSpace(line) == "" {
				inRequest = false
			}

			// 尝试从Host头提取目标
			if strings.HasPrefix(line, "Host: ") {
				host := strings.TrimPrefix(line, "Host: ")
				host = strings.TrimSpace(host)
				if currentTarget != "" {
					currentTarget = "http://" + host + currentTarget
				}
			}
		} else if inResponse {
			currentResponse.WriteString(line + "\n")

			// 响应在空行且有足够内容后结束
			if strings.TrimSpace(line) == "" && currentResponse.Len() > 50 {
				inResponse = false

				// 发送HTTP事件
				if currentRequest.Len() > 0 && currentResponse.Len() > 0 {
					sns.emitHTTPEvent(currentTemplate, currentTarget, currentRequest.String(), currentResponse.String())
				}

				currentRequest.Reset()
				currentResponse.Reset()
			}
		}
	}

	// 处理最后可能剩余的请求/响应对
	if currentRequest.Len() > 0 && currentResponse.Len() > 0 {
		sns.emitHTTPEvent(currentTemplate, currentTarget, currentRequest.String(), currentResponse.String())
	}

	// 最终统计
	sns.progressMu.Lock()
	actualScanned := len(scannedPOCs)
	sns.progress.ScannedTemplates = actualScanned
	sns.progress.CompletedTemplates = actualScanned

	// 计算被跳过的POC数量
	// 跳过数 = 总数 - 已扫描 - 被过滤
	sns.progress.SkippedTemplates = sns.progress.TotalTemplates - actualScanned - sns.progress.FilteredTemplates
	if sns.progress.SkippedTemplates < 0 {
		sns.progress.SkippedTemplates = 0
	}
	sns.progressMu.Unlock()

	fmt.Printf("\n✅ Stderr监控结束\n")
	fmt.Printf("   - 总POC数: %d\n", sns.progress.TotalTemplates)
	fmt.Printf("   - 已扫描: %d\n", actualScanned)
	fmt.Printf("   - 被过滤: %d (Nuclei安全机制)\n", sns.progress.FilteredTemplates)
	fmt.Printf("   - 被跳过: %d (条件不符)\n", sns.progress.SkippedTemplates)
	fmt.Printf("   - 统计: %d + %d + %d = %d\n\n",
		actualScanned, sns.progress.FilteredTemplates, sns.progress.SkippedTemplates, sns.progress.TotalTemplates)
}

// emitHTTPEvent 发送HTTP请求/响应事件到前端
func (sns *SimpleNucleiScanner) emitHTTPEvent(templateID, target, request, response string) {
	if request == "" || response == "" {
		return
	}

	// 更新进度中的当前模板和目标
	sns.progressMu.Lock()
	if templateID == "" {
		templateID = sns.progress.CurrentTemplate
	}
	if target != "" {
		sns.progress.CurrentTarget = target
	}
	if templateID != "" {
		sns.progress.CurrentTemplate = templateID
	}
	sns.progressMu.Unlock()

	// 解析HTTP方法和状态码
	method := "GET"
	statusCode := 200

	// 从请求中提取方法
	requestLines := strings.Split(request, "\n")
	if len(requestLines) > 0 {
		firstLine := strings.Fields(requestLines[0])
		if len(firstLine) > 0 {
			method = firstLine[0]
		}
	}

	// 从响应中提取状态码
	responseLines := strings.Split(response, "\n")
	if len(responseLines) > 0 {
		firstLine := strings.Fields(responseLines[0])
		if len(firstLine) >= 2 {
			if code, err := strconv.Atoi(firstLine[1]); err == nil {
				statusCode = code
			}
		}
	}

	// 记录到HTTP请求日志
	sns.addHTTPRequestLog(
		templateID,
		templateID,
		sns.getTemplateSeverity(templateID), // 从缓存获取severity
		target,
		method,
		statusCode,
		request,
		response,
		false,
		0,
	)

	// 发送HTTP事件
	sns.emitEvent("http", map[string]interface{}{
		"template_id": templateID,
		"target":      target,
		"request":     request,
		"response":    response,
		"timestamp":   time.Now().Format("15:04:05"),
	})

	fmt.Printf("📨 HTTP请求/响应: %s -> %s\n", templateID, target)
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
		"-l", targetsFile, // Target list file
		"-jle", outputFile, // JSONL export to file (matches user's spec)
		"-jsonl",               // Also output JSONL to stdout for real-time parsing
		"-include-rr",          // Include request/response in outputs
		"-stats",               // Show statistics
		"-stats-interval", "2", // Stats interval (as specified)
		"-debug",         // Debug mode to get request/response (KEY FEATURE)
		"-timeout", "30", // HTTP timeout per request (30 seconds) - CRITICAL for preventing hangs
		"-retries", "1", // Retry failed requests once
		"-nc", // No color output
		"-v",  // Verbose
	}

	// 添加 DNS 外带 (Interactsh) 配置
	if sns.manager != nil && sns.manager.config != nil {
		nucleiConfig := sns.manager.config.NucleiConfig

		// 如果完全禁用 Interactsh
		if nucleiConfig.InteractshDisable {
			args = append(args, "-no-interactsh")
			fmt.Printf("🔧 DNS外带功能已禁用: -no-interactsh\n")
		} else if nucleiConfig.InteractshEnabled {
			// 启用 Interactsh 并配置自定义服务器
			if nucleiConfig.InteractshServer != "" {
				args = append(args, "-interactsh-server", nucleiConfig.InteractshServer)
				fmt.Printf("🔧 使用自定义Interactsh服务器: %s\n", nucleiConfig.InteractshServer)
			}

			// 添加 Interactsh Token（如果有）
			if nucleiConfig.InteractshToken != "" {
				args = append(args, "-interactsh-token", nucleiConfig.InteractshToken)
				fmt.Printf("🔧 使用Interactsh认证Token\n")
			}
		}
	}

	// Use temporary directory approach to avoid Windows command line length limits
	if len(sns.task.POCs) > 100 { // Use temp directory for large template sets
		tempManager, err := NewTempManager()
		if err != nil {
			fmt.Printf("⚠️  创建临时目录管理器失败，回退到单个模板模式: %v\n", err)
			// Fallback to individual templates
			sns.addIndividualTemplates(&args)
		} else {
			// Create temporary directory with selected templates
			tempDir, err := tempManager.CreateTempTemplateDir(sns.task.ID, sns.task.POCs)
			if err != nil {
				fmt.Printf("⚠️  创建临时模板目录失败，回退到单个模板模式: %v\n", err)
				// Fallback to individual templates
				sns.addIndividualTemplates(&args)
			} else {
				// Use directory parameter instead of individual -t parameters
				args = append(args, "-t", tempDir)
				fmt.Printf("🚀 使用临时目录模式: %s (包含 %d 个模板)\n", tempDir, len(sns.task.POCs))

				// Store temp directory for cleanup
				sns.tempDir = tempDir
			}
		}
	} else {
		// Use individual templates for smaller sets (< 100 templates)
		sns.addIndividualTemplates(&args)
	}

	// Log the command being executed for debugging
	fmt.Printf("🔧 执行命令: %s %v\n", sns.nucleiPath, args)

	// Save debug info to log file
	sns.logDebugInfo(sns.nucleiPath, args, outputFile)

	cmd := exec.Command(sns.nucleiPath, args...)

	// Set working directory to the project root or a safe directory
	// Don't use the output file directory as working directory
	if workDir, err := os.Getwd(); err == nil {
		cmd.Dir = workDir
	} else {
		// Fallback to home directory if current directory is not accessible
		if homeDir, err := os.UserHomeDir(); err == nil {
			cmd.Dir = homeDir
		}
	}

	// Set environment variables for Windows
	if runtime.GOOS == "windows" {
		// Add nuclei directory to PATH
		nucleiDir := filepath.Dir(sns.nucleiPath)
		currentPath := os.Getenv("PATH")
		newPath := nucleiDir + ";" + currentPath
		cmd.Env = append(os.Environ(), "PATH="+newPath)

		// Hide the command window on Windows
		hideWindowOnWindows(cmd)
	}

	return cmd
}

// addIndividualTemplates adds individual template files to the command arguments
func (sns *SimpleNucleiScanner) addIndividualTemplates(args *[]string) {
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
		*args = append(*args, "-t", templateFile)
		fmt.Printf("  📄 %s\n", templateFile)
	}
	fmt.Printf("模板数量: %d\n", len(sns.task.POCs))
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

	// Store log file path for later use
	sns.debugLogFile = logFile

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

// logNucleiOutput logs nuclei stdout/stderr to debug file
func (sns *SimpleNucleiScanner) logNucleiOutput(line string, isStderr bool) {
	if sns.debugLogFile == "" {
		return
	}

	file, err := os.OpenFile(sns.debugLogFile, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	prefix := "[STDOUT]"
	if isStderr {
		prefix = "[STDERR]"
	}
	
	fmt.Fprintf(file, "%s %s %s\n", time.Now().Format("15:04:05"), prefix, line)
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

	// 获取实际的统计数据
	sns.progressMu.RLock()
	actualTotalRequests := sns.progress.TotalRequests
	actualCompletedRequests := sns.progress.CompletedRequests
	scannedTemplates := sns.progress.ScannedTemplates
	filteredTemplates := sns.progress.FilteredTemplates
	skippedTemplates := sns.progress.SkippedTemplates
	failedTemplates := sns.progress.FailedTemplates
	filteredTemplateIDs := append([]string{}, sns.progress.FilteredTemplateIDs...)
	skippedTemplateIDs := append([]string{}, sns.progress.SkippedTemplateIDs...)
	failedTemplateIDs := append([]string{}, sns.progress.FailedTemplateIDs...)
	scannedTemplateIDs := append([]string{}, sns.progress.ScannedTemplateIDs...)
	sns.progressMu.RUnlock()

	// 计算成功率
	successRate := 100.0
	if actualTotalRequests > 0 {
		successRate = float64(actualCompletedRequests) / float64(actualTotalRequests) * 100
	}

	// Create result object with actual statistics
	result := &TaskResult{
		TaskID:            sns.task.ID,
		TaskName:          sns.task.Name,
		Status:            "completed",
		StartTime:         sns.task.StartTime,
		EndTime:           time.Now(),
		Duration:          time.Since(sns.task.StartTime).String(),
		Targets:           sns.task.Targets,
		Templates:         sns.task.POCs,
		TemplateCount:     len(sns.task.POCs),
		TargetCount:       len(sns.task.Targets),
		TotalRequests:     actualTotalRequests,          // 使用实际值
		CompletedRequests: actualCompletedRequests,      // 使用实际值
		FoundVulns:        len(vulnerabilities),
		SuccessRate:       successRate,                  // 使用计算的成功率
		Vulnerabilities:   vulnerabilities,

		// 详细统计信息
		ScannedTemplates:    scannedTemplates,
		FilteredTemplates:   filteredTemplates,
		SkippedTemplates:    skippedTemplates,
		FailedTemplates:     failedTemplates,
		FilteredTemplateIDs: filteredTemplateIDs,
		SkippedTemplateIDs:  skippedTemplateIDs,
		FailedTemplateIDs:   failedTemplateIDs,
		ScannedTemplateIDs:  scannedTemplateIDs,
		HTTPRequests:        actualCompletedRequests,    // HTTP请求数等于完成的请求数

		Summary: map[string]interface{}{
			"total_requests":      actualTotalRequests,
			"completed_requests":  actualCompletedRequests,
			"found_vulns":         len(vulnerabilities),
			"duration":            time.Since(sns.task.StartTime).String(),
			"success_rate":        successRate,
			"scanned_templates":   scannedTemplates,
			"filtered_templates":  filteredTemplates,
			"skipped_templates":   skippedTemplates,
			"failed_templates":    failedTemplates,
			"http_requests":       actualCompletedRequests,
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

	// 保存HTTP请求日志
	sns.httpLogsMu.Lock()
	httpLogs := append([]*HTTPRequestLog{}, sns.httpRequestLogs...)
	sns.httpLogsMu.Unlock()

	if len(httpLogs) > 0 {
		fmt.Printf("💾 保存HTTP请求日志 (%d 条记录)...\n", len(httpLogs))
		if err := sns.manager.SaveHTTPRequestLogs(sns.task.ID, httpLogs); err != nil {
			fmt.Printf("⚠️ 保存HTTP请求日志失败: %v\n", err)
			// 不返回错误，因为主结果已保存成功
		}
	}

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

	// 获取实际的统计数据
	sns.progressMu.RLock()
	actualTotalRequests := sns.progress.TotalRequests
	actualCompletedRequests := sns.progress.CompletedRequests
	scannedTemplates := sns.progress.ScannedTemplates
	filteredTemplates := sns.progress.FilteredTemplates
	skippedTemplates := sns.progress.SkippedTemplates
	failedTemplates := sns.progress.FailedTemplates
	filteredTemplateIDs := append([]string{}, sns.progress.FilteredTemplateIDs...)
	skippedTemplateIDs := append([]string{}, sns.progress.SkippedTemplateIDs...)
	failedTemplateIDs := append([]string{}, sns.progress.FailedTemplateIDs...)
	scannedTemplateIDs := append([]string{}, sns.progress.ScannedTemplateIDs...)
	sns.progressMu.RUnlock()

	// 计算成功率
	successRate := 100.0
	if actualTotalRequests > 0 {
		successRate = float64(actualCompletedRequests) / float64(actualTotalRequests) * 100
	}

	result := &TaskResult{
		TaskID:            sns.task.ID,
		TaskName:          sns.task.Name,
		Status:            "completed",
		StartTime:         sns.task.StartTime,
		EndTime:           time.Now(),
		Duration:          time.Since(sns.task.StartTime).String(),
		Targets:           sns.task.Targets,
		Templates:         sns.task.POCs,
		TemplateCount:     len(sns.task.POCs),
		TargetCount:       len(sns.task.Targets),
		TotalRequests:     actualTotalRequests,
		CompletedRequests: actualCompletedRequests,
		FoundVulns:        0,
		SuccessRate:       successRate,
		Vulnerabilities:   []*models.NucleiResult{},

		// 详细统计信息
		ScannedTemplates:    scannedTemplates,
		FilteredTemplates:   filteredTemplates,
		SkippedTemplates:    skippedTemplates,
		FailedTemplates:     failedTemplates,
		FilteredTemplateIDs: filteredTemplateIDs,
		SkippedTemplateIDs:  skippedTemplateIDs,
		FailedTemplateIDs:   failedTemplateIDs,
		ScannedTemplateIDs:  scannedTemplateIDs,
		HTTPRequests:        actualCompletedRequests,

		Summary: map[string]interface{}{
			"total_requests":      actualTotalRequests,
			"completed_requests":  actualCompletedRequests,
			"found_vulns":         0,
			"duration":            time.Since(sns.task.StartTime).String(),
			"success_rate":        successRate,
			"scanned_templates":   scannedTemplates,
			"filtered_templates":  filteredTemplates,
			"skipped_templates":   skippedTemplates,
			"failed_templates":    failedTemplates,
			"http_requests":       actualCompletedRequests,
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
