package scanner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"wepoc/internal/models"
)

// JSONTaskManager manages scanning tasks using JSON files instead of database
type JSONTaskManager struct {
	tasksDir      string
	resultsDir    string
	logsDir       string
	mu            sync.RWMutex
	nextTaskID    int64
	eventHandlers map[int64]func(*ScanEvent) // Task ID -> event handler
	handlersMu    sync.RWMutex
	config        *models.Config // Add configuration support
}

// TaskConfig represents a task configuration stored in JSON
type TaskConfig struct {
	ID                int64      `json:"id"`
	Name              string     `json:"name"`
	Status            string     `json:"status"` // pending, running, completed, failed
	POCs              []string   `json:"pocs"`
	Targets           []string   `json:"targets"`
	TotalRequests     int        `json:"total_requests"`
	CompletedRequests int        `json:"completed_requests"`
	FoundVulns        int        `json:"found_vulns"`
	StartTime         time.Time  `json:"start_time"`
	EndTime           *time.Time `json:"end_time,omitempty"`
	OutputFile        string     `json:"output_file"`
	LogFile           string     `json:"log_file"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// TaskResult represents the scan result stored in JSON
type TaskResult struct {
	TaskID            int64                  `json:"task_id"`
	TaskName          string                 `json:"task_name"`
	Status            string                 `json:"status"`
	StartTime         time.Time              `json:"start_time"`
	EndTime           time.Time              `json:"end_time"`
	Duration          string                 `json:"duration"`
	Targets           []string               `json:"targets"`
	Templates         []string               `json:"templates"`
	TemplateCount     int                    `json:"template_count"`
	TargetCount       int                    `json:"target_count"`
	TotalRequests     int                    `json:"total_requests"`
	CompletedRequests int                    `json:"completed_requests"`
	FoundVulns        int                    `json:"found_vulns"`
	SuccessRate       float64                `json:"success_rate"`
	Vulnerabilities   []*models.NucleiResult `json:"vulnerabilities"`
	Summary           map[string]interface{} `json:"summary"`
	CreatedAt         time.Time              `json:"created_at"`

	// 新增：详细统计信息
	ScannedTemplates    int      `json:"scanned_templates"`      // 实际扫描的模板数量
	FilteredTemplates   int      `json:"filtered_templates"`     // 被Nuclei过滤的模板数量
	SkippedTemplates    int      `json:"skipped_templates"`      // 被跳过的模板数量
	FailedTemplates     int      `json:"failed_templates"`       // 扫描失败的模板数量
	FilteredTemplateIDs []string `json:"filtered_template_ids"`  // 被过滤的模板ID列表
	SkippedTemplateIDs  []string `json:"skipped_template_ids"`   // 被跳过的模板ID列表
	FailedTemplateIDs   []string `json:"failed_template_ids"`    // 失败的模板ID列表
	ScannedTemplateIDs  []string `json:"scanned_template_ids"`   // 已扫描的模板ID列表
	HTTPRequests        int      `json:"http_requests"`          // 实际HTTP请求数量
}

// NewJSONTaskManager creates a new JSON-based task manager
func NewJSONTaskManager(config *models.Config) (*JSONTaskManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	baseDir := filepath.Join(homeDir, ".wepoc")
	tasksDir := filepath.Join(baseDir, "tasks")
	resultsDir := filepath.Join(baseDir, "results")
	logsDir := filepath.Join(baseDir, "logs")

	// Create directories if they don't exist
	for _, dir := range []string{tasksDir, resultsDir, logsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Find the next task ID
	nextID, err := findNextTaskID(tasksDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find next task ID: %w", err)
	}

	return &JSONTaskManager{
		tasksDir:      tasksDir,
		resultsDir:    resultsDir,
		logsDir:       logsDir,
		nextTaskID:    nextID,
		eventHandlers: make(map[int64]func(*ScanEvent)),
		config:        config,
	}, nil
}

// findNextTaskID finds the next available task ID
func findNextTaskID(tasksDir string) (int64, error) {
	files, err := filepath.Glob(filepath.Join(tasksDir, "task_*.json"))
	if err != nil {
		return 1, nil // Start from 1 if no files exist
	}

	maxID := int64(0)
	for _, file := range files {
		filename := filepath.Base(file)
		if len(filename) > 10 && filename[:5] == "task_" {
			idStr := filename[5 : len(filename)-5] // Remove "task_" and ".json"
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
				if id > maxID {
					maxID = id
				}
			}
		}
	}

	return maxID + 1, nil
}

// CreateTask creates a new scanning task
func (tm *JSONTaskManager) CreateTask(pocs []string, targets []string, taskName string) (*TaskConfig, error) {
	fmt.Printf("=== JSONTaskManager.CreateTask called ===\n")
	fmt.Printf("POCs: %v\n", pocs)
	fmt.Printf("Targets: %v\n", targets)
	fmt.Printf("TaskName: %s\n", taskName)

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Use provided task name or generate default
	if taskName == "" {
		taskName = fmt.Sprintf("Task-%d", tm.nextTaskID)
	}

	now := time.Now()
	taskID := tm.nextTaskID
	tm.nextTaskID++

	fmt.Printf("Assigned task ID: %d\n", taskID)

	// Create task configuration
	task := &TaskConfig{
		ID:                taskID,
		Name:              taskName,
		Status:            "pending",
		POCs:              pocs,
		Targets:           targets,
		TotalRequests:     len(pocs) * len(targets),
		CompletedRequests: 0,
		FoundVulns:        0,
		StartTime:         now,
		OutputFile:        filepath.Join(tm.resultsDir, fmt.Sprintf("task_%d_result.json", taskID)),
		LogFile:           filepath.Join(tm.logsDir, fmt.Sprintf("task_%d.log", taskID)),
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	fmt.Printf("Task config created: %+v\n", task)

	// Save task configuration to JSON file
	if err := tm.saveTaskConfig(task); err != nil {
		fmt.Printf("Failed to save task config: %v\n", err)
		return nil, fmt.Errorf("failed to save task config: %w", err)
	}

	fmt.Printf("Task config saved successfully to disk\n")

	return task, nil
}

// StartTask starts a scanning task
func (tm *JSONTaskManager) StartTask(taskID int64) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Load task configuration
	task, err := tm.loadTaskConfig(taskID)
	if err != nil {
		return fmt.Errorf("failed to load task config: %w", err)
	}

	// Update task status to running
	task.Status = "running"
	task.StartTime = time.Now()
	task.UpdatedAt = time.Now()

	// 重置任务的进度数据（清零）
	task.CompletedRequests = 0
	task.FoundVulns = 0

	// Save updated task configuration
	if err := tm.saveTaskConfig(task); err != nil {
		return fmt.Errorf("failed to save task config: %w", err)
	}

	// 发送初始化的进度事件到前端，清零旧数据
	fmt.Printf("🔄 任务 %d 启动，发送初始化进度事件（清零）\n", taskID)
	initialProgress := &ScanProgress{
		TaskID:              taskID,
		TotalRequests:       task.TotalRequests,
		CompletedRequests:   0,
		FoundVulns:          0,
		Percentage:          0,
		Status:              "running",
		CurrentTemplate:     "",
		CurrentTarget:       "",
		TotalTemplates:      len(task.POCs),
		CompletedTemplates:  0,
		ScannedTemplates:    0,
		FailedTemplates:     0,
		FilteredTemplates:   0,
		SkippedTemplates:    0,
		CurrentIndex:        0,
		SelectedTemplates:   task.POCs,
		ScannedTemplateIDs:  []string{},
		FailedTemplateIDs:   []string{},
		FilteredTemplateIDs: []string{},
		SkippedTemplateIDs:  []string{},
	}

	tm.emitEvent(taskID, &ScanEvent{
		TaskID:    taskID,
		EventType: "progress",
		Data:      initialProgress,
	})

	// Start scanning in background
	go tm.runScanTask(task)

	return nil
}

// UpdateConfig updates the configuration
func (tm *JSONTaskManager) UpdateConfig(config *models.Config) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.config = config
}

// RescanTask restarts a completed or failed task with the same configuration
func (tm *JSONTaskManager) RescanTask(taskID int64) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Load task configuration
	task, err := tm.loadTaskConfig(taskID)
	if err != nil {
		return fmt.Errorf("failed to load task config: %w", err)
	}

	// Check if task can be rescanned
	if task.Status != "completed" && task.Status != "failed" {
		return fmt.Errorf("task %d is not in a rescanable state (current status: %s)", taskID, task.Status)
	}

	// Reset task state for rescan
	task.Status = "running"
	task.StartTime = time.Now()
	task.EndTime = nil
	task.CompletedRequests = 0
	task.FoundVulns = 0
	task.UpdatedAt = time.Now()

	// Clear previous results and logs
	task.OutputFile = filepath.Join(tm.resultsDir, fmt.Sprintf("task_%d_result.json", taskID))
	task.LogFile = filepath.Join(tm.logsDir, fmt.Sprintf("task_%d.log", taskID))

	// Save updated task configuration
	if err := tm.saveTaskConfig(task); err != nil {
		return fmt.Errorf("failed to save task config: %w", err)
	}

	// Start scanning in background
	go tm.runScanTask(task)

	return nil
}

// RegisterEventHandler registers an event handler for a task
func (tm *JSONTaskManager) RegisterEventHandler(taskID int64, handler func(*ScanEvent)) {
	tm.handlersMu.Lock()
	defer tm.handlersMu.Unlock()
	tm.eventHandlers[taskID] = handler
}

// UnregisterEventHandler unregisters the event handler for a task
func (tm *JSONTaskManager) UnregisterEventHandler(taskID int64) {
	tm.handlersMu.Lock()
	defer tm.handlersMu.Unlock()
	delete(tm.eventHandlers, taskID)
}

// emitEvent emits an event to the registered handler
func (tm *JSONTaskManager) emitEvent(taskID int64, event *ScanEvent) {
	tm.handlersMu.RLock()
	handler, exists := tm.eventHandlers[taskID]
	tm.handlersMu.RUnlock()

	if exists && handler != nil {
		handler(event)
	}
}

// runScanTask executes the scanning task
func (tm *JSONTaskManager) runScanTask(task *TaskConfig) {
	// Create a simple scanner that runs nuclei and saves results
	scanner := NewSimpleNucleiScanner(task, tm)

	// Listen to scanner events and forward them
	go func() {
		for event := range scanner.GetEventChannel() {
			tm.emitEvent(task.ID, event)

			// Update task configuration based on event
			if event.EventType == "progress" {
				if progress, ok := event.Data.(*ScanProgress); ok {
					tm.mu.Lock()
					task.CompletedRequests = progress.CompletedRequests
					task.FoundVulns = progress.FoundVulns
					task.Status = progress.Status
					task.UpdatedAt = time.Now()

					// 如果任务完成，设置结束时间
					if progress.Status == "completed" || progress.Status == "failed" {
						now := time.Now()
						task.EndTime = &now
						fmt.Printf("✅ 任务 %d 完成，设置EndTime并保存状态: %s\n", task.ID, progress.Status)
					}

					tm.saveTaskConfig(task)
					tm.mu.Unlock()

					// Also emit event to ensure it's received by frontend
					tm.emitEvent(task.ID, event)
				}
			}
		}

		// Unregister event handler when done
		tm.UnregisterEventHandler(task.ID)
	}()

	// Run the scan
	err := scanner.Start()

	// Update task status based on result
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Reload task to get latest state
	task, loadErr := tm.loadTaskConfig(task.ID)
	if loadErr != nil {
		fmt.Printf("Failed to reload task config: %v\n", loadErr)
		return
	}

	now := time.Now()
	task.EndTime = &now
	task.UpdatedAt = now

	if err != nil {
		task.Status = "failed"
		fmt.Printf("Task %d failed: %v\n", task.ID, err)
	} else {
		task.Status = "completed"
		fmt.Printf("Task %d completed successfully\n", task.ID)
	}

	// Save final task configuration
	if saveErr := tm.saveTaskConfig(task); saveErr != nil {
		fmt.Printf("Failed to save final task config: %v\n", saveErr)
	}
}

// GetAllTasks returns all tasks
func (tm *JSONTaskManager) GetAllTasks() ([]*TaskConfig, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	files, err := filepath.Glob(filepath.Join(tm.tasksDir, "task_*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list task files: %w", err)
	}

	var tasks []*TaskConfig
	for _, file := range files {
		task, err := tm.loadTaskConfigFromFile(file)
		if err != nil {
			fmt.Printf("Failed to load task from file %s: %v\n", file, err)
			continue
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetTaskByID returns a specific task
// UpdateTask updates an existing task configuration
func (tm *JSONTaskManager) UpdateTask(taskID int64, pocs []string, targets []string, taskName string) (*TaskConfig, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Load existing task
	task, err := tm.loadTaskConfig(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to load task: %v", err)
	}

	// Check if task is running
	if task.Status == "running" {
		return nil, fmt.Errorf("cannot update running task")
	}

	// Update task fields
	task.Name = taskName
	task.POCs = pocs
	task.Targets = targets
	task.UpdatedAt = time.Now()

	// Calculate total requests
	task.TotalRequests = len(pocs) * len(targets)

	// Save updated task
	if err := tm.saveTaskConfig(task); err != nil {
		return nil, fmt.Errorf("failed to save updated task: %v", err)
	}

	return task, nil
}

func (tm *JSONTaskManager) GetTaskByID(taskID int64) (*TaskConfig, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.loadTaskConfig(taskID)
}

// DeleteTask deletes a task and its associated files
func (tm *JSONTaskManager) DeleteTask(taskID int64) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Load task to get file paths
	task, err := tm.loadTaskConfig(taskID)
	if err != nil {
		return fmt.Errorf("failed to load task: %w", err)
	}

	// Delete task configuration file
	taskFile := filepath.Join(tm.tasksDir, fmt.Sprintf("task_%d.json", taskID))
	if err := os.Remove(taskFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete task file: %w", err)
	}

	// Delete result file
	if err := os.Remove(task.OutputFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete result file: %w", err)
	}

	// Delete log file
	if err := os.Remove(task.LogFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete log file: %w", err)
	}

	return nil
}

// GetTaskResult returns the scan result for a task
func (tm *JSONTaskManager) GetTaskResult(taskID int64) (*TaskResult, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// Load task configuration
	task, err := tm.loadTaskConfig(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to load task: %w", err)
	}

	// Check if result file exists
	if _, err := os.Stat(task.OutputFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("result file does not exist")
	}

	// Load result file
	result, err := tm.loadTaskResult(task.OutputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load result: %w", err)
	}

	return result, nil
}

// GetAllTaskResults returns all task results with vulnerabilities found
func (tm *JSONTaskManager) GetAllTaskResults() ([]*TaskResult, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	files, err := filepath.Glob(filepath.Join(tm.resultsDir, "task_*_result.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list result files: %w", err)
	}

	var results []*TaskResult
	for _, file := range files {
		result, err := tm.loadTaskResult(file)
		if err != nil {
			fmt.Printf("Failed to load result from file %s: %v\n", file, err)
			continue
		}
		// Only include results that have vulnerabilities found
		if result.FoundVulns > 0 && len(result.Vulnerabilities) > 0 {
			results = append(results, result)
		}
	}

	return results, nil
}

// Helper methods

func (tm *JSONTaskManager) saveTaskConfig(task *TaskConfig) error {
	filename := filepath.Join(tm.tasksDir, fmt.Sprintf("task_%d.json", task.ID))
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func (tm *JSONTaskManager) loadTaskConfig(taskID int64) (*TaskConfig, error) {
	filename := filepath.Join(tm.tasksDir, fmt.Sprintf("task_%d.json", taskID))
	return tm.loadTaskConfigFromFile(filename)
}

func (tm *JSONTaskManager) loadTaskConfigFromFile(filename string) (*TaskConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var task TaskConfig
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

func (tm *JSONTaskManager) loadTaskResult(filename string) (*TaskResult, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var result TaskResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (tm *JSONTaskManager) saveTaskResult(result *TaskResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	// Create result file path
	resultFile := filepath.Join(tm.resultsDir, fmt.Sprintf("task_%d_result.json", result.TaskID))
	return os.WriteFile(resultFile, data, 0644)
}

// SaveHTTPRequestLogs saves HTTP request logs for a task
func (tm *JSONTaskManager) SaveHTTPRequestLogs(taskID int64, logs []*HTTPRequestLog) error {
	data, err := json.MarshalIndent(logs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal HTTP logs: %w", err)
	}

	httpLogsFile := filepath.Join(tm.logsDir, fmt.Sprintf("task_%d_http_logs.json", taskID))
	if err := os.WriteFile(httpLogsFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write HTTP logs: %w", err)
	}

	fmt.Printf("✅ HTTP请求日志已保存: %s (%d 条记录)\n", httpLogsFile, len(logs))
	return nil
}

// GetHTTPRequestLogs returns HTTP request logs for a task
func (tm *JSONTaskManager) GetHTTPRequestLogs(taskID int64) ([]*HTTPRequestLog, error) {
	httpLogsFile := filepath.Join(tm.logsDir, fmt.Sprintf("task_%d_http_logs.json", taskID))

	// Check if file exists
	if _, err := os.Stat(httpLogsFile); os.IsNotExist(err) {
		// Return empty list if file doesn't exist
		return []*HTTPRequestLog{}, nil
	}

	data, err := os.ReadFile(httpLogsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read HTTP logs: %w", err)
	}

	var logs []*HTTPRequestLog
	if err := json.Unmarshal(data, &logs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal HTTP logs: %w", err)
	}

	return logs, nil
}
