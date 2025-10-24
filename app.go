package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		"name":    "wepoc",
		"version": "1.0.0",
		"author":  "wepoc team",
	}
}
