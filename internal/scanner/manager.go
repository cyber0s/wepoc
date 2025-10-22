package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"wepoc/internal/database"
	"wepoc/internal/models"
)

// TaskManager manages multiple scanning tasks
type TaskManager struct {
	tasks      map[int64]*TaskRunner // TaskID -> TaskRunner
	nextTaskID int64
	maxTasks   int // Maximum concurrent tasks
	mu         sync.RWMutex
	db         *database.Database
	eventChan  chan *TaskEvent // For sending events to frontend
	taskLogs   map[int64][]*models.ScanLog // TaskID -> logs for each task
	logsMu     sync.RWMutex
	config     *models.Config // Add configuration support
}

// TaskRunner represents a single running task
type TaskRunner struct {
	ID          int64
	Task        *models.ScanTask
	Scanner     *NucleiScanner
	Context     context.Context
	CancelFunc  context.CancelFunc
	Status      string
	StartTime   time.Time
	LastUpdate  time.Time
}

// TaskEvent represents an event emitted during scanning
type TaskEvent struct {
	TaskID    int64                  `json:"task_id"`
	EventType string                 `json:"event_type"` // progress, log, completed, error
	Data      interface{}            `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewTaskManager creates a new task manager
func NewTaskManager(db *database.Database, maxTasks int, config *models.Config) *TaskManager {
	return &TaskManager{
		tasks:     make(map[int64]*TaskRunner),
		maxTasks:  maxTasks,
		db:        db,
		eventChan: make(chan *TaskEvent, 100),
		taskLogs:  make(map[int64][]*models.ScanLog),
		config:    config,
	}
}

// GetEventChannel returns the event channel for frontend communication
func (tm *TaskManager) GetEventChannel() <-chan *TaskEvent {
	return tm.eventChan
}

// AddTaskLog adds a log entry for a specific task
func (tm *TaskManager) AddTaskLog(taskID int64, log *models.ScanLog) {
	tm.logsMu.Lock()
	defer tm.logsMu.Unlock()
	
	if tm.taskLogs[taskID] == nil {
		tm.taskLogs[taskID] = make([]*models.ScanLog, 0)
	}
	tm.taskLogs[taskID] = append(tm.taskLogs[taskID], log)
	
	// Send log event to frontend
	tm.eventChan <- &TaskEvent{
		TaskID:    taskID,
		EventType: "log",
		Data:      log,
		Timestamp: time.Now(),
	}
}

// GetTaskLogs returns the logs for a specific task
func (tm *TaskManager) GetTaskLogs(taskID int64) ([]*models.ScanLog, error) {
	tm.logsMu.RLock()
	defer tm.logsMu.RUnlock()
	
	logs, exists := tm.taskLogs[taskID]
	if !exists {
		return []*models.ScanLog{}, nil
	}
	
	// Return a copy of the logs
	result := make([]*models.ScanLog, len(logs))
	copy(result, logs)
	return result, nil
}

// GetTaskLogSummary returns the log summary for a specific task
func (tm *TaskManager) GetTaskLogSummary(taskID int64) (map[string]interface{}, error) {
	// Try to get from running task first
	tm.mu.RLock()
	runner, exists := tm.tasks[taskID]
	tm.mu.RUnlock()
	
	if exists && runner.Scanner != nil && runner.Scanner.LogParser != nil {
		summary := runner.Scanner.LogParser.GetSummary()
		// Convert to map for JSON serialization
		summaryMap := map[string]interface{}{
			"task_id":             summary.TaskID,
			"start_time":          summary.StartTime,
			"end_time":            summary.EndTime,
			"duration":            summary.Duration,
			"total_requests":      summary.TotalRequests,
			"completed_requests":  summary.CompletedRequests,
			"found_vulns":         summary.FoundVulns,
			"success_rate":        summary.SuccessRate,
			"key_events":          summary.KeyEvents,
			"vulnerabilities":     summary.Vulnerabilities,
			"errors":              summary.Errors,
			"warnings":            summary.Warnings,
		}
		return summaryMap, nil
	}
	
	// If not running, try to load from file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	
	resultsDir := filepath.Join(homeDir, ".wepoc", "results")
	
	// Look for log summary files for this task
	pattern := filepath.Join(resultsDir, fmt.Sprintf("task_%d_log_summary_*.json", taskID))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search for log summary files: %w", err)
	}
	
	if len(matches) == 0 {
		return map[string]interface{}{
			"task_id":             taskID,
			"start_time":          time.Time{},
			"end_time":            time.Time{},
			"duration":            "",
			"total_requests":      0,
			"completed_requests":  0,
			"found_vulns":         0,
			"success_rate":        0.0,
			"key_events":          []interface{}{},
			"vulnerabilities":     []interface{}{},
			"errors":              []interface{}{},
			"warnings":            []interface{}{},
		}, nil
	}
	
	// Read the most recent log summary file
	latestFile := matches[len(matches)-1]
	content, err := os.ReadFile(latestFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read log summary file: %w", err)
	}
	
	var summaryMap map[string]interface{}
	if err := json.Unmarshal(content, &summaryMap); err != nil {
		return nil, fmt.Errorf("failed to parse log summary file: %w", err)
	}
	
	return summaryMap, nil
}

// GetScanResult returns the comprehensive scan result for a specific task
func (tm *TaskManager) GetScanResult(taskID int64) (map[string]interface{}, error) {
	// Create result manager
	resultManager, err := NewResultManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create result manager: %w", err)
	}
	
	// Get scan result
	result, err := resultManager.GetScanResultByTaskID(taskID)
	if err != nil {
		return nil, err
	}
	
	// Convert to map for JSON serialization
	resultMap := map[string]interface{}{
		"id":                result.ID,
		"task_id":           result.TaskID,
		"task_name":         result.TaskName,
		"status":            result.Status,
		"start_time":        result.StartTime,
		"end_time":          result.EndTime,
		"duration":          result.Duration,
		"targets":           result.Targets,
		"templates":         result.Templates,
		"template_count":    result.TemplateCount,
		"target_count":      result.TargetCount,
		"total_requests":    result.TotalRequests,
		"completed_requests": result.CompletedRequests,
		"found_vulns":       result.FoundVulns,
		"success_rate":      result.SuccessRate,
		"vulnerabilities":   result.Vulnerabilities,
		"log_summary":       result.LogSummary,
		"output_file":       result.OutputFile,
		"log_file":          result.LogFile,
		"created_at":        result.CreatedAt,
		"updated_at":        result.UpdatedAt,
	}
	
	return resultMap, nil
}

// GetAllScanResults returns all scan results
func (tm *TaskManager) GetAllScanResults() ([]map[string]interface{}, error) {
	// Create result manager
	resultManager, err := NewResultManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create result manager: %w", err)
	}
	
	// Get all scan results
	results, err := resultManager.GetAllScanResults()
	if err != nil {
		return nil, err
	}
	
	// Convert to maps for JSON serialization
	var resultMaps []map[string]interface{}
	for _, result := range results {
		resultMap := map[string]interface{}{
			"id":                result.ID,
			"task_id":           result.TaskID,
			"task_name":         result.TaskName,
			"status":            result.Status,
			"start_time":        result.StartTime,
			"end_time":          result.EndTime,
			"duration":          result.Duration,
			"targets":           result.Targets,
			"templates":         result.Templates,
			"template_count":    result.TemplateCount,
			"target_count":      result.TargetCount,
			"total_requests":    result.TotalRequests,
			"completed_requests": result.CompletedRequests,
			"found_vulns":       result.FoundVulns,
			"success_rate":      result.SuccessRate,
			"vulnerabilities":   result.Vulnerabilities,
			"log_summary":       result.LogSummary,
			"output_file":       result.OutputFile,
			"log_file":          result.LogFile,
			"created_at":        result.CreatedAt,
			"updated_at":        result.UpdatedAt,
		}
		resultMaps = append(resultMaps, resultMap)
	}
	
	return resultMaps, nil
}

// DeleteScanResult deletes a scan result file
func (tm *TaskManager) DeleteScanResult(filepath string) error {
	// Create result manager
	resultManager, err := NewResultManager()
	if err != nil {
		return fmt.Errorf("failed to create result manager: %w", err)
	}
	
	return resultManager.DeleteScanResult(filepath)
}

// CreateTask creates a new scanning task
func (tm *TaskManager) CreateTask(pocs []string, targets []string, taskName string) (*models.ScanTask, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check if we can create a new task
	runningTasks := tm.countRunningTasks()
	if runningTasks >= tm.maxTasks {
		return nil, fmt.Errorf("maximum number of concurrent tasks (%d) reached", tm.maxTasks)
	}

	// Use provided task name or generate default
	if taskName == "" {
		taskName = fmt.Sprintf("Task-%d", tm.nextTaskID+1)
	}

	// Create task object
	task := &models.ScanTask{
		Name:              taskName,
		Status:            "pending",
		POCs:              convertToJSON(pocs),
		Targets:           convertToJSON(targets),
		TotalRequests:     len(pocs) * len(targets),
		CompletedRequests: 0,
		FoundVulns:        0,
		StartTime:         time.Now(),
		OutputFile:        fmt.Sprintf("~/.wepoc/results/task_%d_%s.json", tm.nextTaskID+1, time.Now().Format("20060102_150405")),
	}

	// Insert task into database
	if err := tm.db.InsertScanTask(task); err != nil {
		return nil, fmt.Errorf("failed to insert task: %w", err)
	}

	tm.nextTaskID = task.ID

	return task, nil
}

// StartTask starts a scanning task
func (tm *TaskManager) StartTask(taskID int64) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check if task is already running
	if _, exists := tm.tasks[taskID]; exists {
		return fmt.Errorf("task %d is already running", taskID)
	}

	// Check if we can start a new task
	runningTasks := tm.countRunningTasks()
	if runningTasks >= tm.maxTasks {
		return fmt.Errorf("maximum number of concurrent tasks (%d) reached", tm.maxTasks)
	}

	// Get task from database
	task, err := tm.db.GetScanTaskByID(taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Create scanner with task manager reference for logging
	scanner := NewNucleiScanner(task, tm.db, tm, taskID)

	// Create task runner
	runner := &TaskRunner{
		ID:         taskID,
		Task:       task,
		Scanner:    scanner,
		Context:    ctx,
		CancelFunc: cancel,
		Status:     "running",
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
	}

	tm.tasks[taskID] = runner

	// Update task status in database
	if err := tm.db.UpdateScanTaskStatus(taskID, "running"); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Start scanning in goroutine
	go tm.runTask(runner)

	return nil
}

// runTask executes the scanning task
func (tm *TaskManager) runTask(runner *TaskRunner) {
	defer func() {
		tm.mu.Lock()
		delete(tm.tasks, runner.ID)
		tm.mu.Unlock()
	}()

	// Start scanner with progress callback
	progressChan := make(chan *models.TaskProgress, 10)
	
	// Create a ticker for periodic database updates
	dbUpdateTicker := time.NewTicker(2 * time.Second)
	defer dbUpdateTicker.Stop()
	
	// Last progress state to avoid duplicate database updates
	var lastProgress *models.TaskProgress
	
	go func() {
		for {
			select {
			case progress, ok := <-progressChan:
				if !ok {
					// Channel closed, exit goroutine
					return
				}
				
				// Update database if progress has changed significantly
				shouldUpdateDB := false
				if lastProgress == nil {
					shouldUpdateDB = true
				} else if progress.CompletedRequests != lastProgress.CompletedRequests ||
					progress.FoundVulns != lastProgress.FoundVulns ||
					progress.Status != lastProgress.Status {
					shouldUpdateDB = true
				}
				
				if shouldUpdateDB {
					// Update database
					tm.db.UpdateScanTaskProgress(
						runner.ID,
						progress.TotalRequests,
						progress.CompletedRequests,
						progress.FoundVulns,
					)
					lastProgress = progress
				}

				// Send event to frontend
				tm.eventChan <- &TaskEvent{
					TaskID:    runner.ID,
					EventType: "progress",
					Data:      progress,
					Timestamp: time.Now(),
				}
				
			case <-dbUpdateTicker.C:
				// Periodic database update to ensure consistency
				if lastProgress != nil {
					tm.db.UpdateScanTaskProgress(
						runner.ID,
						lastProgress.TotalRequests,
						lastProgress.CompletedRequests,
						lastProgress.FoundVulns,
					)
				}
			}
		}
	}()

	// Run scanner
	err := runner.Scanner.Start(runner.Context, progressChan)
	close(progressChan)

	// Handle completion
	if err != nil {
		// Task failed or timed out
		// Get final progress before marking as failed
		finalProgress := runner.Scanner.GetProgress()
		
		// Update database with final progress including vulnerability count
		tm.db.UpdateScanTaskProgress(
			runner.ID,
			finalProgress.TotalRequests,
			finalProgress.CompletedRequests,
			finalProgress.FoundVulns,
		)
		
		// Mark task as failed
		tm.db.UpdateScanTaskStatus(runner.ID, "failed")
		
		// Set final status
		finalProgress.Status = "failed"
		
		tm.eventChan <- &TaskEvent{
			TaskID:    runner.ID,
			EventType: "error",
			Data:      map[string]interface{}{
				"error": err.Error(),
				"progress": finalProgress,
			},
			Timestamp: time.Now(),
		}
	} else {
		// Task completed successfully
		// Get final progress before completing the task
		finalProgress := runner.Scanner.GetProgress()
		
		// Update database with final progress including vulnerability count
		tm.db.UpdateScanTaskProgress(
			runner.ID,
			finalProgress.TotalRequests,
			finalProgress.CompletedRequests,
			finalProgress.FoundVulns,
		)
		
		// Mark task as completed
		tm.db.CompleteScanTask(runner.ID)
		
		// Set final status
		finalProgress.Status = "completed"
		
		// Get log summary for completed task
		var logSummary interface{}
		if runner.Scanner.LogParser != nil {
			summary := runner.Scanner.LogParser.GetSummary()
			logSummary = map[string]interface{}{
				"total_requests": summary.TotalRequests,
				"completed_requests": summary.CompletedRequests,
				"found_vulns": summary.FoundVulns,
				"duration": summary.Duration,
				"success_rate": summary.SuccessRate,
			}
		}
		
		tm.eventChan <- &TaskEvent{
			TaskID:    runner.ID,
			EventType: "completed",
			Data: map[string]interface{}{
				"progress": finalProgress,
				"summary": logSummary,
			},
			Timestamp: time.Now(),
		}
	}
}

// UpdateConfig updates the configuration
func (tm *TaskManager) UpdateConfig(config *models.Config) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.config = config
}

// PauseTask pauses a running task
func (tm *TaskManager) PauseTask(taskID int64) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	runner, exists := tm.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %d is not running", taskID)
	}

	// Cancel the task context
	runner.CancelFunc()
	runner.Status = "paused"

	// Update database
	if err := tm.db.UpdateScanTaskStatus(taskID, "paused"); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	return nil
}

// StopTask stops a running task
func (tm *TaskManager) StopTask(taskID int64) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	runner, exists := tm.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %d is not running", taskID)
	}

	// Cancel the task context
	runner.CancelFunc()
	runner.Status = "stopped"

	// Update database
	if err := tm.db.UpdateScanTaskStatus(taskID, "stopped"); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Remove from running tasks
	delete(tm.tasks, taskID)

	return nil
}

// GetTaskStatus returns the status of a task
func (tm *TaskManager) GetTaskStatus(taskID int64) (*models.TaskProgress, error) {
	tm.mu.RLock()
	runner, exists := tm.tasks[taskID]
	tm.mu.RUnlock()

	if !exists {
		// Task not running, get from database
		task, err := tm.db.GetScanTaskByID(taskID)
		if err != nil {
			return nil, err
		}

		percentage := 0.0
		if task.TotalRequests > 0 {
			percentage = float64(task.CompletedRequests) / float64(task.TotalRequests) * 100
		}

		return &models.TaskProgress{
			TaskID:            int(task.ID),
			TotalRequests:     task.TotalRequests,
			CompletedRequests: task.CompletedRequests,
			FoundVulns:        task.FoundVulns,
			Percentage:        percentage,
			Status:            task.Status,
		}, nil
	}

	// Task is running, get live progress
	return runner.Scanner.GetProgress(), nil
}

// GetAllTasks returns all tasks from database
func (tm *TaskManager) GetAllTasks() ([]*models.ScanTask, error) {
	return tm.db.GetAllScanTasks()
}

// GetRunningTasks returns all currently running tasks
func (tm *TaskManager) GetRunningTasks() []*models.ScanTask {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var tasks []*models.ScanTask
	for _, runner := range tm.tasks {
		tasks = append(tasks, runner.Task)
	}
	return tasks
}

// countRunningTasks counts the number of running tasks (must be called with lock)
func (tm *TaskManager) countRunningTasks() int {
	count := 0
	for _, runner := range tm.tasks {
		if runner.Status == "running" {
			count++
		}
	}
	return count
}

// DeleteTask deletes a task from database
func (tm *TaskManager) DeleteTask(taskID int64) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check if task is running
	if _, exists := tm.tasks[taskID]; exists {
		return fmt.Errorf("cannot delete running task %d, please stop it first", taskID)
	}

	// Delete from database
	return tm.db.DeleteScanTask(taskID)
}

// Helper function to convert string slice to JSON
func convertToJSON(slice []string) string {
	// Simple JSON conversion
	result := "["
	for i, s := range slice {
		if i > 0 {
			result += ","
		}
		result += `"` + s + `"`
	}
	result += "]"
	return result
}
