package database

import (
	"database/sql"
	"fmt"
	"time"

	"wepoc/internal/models"
)

// InsertScanTask inserts a new scan task into the database
func (d *Database) InsertScanTask(task *models.ScanTask) error {
	query := `
		INSERT INTO scan_tasks (name, status, pocs, targets, total_requests, completed_requests, found_vulns, output_file, start_time)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := d.db.Exec(query,
		task.Name,
		task.Status,
		task.POCs,
		task.Targets,
		task.TotalRequests,
		task.CompletedRequests,
		task.FoundVulns,
		task.OutputFile,
		task.StartTime,
	)
	if err != nil {
		return fmt.Errorf("failed to insert scan task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	task.ID = id
	return nil
}

// GetScanTaskByID retrieves a scan task by its ID
func (d *Database) GetScanTaskByID(id int64) (*models.ScanTask, error) {
	query := `
		SELECT id, name, status, pocs, targets, total_requests, completed_requests, found_vulns, output_file, start_time, end_time, created_at
		FROM scan_tasks
		WHERE id = ?
	`
	task := &models.ScanTask{}
	err := d.db.QueryRow(query, id).Scan(
		&task.ID,
		&task.Name,
		&task.Status,
		&task.POCs,
		&task.Targets,
		&task.TotalRequests,
		&task.CompletedRequests,
		&task.FoundVulns,
		&task.OutputFile,
		&task.StartTime,
		&task.EndTime,
		&task.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("scan task not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get scan task: %w", err)
	}
	return task, nil
}

// GetAllScanTasks retrieves all scan tasks from the database
func (d *Database) GetAllScanTasks() ([]*models.ScanTask, error) {
	query := `
		SELECT id, name, status, pocs, targets, total_requests, completed_requests, found_vulns, output_file, start_time, end_time, created_at
		FROM scan_tasks
		ORDER BY created_at DESC
	`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query scan tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.ScanTask
	for rows.Next() {
		task := &models.ScanTask{}
		err := rows.Scan(
			&task.ID,
			&task.Name,
			&task.Status,
			&task.POCs,
			&task.Targets,
			&task.TotalRequests,
			&task.CompletedRequests,
			&task.FoundVulns,
			&task.OutputFile,
			&task.StartTime,
			&task.EndTime,
			&task.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// UpdateScanTaskStatus updates the status of a scan task
func (d *Database) UpdateScanTaskStatus(id int64, status string) error {
	query := "UPDATE scan_tasks SET status = ? WHERE id = ?"
	result, err := d.db.Exec(query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update scan task status: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("scan task not found")
	}
	return nil
}

// UpdateScanTaskProgress updates the progress of a scan task
func (d *Database) UpdateScanTaskProgress(id int64, totalRequests, completedRequests, foundVulns int) error {
	query := `
		UPDATE scan_tasks
		SET total_requests = ?, completed_requests = ?, found_vulns = ?
		WHERE id = ?
	`
	result, err := d.db.Exec(query, totalRequests, completedRequests, foundVulns, id)
	if err != nil {
		return fmt.Errorf("failed to update scan task progress: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("scan task not found")
	}
	return nil
}

// CompleteScanTask marks a scan task as completed
func (d *Database) CompleteScanTask(id int64) error {
	query := `
		UPDATE scan_tasks
		SET status = 'completed', end_time = ?
		WHERE id = ?
	`
	result, err := d.db.Exec(query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to complete scan task: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("scan task not found")
	}
	return nil
}

// DeleteScanTask deletes a scan task by ID
func (d *Database) DeleteScanTask(id int64) error {
	query := "DELETE FROM scan_tasks WHERE id = ?"
	result, err := d.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete scan task: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("scan task not found")
	}
	return nil
}

// GetRunningScanTasks retrieves all running scan tasks
func (d *Database) GetRunningScanTasks() ([]*models.ScanTask, error) {
	query := `
		SELECT id, name, status, pocs, targets, total_requests, completed_requests, found_vulns, output_file, start_time, end_time, created_at
		FROM scan_tasks
		WHERE status = 'running'
		ORDER BY start_time ASC
	`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query running scan tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.ScanTask
	for rows.Next() {
		task := &models.ScanTask{}
		err := rows.Scan(
			&task.ID,
			&task.Name,
			&task.Status,
			&task.POCs,
			&task.Targets,
			&task.TotalRequests,
			&task.CompletedRequests,
			&task.FoundVulns,
			&task.OutputFile,
			&task.StartTime,
			&task.EndTime,
			&task.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}
