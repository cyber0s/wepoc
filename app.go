package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"wepoc/internal/config"
	"wepoc/internal/database"
	"wepoc/internal/models"
	"wepoc/internal/scanner"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
	db  *database.Database
	taskManager *scanner.TaskManager
	jsonTaskManager *scanner.JSONTaskManager
	config *models.Config
	templateParser *scanner.TemplateParser
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		runtime.LogErrorf(ctx, "Failed to load config: %v", err)
		return
	}
	a.config = cfg

	// Validate and fix nuclei path if needed
	if err := config.ValidateNucleiPath(cfg); err != nil {
		runtime.LogErrorf(ctx, "Nuclei path validation failed: %v", err)
		// Continue anyway, but log the issue
	}

	// Ensure directories exist
	if err := config.EnsureDirectories(cfg); err != nil {
		runtime.LogErrorf(ctx, "Failed to create directories: %v", err)
		return
	}

	// Initialize database
	db, err := database.NewDatabase(cfg.DatabasePath)
	if err != nil {
		runtime.LogErrorf(ctx, "Failed to initialize database: %v", err)
		return
	}
	a.db = db

	// Initialize task manager (legacy database-based)
	a.taskManager = scanner.NewTaskManager(db, cfg.MaxConcurrency, cfg)

	// Initialize JSON task manager (new lightweight approach)
	jsonTaskManager, err := scanner.NewJSONTaskManager(cfg)
	if err != nil {
		runtime.LogErrorf(ctx, "Failed to initialize JSON task manager: %v", err)
		return
	}
	a.jsonTaskManager = jsonTaskManager

	// Initialize template parser
	a.templateParser = scanner.NewTemplateParser()

	// Start event listener for task updates (legacy)
	go a.listenForTaskEvents()

	runtime.LogInfo(ctx, "Application started successfully")
}

// shutdown is called at application termination
func (a *App) shutdown(ctx context.Context) {
	if a.db != nil {
		a.db.Close()
	}
}

// listenForTaskEvents listens for task events and emits them to frontend
func (a *App) listenForTaskEvents() {
	for event := range a.taskManager.GetEventChannel() {
		runtime.EventsEmit(a.ctx, "task-event", event)
	}
}

// ============ Configuration Methods ============

// GetConfig returns the current configuration
func (a *App) GetConfig() *models.Config {
	if a.config == nil {
		// Return default config if not initialized
		cfg, _ := config.GetDefaultConfig()
		return cfg
	}
	return a.config
}

// SaveConfig saves the configuration
func (a *App) SaveConfig(cfg *models.Config) error {
	if err := config.SaveConfig(cfg); err != nil {
		return err
	}
	a.config = cfg
	
	// Update task managers with new configuration
	if a.taskManager != nil {
		a.taskManager.UpdateConfig(cfg)
	}
	if a.jsonTaskManager != nil {
		a.jsonTaskManager.UpdateConfig(cfg)
	}
	
	return nil
}

// ============ Template Management Methods ============

// PreValidateTemplates validates templates without importing them
func (a *App) PreValidateTemplates(dirPath string) (*scanner.ImportResult, error) {
	if a.templateParser == nil {
		return nil, fmt.Errorf("application not initialized properly")
	}

	nucleiPath := a.config.NucleiPath

	// Pre-validate templates
	result, err := a.templateParser.PreValidateTemplates(dirPath, nucleiPath)
	if err != nil {
		return nil, fmt.Errorf("failed to pre-validate templates: %w", err)
	}

	return result, nil
}

// ConfirmAndImportTemplates imports only the pre-validated templates with progress updates
func (a *App) ConfirmAndImportTemplates(validTemplates []*models.Template) (*scanner.ImportResult, error) {
	if a.db == nil || a.templateParser == nil {
		return nil, fmt.Errorf("application not initialized properly")
	}

	// Get target directory from config
	targetDir := a.config.POCDirectory

	// Create progress callback with real-time stats
	var currentStats = struct {
		successful int
		errors     int
		duplicates int
	}{}
	
	progressCallback := func(current, total int, status string, stats ...map[string]int) {
		percentage := float64(current) / float64(total) * 100
		
		// Update stats if provided
		if len(stats) > 0 {
			if val, ok := stats[0]["successful"]; ok {
				currentStats.successful = val
			}
			if val, ok := stats[0]["errors"]; ok {
				currentStats.errors = val
			}
			if val, ok := stats[0]["duplicates"]; ok {
				currentStats.duplicates = val
			}
		}
		
		event := map[string]interface{}{
			"type": "template_import_progress",
			"data": map[string]interface{}{
				"current":     current,
				"total":       total,
				"percentage":  percentage,
				"status":      status,
				"totalFound":  total,
				"imported":    current,
				"successful":  currentStats.successful,
				"errors":      currentStats.errors,
				"duplicates":  currentStats.duplicates,
			},
		}
		runtime.EventsEmit(a.ctx, "template-import-progress", event)
	}

	// Import pre-validated templates with progress
	result, err := a.templateParser.ConfirmAndImportTemplatesWithProgress(validTemplates, targetDir, progressCallback)
	if err != nil {
		return nil, fmt.Errorf("failed to import templates: %w", err)
	}

	// Insert templates into database
	if len(result.ValidTemplates) > 0 {
		if err := a.db.BatchInsertTemplates(result.ValidTemplates); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to save templates to database: %v", err))
		}
	}

	// Send completion event
	completionEvent := map[string]interface{}{
		"type": "template_import_complete",
		"data": map[string]interface{}{
			"totalFound":  result.TotalFound,
			"imported":    result.Validated,
			"successful":  result.Validated,
			"errors":      result.Failed,
			"duplicates":  result.AlreadyExists,
			"percentage":  100.0,
			"status":      "导入完成!",
		},
	}
	runtime.EventsEmit(a.ctx, "template-import-progress", completionEvent)

	return result, nil
}

// DeleteTemplate deletes a template file from the filesystem
func (a *App) DeleteTemplate(templateID string) error {
	if a.db == nil {
		return fmt.Errorf("application not initialized properly")
	}

	// Get template from database by template_id
	template, err := a.db.GetTemplateByTemplateID(templateID)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	if template == nil {
		return fmt.Errorf("template not found")
	}

	// Delete file from filesystem
	if err := os.Remove(template.FilePath); err != nil {
		return fmt.Errorf("failed to delete template file: %w", err)
	}

	// Delete from database
	if err := a.db.DeleteTemplate(template.ID); err != nil {
		return fmt.Errorf("failed to delete template from database: %w", err)
	}

	return nil
}

// ImportTemplates imports templates from a directory with validation and progress updates
func (a *App) ImportTemplates(dirPath string) (*scanner.ImportResult, error) {
	if a.db == nil || a.templateParser == nil {
		return nil, fmt.Errorf("application not initialized properly")
	}

	// Get target directory from config
	targetDir := a.config.POCDirectory
	nucleiPath := a.config.NucleiPath

	// First, scan the directory to get total count
	templates, _ := a.templateParser.ScanDirectory(dirPath)
	totalTemplates := len(templates)

	// Create progress callback with real-time stats
	var currentStats = struct {
		successful int
		errors     int
		duplicates int
	}{}
	
	progressCallback := func(current, total int, status string, stats ...map[string]int) {
		percentage := float64(current) / float64(total) * 100
		
		// Update stats if provided
		if len(stats) > 0 {
			if val, ok := stats[0]["successful"]; ok {
				currentStats.successful = val
			}
			if val, ok := stats[0]["errors"]; ok {
				currentStats.errors = val
			}
			if val, ok := stats[0]["duplicates"]; ok {
				currentStats.duplicates = val
			}
		}
		
		event := map[string]interface{}{
			"type": "template_import_progress",
			"data": map[string]interface{}{
				"current":     current,
				"total":       total,
				"percentage":  percentage,
				"status":      status,
				"totalFound":  total,
				"imported":    current,
				"successful":  currentStats.successful,
				"errors":      currentStats.errors,
				"duplicates":  currentStats.duplicates,
			},
		}
		runtime.EventsEmit(a.ctx, "template-import-progress", event)
	}

	// Send initial progress
	progressCallback(0, totalTemplates, "开始扫描模板...")

	// Import templates with validation
	result, err := a.templateParser.ImportTemplatesWithValidationAndProgress(dirPath, targetDir, nucleiPath, progressCallback)
	if err != nil {
		return nil, fmt.Errorf("failed to import templates: %w", err)
	}

	// Load templates from target directory and insert into database
	templates, _ = a.templateParser.ScanDirectory(targetDir)
	if len(templates) > 0 {
		if err := a.db.BatchInsertTemplates(templates); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to save templates to database: %v", err))
		}
	}

	// Send completion event
	completionEvent := map[string]interface{}{
		"type": "template_import_complete",
		"data": map[string]interface{}{
			"totalFound":  result.TotalFound,
			"imported":    result.Validated,
			"successful":  result.Validated,
			"errors":      result.Failed,
			"duplicates":  result.AlreadyExists,
			"percentage":  100.0,
			"status":      "导入完成!",
		},
	}
	runtime.EventsEmit(a.ctx, "template-import-progress", completionEvent)

	return result, nil
}

// GetAllTemplates returns all templates from database
func (a *App) GetAllTemplates() ([]*models.Template, error) {
	if a.db == nil {
		return []*models.Template{}, nil
	}
	templates, err := a.db.GetAllTemplates()
	if err != nil {
		runtime.LogErrorf(a.ctx, "Failed to get templates: %v", err)
		return []*models.Template{}, nil
	}
	if templates == nil {
		return []*models.Template{}, nil
	}
	return templates, nil
}

// SearchTemplates searches templates by keyword and severity
func (a *App) SearchTemplates(keyword string, severity string) ([]*models.Template, error) {
	return a.db.SearchTemplates(keyword, severity)
}


// ClearAllTemplates removes all templates
func (a *App) ClearAllTemplates() error {
	return a.db.ClearAllTemplates()
}

// SelectDirectory opens a directory selection dialog
func (a *App) SelectDirectory() (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Template Directory",
	})
	return dir, err
}

// SelectNucleiDirectory opens a directory selection dialog and finds nuclei executable
func (a *App) SelectNucleiDirectory() (string, error) {
	// Set default directory based on OS
	defaultDir := "/usr/local/bin"
	if runtime.Environment(a.ctx).Platform == "windows" {
		defaultDir = "C:\\Program Files"
	}
	
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Directory Containing Nuclei Executable",
		DefaultDirectory: defaultDir,
	})
	
	if err != nil {
		runtime.LogErrorf(a.ctx, "Failed to open directory dialog: %v", err)
		return "", fmt.Errorf("打开目录选择对话框失败: %v", err)
	}
	
	if dir == "" {
		return "", fmt.Errorf("未选择目录")
	}
	
	runtime.LogInfof(a.ctx, "Selected directory: %s", dir)
	
	// Look for nuclei executable in the selected directory
	nucleiPath, err := a.findNucleiInDirectory(dir)
	if err != nil {
		runtime.LogErrorf(a.ctx, "Failed to find nuclei in directory %s: %v", dir, err)
		return "", fmt.Errorf("在目录 %s 中未找到 nuclei 可执行文件: %w", dir, err)
	}
	
	runtime.LogInfof(a.ctx, "Found nuclei executable: %s", nucleiPath)
	return nucleiPath, nil
}

// findNucleiInDirectory searches for nuclei executable in a directory
func (a *App) findNucleiInDirectory(dir string) (string, error) {
	// Normalize the directory path
	normalizedDir := filepath.Clean(dir)
	runtime.LogInfof(a.ctx, "Searching for nuclei in directory: %s", normalizedDir)
	
	// Check if directory exists
	if _, err := os.Stat(normalizedDir); os.IsNotExist(err) {
		return "", fmt.Errorf("目录不存在: %s", normalizedDir)
	}
	
	// Read directory contents
	entries, err := os.ReadDir(normalizedDir)
	if err != nil {
		return "", fmt.Errorf("无法读取目录: %v", err)
	}
	
	runtime.LogInfof(a.ctx, "Found %d entries in directory", len(entries))
	
	// Common nuclei executable names (Windows first, then Unix)
	executableNames := []string{"nuclei.exe", "nuclei"}
	
	// First, try exact matches
	for _, name := range executableNames {
		fullPath := filepath.Join(normalizedDir, name)
		runtime.LogInfof(a.ctx, "Checking: %s", fullPath)
		if a.isExecutableFile(fullPath) {
			// Return absolute path
			absPath, err := filepath.Abs(fullPath)
			if err != nil {
				absPath = fullPath
			}
			runtime.LogInfof(a.ctx, "Found nuclei executable: %s", absPath)
			return absPath, nil
		}
	}
	
	// Then, look for files starting with "nuclei_"
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "nuclei_") {
			fullPath := filepath.Join(normalizedDir, entry.Name())
			runtime.LogInfof(a.ctx, "Checking nuclei_ file: %s", fullPath)
			if a.isExecutableFile(fullPath) {
				// Return absolute path
				absPath, err := filepath.Abs(fullPath)
				if err != nil {
					absPath = fullPath
				}
				runtime.LogInfof(a.ctx, "Found nuclei executable: %s", absPath)
				return absPath, nil
			}
		}
	}
	
	// List all files for debugging
	runtime.LogInfof(a.ctx, "Directory contents:")
	for _, entry := range entries {
		runtime.LogInfof(a.ctx, "  %s (dir: %v)", entry.Name(), entry.IsDir())
	}
	
	return "", fmt.Errorf("在目录中未找到 nuclei 可执行文件")
}

// isExecutableFile checks if a file exists and is executable
func (a *App) isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		runtime.LogErrorf(a.ctx, "File stat error for %s: %v", path, err)
		return false
	}
	
	// Check if it's a regular file
	if info.IsDir() {
		return false
	}
	
	// On Windows, check for .exe extension or executable permissions
	if runtime.Environment(a.ctx).Platform == "windows" {
		// On Windows, check for .exe extension or if it has executable permissions
		ext := strings.ToLower(filepath.Ext(path))
		return ext == ".exe" || (info.Mode()&0111) != 0
	}
	
	// On Unix-like systems, check executable permissions
	return (info.Mode()&0111) != 0
}

// ============ Scan Task Methods ============

// CreateScanTask creates a new scanning task (JSON-based)
func (a *App) CreateScanTask(pocsJSON string, targetsJSON string, taskName string) (*scanner.TaskConfig, error) {
	runtime.LogInfof(a.ctx, "=== CreateScanTask called ===")
	runtime.LogInfof(a.ctx, "pocsJSON: %s", pocsJSON)
	runtime.LogInfof(a.ctx, "targetsJSON: %s", targetsJSON)
	runtime.LogInfof(a.ctx, "taskName: %s", taskName)

	var pocs []string
	var targets []string

	if err := json.Unmarshal([]byte(pocsJSON), &pocs); err != nil {
		runtime.LogErrorf(a.ctx, "Failed to unmarshal POCs: %v", err)
		return nil, fmt.Errorf("invalid POCs JSON: %w", err)
	}
	runtime.LogInfof(a.ctx, "Unmarshaled POCs: %v", pocs)

	if err := json.Unmarshal([]byte(targetsJSON), &targets); err != nil {
		runtime.LogErrorf(a.ctx, "Failed to unmarshal targets: %v", err)
		return nil, fmt.Errorf("invalid targets JSON: %w", err)
	}
	runtime.LogInfof(a.ctx, "Unmarshaled targets: %v", targets)

	task, err := a.jsonTaskManager.CreateTask(pocs, targets, taskName)
	if err != nil {
		runtime.LogErrorf(a.ctx, "Failed to create task: %v", err)
		return nil, err
	}
	runtime.LogInfof(a.ctx, "Task created successfully: %+v", task)

	return task, nil
}

// StartScanTask starts a scanning task (JSON-based) with real-time event emission
func (a *App) StartScanTask(taskID int64) error {
	// Register event handler to emit events to frontend
	a.jsonTaskManager.RegisterEventHandler(taskID, func(event *scanner.ScanEvent) {
		// Emit event to frontend via Wails runtime
		runtime.EventsEmit(a.ctx, "scan-event", event)
	})

	return a.jsonTaskManager.StartTask(taskID)
}

// RescanTask restarts a completed or failed task with the same configuration
func (a *App) RescanTask(taskID int64) error {
	// Register event handler to emit events to frontend
	a.jsonTaskManager.RegisterEventHandler(taskID, func(event *scanner.ScanEvent) {
		// Emit event to frontend via Wails runtime
		runtime.EventsEmit(a.ctx, "scan-event", event)
	})

	return a.jsonTaskManager.RescanTask(taskID)
}

// PauseScanTask pauses a running task
func (a *App) PauseScanTask(taskID int64) error {
	return a.taskManager.PauseTask(taskID)
}

// StopScanTask stops a running task
func (a *App) StopScanTask(taskID int64) error {
	return a.taskManager.StopTask(taskID)
}

// GetAllScanTasks returns all scan tasks (JSON-based)
func (a *App) GetAllScanTasks() ([]*scanner.TaskConfig, error) {
	return a.jsonTaskManager.GetAllTasks()
}

// GetRunningScanTasks returns all running tasks (legacy)
func (a *App) GetRunningScanTasks() []*models.ScanTask {
	return a.taskManager.GetRunningTasks()
}

// UpdateScanTask updates an existing scan task (JSON-based)
func (a *App) UpdateScanTask(taskID int64, pocsJSON string, targetsJSON string, taskName string) (*scanner.TaskConfig, error) {
	var pocs []string
	if err := json.Unmarshal([]byte(pocsJSON), &pocs); err != nil {
		return nil, fmt.Errorf("invalid POCs JSON: %v", err)
	}

	var targets []string
	if err := json.Unmarshal([]byte(targetsJSON), &targets); err != nil {
		return nil, fmt.Errorf("invalid targets JSON: %v", err)
	}

	return a.jsonTaskManager.UpdateTask(taskID, pocs, targets, taskName)
}

// DeleteScanTask deletes a scan task (JSON-based)
func (a *App) DeleteScanTask(taskID int64) error {
	return a.jsonTaskManager.DeleteTask(taskID)
}

// GetScanTaskResult returns the scan result for a task (JSON-based)
func (a *App) GetScanTaskResult(taskID int64) (*scanner.TaskResult, error) {
	return a.jsonTaskManager.GetTaskResult(taskID)
}

// GetAllScanResults returns all scan results (JSON-based)
func (a *App) GetAllScanResults() ([]*scanner.TaskResult, error) {
	return a.jsonTaskManager.GetAllTaskResults()
}

// GetTaskLogs returns the logs for a specific task from JSON file
func (a *App) GetTaskLogsFromFile(taskID int64) ([]*scanner.ScanLogEntry, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	logFile := filepath.Join(homeDir, ".wepoc", "logs", fmt.Sprintf("task_%d.json", taskID))

	// Check if log file exists
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return []*scanner.ScanLogEntry{}, nil // Return empty logs if file doesn't exist
	}

	// Read log file
	data, err := os.ReadFile(logFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	// Parse JSON
	var allLogs []*scanner.ScanLogEntry
	if err := json.Unmarshal(data, &allLogs); err != nil {
		return nil, fmt.Errorf("failed to parse log file: %w", err)
	}

	// Filter logs - only return DEBUG (HTTP packets) and VULN entries
	// Skip INFO/WARN/ERROR to reduce data size and improve performance
	var filteredLogs []*scanner.ScanLogEntry
	for _, log := range allLogs {
		if log.Level == "DEBUG" || log.Level == "VULN" {
			filteredLogs = append(filteredLogs, log)
		}
	}

	runtime.LogInfof(a.ctx, "Loaded %d filtered logs (from %d total) for task %d", len(filteredLogs), len(allLogs), taskID)
	return filteredLogs, nil
}

// GetTaskProgress returns the progress of a task
func (a *App) GetTaskProgress(taskID int64) (*models.TaskProgress, error) {
	return a.taskManager.GetTaskStatus(taskID)
}

// GetTaskLogs returns the logs for a specific task
func (a *App) GetTaskLogs(taskID int64) ([]*models.ScanLog, error) {
	return a.taskManager.GetTaskLogs(taskID)
}

// GetTaskLogSummary returns the log summary for a specific task
func (a *App) GetTaskLogSummary(taskID int64) (map[string]interface{}, error) {
	return a.taskManager.GetTaskLogSummary(taskID)
}

// GetScanResult returns the comprehensive scan result for a specific task
func (a *App) GetScanResult(taskID int64) (map[string]interface{}, error) {
	return a.taskManager.GetScanResult(taskID)
}

// DeleteScanResult deletes a scan result file (legacy)
func (a *App) DeleteScanResult(filepath string) error {
	return a.taskManager.DeleteScanResult(filepath)
}

// ============ Results Methods ============

// GetScanResults returns results from a scan output file
func (a *App) GetScanResults(taskID int64) ([]*models.NucleiResult, error) {
	task, err := a.db.GetScanTaskByID(taskID)
	if err != nil {
		return nil, err
	}

	// Read output file
	data, err := json.Marshal([]byte(task.OutputFile))
	if err != nil {
		return nil, fmt.Errorf("failed to read results file: %w", err)
	}

	var results []*models.NucleiResult
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("failed to parse results: %w", err)
	}

	return results, nil
}

// ListResultFiles lists all result files in the results directory
func (a *App) ListResultFiles() ([]map[string]interface{}, error) {
	// This will be implemented to scan the results directory
	// and return metadata about each result file
	return nil, nil
}

// ============ Utility Methods ============

// CheckNucleiInstalled checks if nuclei is installed
func (a *App) CheckNucleiInstalled() (bool, string, error) {
	return config.CheckNucleiInstalled(a.config.NucleiPath)
}

// ValidateNucleiPath validates the current nuclei path and tries to fix it if needed
func (a *App) ValidateNucleiPath() error {
	if err := config.ValidateNucleiPath(a.config); err != nil {
		return err
	}
	
	// Save the updated config if it was changed
	return config.SaveConfig(a.config)
}

// NucleiTestResult represents the result of testing a nuclei path
type NucleiTestResult struct {
	Valid   bool   `json:"valid"`
	Version string `json:"version"`
	Error   string `json:"error,omitempty"`
}

// ProxyTestResult represents the result of a proxy test
type ProxyTestResult struct {
	URL         string `json:"url"`
	Available   bool   `json:"available"`
	ResponseTime int64  `json:"response_time"` // in milliseconds
	Error       string `json:"error,omitempty"`
}

// ProxyTestResults represents the results of testing multiple proxies
type ProxyTestResults struct {
	Results []ProxyTestResult `json:"results"`
	Summary struct {
		Total     int `json:"total"`
		Available int `json:"available"`
		Failed    int `json:"failed"`
	} `json:"summary"`
}

// TestNucleiPath tests a user-specified nuclei path
func (a *App) TestNucleiPath(userPath string) NucleiTestResult {
	valid, version, err := config.ValidateUserNucleiPath(userPath)
	result := NucleiTestResult{
		Valid:   valid,
		Version: version,
	}
	if err != nil {
		result.Error = err.Error()
	}
	return result
}

// SetNucleiPath sets a new nuclei path after validation
func (a *App) SetNucleiPath(newPath string) error {
	// Normalize the path
	normalizedPath := filepath.Clean(newPath)
	
	// First validate the new path
	valid, version, err := config.ValidateUserNucleiPath(normalizedPath)
	if err != nil {
		return fmt.Errorf("invalid nuclei path: %w", err)
	}
	
	if !valid {
		return fmt.Errorf("nuclei path validation failed")
	}
	
	// Update the config with normalized path
	a.config.NucleiPath = normalizedPath
	
	// Save the updated config
	if err := config.SaveConfig(a.config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	
	// Update JSONTaskManager config
	if a.jsonTaskManager != nil {
		a.jsonTaskManager.UpdateConfig(a.config)
	}
	
	runtime.LogInfof(a.ctx, "Nuclei path updated to: %s (version: %s)", normalizedPath, version)
	return nil
}

// ReloadConfig reloads the configuration and updates task managers
func (a *App) ReloadConfig() error {
	// Reload config from file
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}
	
	// Update app config
	a.config = cfg
	
	// Update JSONTaskManager config
	if a.jsonTaskManager != nil {
		a.jsonTaskManager.UpdateConfig(cfg)
	}
	
	// Update legacy task manager config
	if a.taskManager != nil {
		a.taskManager.UpdateConfig(cfg)
	}
	
	runtime.LogInfof(a.ctx, "Configuration reloaded successfully")
	return nil
}

// GetAppInfo returns application information
func (a *App) GetAppInfo() map[string]string {
	return map[string]string{
		"version": "1.0.0",
		"name":    "WePOC",
		"author":  "Security Team",
	}
}

// TestProxies tests the availability of proxy servers
func (a *App) TestProxies(proxyList []string) *ProxyTestResults {
	results := &ProxyTestResults{
		Results: make([]ProxyTestResult, 0, len(proxyList)),
	}
	
	// Test each proxy
	for _, proxyURL := range proxyList {
		if strings.TrimSpace(proxyURL) == "" {
			continue
		}
		
		result := a.testSingleProxy(proxyURL)
		results.Results = append(results.Results, result)
		
		if result.Available {
			results.Summary.Available++
		} else {
			results.Summary.Failed++
		}
	}
	
	results.Summary.Total = len(results.Results)
	return results
}

// testSingleProxy tests a single proxy server
func (a *App) testSingleProxy(proxyURL string) ProxyTestResult {
	result := ProxyTestResult{
		URL: proxyURL,
	}
	
	// Parse proxy URL
	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		result.Error = fmt.Sprintf("Invalid proxy URL: %v", err)
		return result
	}
	
	// Create HTTP client with proxy
	transport := &http.Transport{}
	
	// Set proxy based on scheme
	switch parsedURL.Scheme {
	case "http", "https":
		transport.Proxy = http.ProxyURL(parsedURL)
	case "socks5":
		// For SOCKS5, we'll test TCP connection directly
		return a.testSOCKS5Proxy(parsedURL)
	default:
		result.Error = "Unsupported proxy scheme"
		return result
	}
	
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}
	
	// Test proxy by making a request to a test URL
	start := time.Now()
	resp, err := client.Get("http://httpbin.org/ip")
	responseTime := time.Since(start).Milliseconds()
	
	if err != nil {
		result.Error = fmt.Sprintf("Connection failed: %v", err)
		result.ResponseTime = responseTime
		return result
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == 200 {
		result.Available = true
		result.ResponseTime = responseTime
	} else {
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		result.ResponseTime = responseTime
	}
	
	return result
}

// testSOCKS5Proxy tests SOCKS5 proxy by attempting TCP connection
func (a *App) testSOCKS5Proxy(parsedURL *url.URL) ProxyTestResult {
	result := ProxyTestResult{
		URL: parsedURL.String(),
	}
	
	// Extract host and port
	host := parsedURL.Hostname()
	port := parsedURL.Port()
	if port == "" {
		port = "1080" // Default SOCKS5 port
	}
	
	// Test TCP connection
	start := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 10*time.Second)
	responseTime := time.Since(start).Milliseconds()
	
	if err != nil {
		result.Error = fmt.Sprintf("Connection failed: %v", err)
		result.ResponseTime = responseTime
		return result
	}
	defer conn.Close()
	
	result.Available = true
	result.ResponseTime = responseTime
	return result
}

// GetTaskHTTPLogs returns HTTP request logs for a specific task
func (a *App) GetTaskHTTPLogs(taskID int64) ([]*scanner.HTTPRequestLog, error) {
	runtime.LogInfo(a.ctx, fmt.Sprintf("获取任务 %d 的HTTP请求日志", taskID))
	return a.jsonTaskManager.GetHTTPRequestLogs(taskID)
}

// GetPOCTemplateContent returns the raw YAML content of a POC template
func (a *App) GetPOCTemplateContent(templatePath string) (string, error) {
	runtime.LogInfo(a.ctx, fmt.Sprintf("读取POC模板内容: %s", templatePath))

	// 读取文件内容
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file: %w", err)
	}

	return string(content), nil
}

// ExportTaskResultAsJSON exports scan result as JSON file
func (a *App) ExportTaskResultAsJSON(taskID int64) (string, error) {
	runtime.LogInfo(a.ctx, fmt.Sprintf("导出任务 %d 的结果为JSON", taskID))

	// 获取扫描结果
	result, err := a.jsonTaskManager.GetTaskResult(taskID)
	if err != nil {
		return "", fmt.Errorf("failed to get task result: %w", err)
	}

	// 获取HTTP请求日志
	httpLogs, err := a.jsonTaskManager.GetHTTPRequestLogs(taskID)
	if err != nil {
		runtime.LogWarning(a.ctx, fmt.Sprintf("获取HTTP日志失败: %v", err))
		httpLogs = []*scanner.HTTPRequestLog{} // 使用空列表
	}

	// 组合完整的导出数据
	exportData := map[string]interface{}{
		"task_result":  result,
		"http_logs":    httpLogs,
		"exported_at":  time.Now().Format("2006-01-02 15:04:05"),
		"export_version": "1.0",
	}

	// 打开文件选择对话框
	defaultFilename := fmt.Sprintf("扫描结果_%s_%d.json", result.TaskName, taskID)
	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: defaultFilename,
		Title:           "导出扫描结果",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files (*.json)", Pattern: "*.json"},
		},
	})

	if err != nil || savePath == "" {
		return "", fmt.Errorf("用户取消导出")
	}

	// 序列化为JSON
	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(savePath, jsonData, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("✅ 导出成功: %s", savePath))
	return savePath, nil
}

// TestSinglePOCParams represents parameters for testing a single POC
type TestSinglePOCParams struct {
	TemplateContent string `json:"template_content"` // POC YAML content
	Target          string `json:"target"`           // Target URL
	Concurrency     int    `json:"concurrency"`      // Concurrency level
	RateLimit       int    `json:"rate_limit"`       // Rate limit
	InteractshURL   string `json:"interactsh_url"`   // Interactsh server URL
	InteractshToken string `json:"interactsh_token"` // Interactsh token
	ProxyURL        string `json:"proxy_url"`        // Proxy server URL
}

// TestSinglePOC tests a single POC template with custom parameters
func (a *App) TestSinglePOC(params TestSinglePOCParams) (map[string]interface{}, error) {
	runtime.LogInfo(a.ctx, fmt.Sprintf("测试单个POC: target=%s", params.Target))

	if params.TemplateContent == "" {
		return nil, fmt.Errorf("模板内容不能为空")
	}

	if params.Target == "" {
		return nil, fmt.Errorf("目标URL不能为空")
	}

	// Normalize target URL - add protocol if missing
	target := params.Target
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		// Check if it looks like a host:port format
		if strings.Contains(target, ":") && !strings.Contains(target, "://") {
			// For host:port format, try to determine protocol
			// Default to http for common web ports, otherwise use the target as-is
			parts := strings.Split(target, ":")
			if len(parts) == 2 {
				port := parts[1]
				switch port {
				case "80", "8080", "8000", "3000", "5000":
					target = "http://" + target
				case "443", "8443":
					target = "https://" + target
				default:
					// For other ports like Redis (6379), keep as-is without protocol
					// Nuclei can handle raw host:port for network protocols
				}
			}
		} else {
			// Plain hostname, default to http
			target = "http://" + target
		}
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("标准化目标: %s -> %s", params.Target, target))

	// Create temporary template file
	tmpDir := filepath.Join(os.TempDir(), "wepoc-poc-test")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, fmt.Errorf("无法创建临时目录: %w", err)
	}

	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("test-poc-%d.yaml", time.Now().UnixNano()))
	if err := os.WriteFile(tmpFile, []byte(params.TemplateContent), 0644); err != nil {
		return nil, fmt.Errorf("无法创建临时模板文件: %w", err)
	}
	defer os.Remove(tmpFile)

	// Build nuclei command arguments
	args := []string{
		"-t", tmpFile,
		"-u", target,
		"-jsonl", // Use JSONL format (newer Nuclei versions)
	}

	// Add concurrency
	if params.Concurrency > 0 {
		args = append(args, "-c", fmt.Sprintf("%d", params.Concurrency))
	}

	// Add rate limit
	if params.RateLimit > 0 {
		args = append(args, "-rl", fmt.Sprintf("%d", params.RateLimit))
	}

	// Add interactsh configuration
	if params.InteractshURL != "" {
		args = append(args, "-iserver", params.InteractshURL)
	}
	if params.InteractshToken != "" {
		args = append(args, "-itoken", params.InteractshToken)
	}

	// Add proxy
	if params.ProxyURL != "" {
		args = append(args, "-proxy", params.ProxyURL)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("Nuclei命令: %s %s", a.config.NucleiPath, strings.Join(args, " ")))

	// Execute nuclei command
	cmd := exec.Command(a.config.NucleiPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set timeout to 5 minutes
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("无法启动Nuclei: %w", err)
	}

	// Wait for completion or timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var execErr error
	select {
	case <-ctx.Done():
		cmd.Process.Kill()
		return nil, fmt.Errorf("测试超时（5分钟）")
	case execErr = <-done:
		// Don't return error immediately, check output first
	}

	// Get outputs
	stdoutOutput := stdout.String()
	stderrOutput := stderr.String()

	runtime.LogInfo(a.ctx, fmt.Sprintf("Nuclei stdout length: %d bytes", len(stdoutOutput)))
	runtime.LogInfo(a.ctx, fmt.Sprintf("Nuclei stderr length: %d bytes", len(stderrOutput)))

	if stderrOutput != "" {
		runtime.LogWarning(a.ctx, fmt.Sprintf("Nuclei stderr: %s", stderrOutput))
	}

	if stdoutOutput != "" {
		runtime.LogInfo(a.ctx, fmt.Sprintf("Nuclei stdout: %s", stdoutOutput))
	} else {
		runtime.LogWarning(a.ctx, "Nuclei stdout is empty")
	}

	if execErr != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("Nuclei exit error: %v", execErr))
	}

	// Parse output
	output := stdout.String()
	lines := strings.Split(output, "\n")

	var results []map[string]interface{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(line), &result); err == nil {
			results = append(results, result)
		}
	}

	// Build response
	response := map[string]interface{}{
		"success":       execErr == nil,
		"results_count": len(results),
		"results":       results,
		"raw_output":    output,
		"stderr":        stderrOutput,
	}

	if execErr != nil {
		// Check if it's just "no results" (exit code 2) or a real error
		exitCode := -1
		if exitError, ok := execErr.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}

		runtime.LogInfo(a.ctx, fmt.Sprintf("Nuclei exit code: %d", exitCode))

		// Exit code 2 often means "no vulnerabilities found" in Nuclei
		// Only treat it as error if we have no output at all
		if exitCode == 2 && (len(results) > 0 || stdoutOutput != "") {
			// This is likely just "no vulnerabilities found"
			response["success"] = true
			if len(results) > 0 {
				response["message"] = fmt.Sprintf("发现 %d 个漏洞", len(results))
			} else {
				response["message"] = "测试完成，未发现漏洞"
			}
			response["warning"] = "Nuclei 返回退出码 2（通常表示未发现漏洞）"
		} else if len(results) > 0 {
			// We have results despite error
			response["message"] = fmt.Sprintf("测试完成（有警告），发现 %d 个漏洞", len(results))
			response["warning"] = fmt.Sprintf("Nuclei 退出码 %d: %v", exitCode, execErr)
		} else {
			// Real error - no output and error
			response["success"] = false
			response["message"] = "测试失败"
			response["error"] = fmt.Sprintf("退出码 %d: %v", exitCode, execErr)
			if stderrOutput != "" {
				response["error_detail"] = stderrOutput
			} else if stdoutOutput != "" {
				response["error_detail"] = "标准输出: " + stdoutOutput
			}
			runtime.LogError(a.ctx, fmt.Sprintf("❌ Nuclei测试失败: 退出码 %d, stdout: %s, stderr: %s", exitCode, stdoutOutput, stderrOutput))
			return response, fmt.Errorf("Nuclei执行失败 (退出码 %d): %v\nStdout: %s\nStderr: %s", exitCode, execErr, stdoutOutput, stderrOutput)
		}
	} else {
		if len(results) == 0 {
			response["message"] = "未发现漏洞"
		} else {
			response["message"] = fmt.Sprintf("发现 %d 个漏洞", len(results))
		}
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("✅ 测试完成: %s", response["message"]))
	return response, nil
}

// SavePOCTemplate saves modified POC template content to file
func (a *App) SavePOCTemplate(templatePath string, content string) error {
	runtime.LogInfo(a.ctx, fmt.Sprintf("保存POC模板: %s", templatePath))

	if templatePath == "" {
		return fmt.Errorf("模板路径不能为空")
	}

	if content == "" {
		return fmt.Errorf("模板内容不能为空")
	}

	// Validate template path is within POC directory
	pocDir := a.config.POCDirectory
	absTemplatePath, err := filepath.Abs(templatePath)
	if err != nil {
		return fmt.Errorf("无法解析模板路径: %w", err)
	}

	absPOCDir, err := filepath.Abs(pocDir)
	if err != nil {
		return fmt.Errorf("无法解析POC目录: %w", err)
	}

	if !strings.HasPrefix(absTemplatePath, absPOCDir) {
		return fmt.Errorf("模板路径必须在POC目录内")
	}

	// Backup original file
	backupPath := templatePath + ".backup." + time.Now().Format("20060102150405")
	if err := copyFile(templatePath, backupPath); err != nil {
		runtime.LogWarning(a.ctx, fmt.Sprintf("无法创建备份文件: %v", err))
	} else {
		runtime.LogInfo(a.ctx, fmt.Sprintf("已创建备份: %s", backupPath))
	}

	// Write new content
	if err := os.WriteFile(templatePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("无法保存模板文件: %w", err)
	}

	runtime.LogInfo(a.ctx, "✅ POC模板保存成功")
	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// SaveCSVFile opens a save dialog and returns the selected file path
func (a *App) SaveCSVFile(defaultFilename string, csvContent string) (string, error) {
	runtime.LogInfo(a.ctx, fmt.Sprintf("打开保存对话框: %s", defaultFilename))

	// Open save file dialog
	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: defaultFilename,
		Title:           "导出 CSV 文件",
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV 文件 (*.csv)", Pattern: "*.csv"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})

	if err != nil || savePath == "" {
		return "", fmt.Errorf("用户取消保存")
	}

	// Add BOM for Excel Chinese support
	BOM := "\uFEFF"
	content := BOM + csvContent

	// Write file
	if err := os.WriteFile(savePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("保存文件失败: %w", err)
	}

	runtime.LogInfo(a.ctx, fmt.Sprintf("✅ CSV文件保存成功: %s", savePath))
	return savePath, nil
}
