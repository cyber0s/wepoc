package scanner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TempManager manages temporary template directories for scanning
type TempManager struct {
	baseDir string // ~/.wepoc/tmp
	logger  *EnhancedLogger
}

// NewTempManager creates a new temporary directory manager
func NewTempManager() (*TempManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}
	
	baseDir := filepath.Join(homeDir, ".wepoc", "tmp")
	
	// Create enhanced logger for temp manager
	logger, err := NewEnhancedLogger(0, "temp_manager")
	if err != nil {
		// If logger creation fails, continue without it
		fmt.Printf("⚠️  Failed to create enhanced logger for temp manager: %v\n", err)
	}
	
	tm := &TempManager{
		baseDir: baseDir,
		logger:  logger,
	}
	
	if logger != nil {
		logger.Info("TempManager initialized", map[string]interface{}{
			"base_dir": baseDir,
		})
	}
	
	return tm, nil
}

// CreateTempTemplateDir creates a temporary directory and copies selected templates
func (tm *TempManager) CreateTempTemplateDir(taskID int64, selectedPOCs []string) (string, error) {
	startTime := time.Now()
	
	// Create unique temp directory for this task
	tempDirName := fmt.Sprintf("task_%d_%d", taskID, time.Now().Unix())
	tempDir := filepath.Join(tm.baseDir, tempDirName)
	
	if tm.logger != nil {
		tm.logger.Info("Creating temporary template directory", map[string]interface{}{
			"task_id":      taskID,
			"temp_dir":     tempDir,
			"poc_count":    len(selectedPOCs),
			"selected_pocs": selectedPOCs,
		})
	}
	
	// Ensure base directory exists
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		if tm.logger != nil {
			tm.logger.Error("Failed to create temp directory", err, map[string]interface{}{
				"temp_dir": tempDir,
				"task_id":  taskID,
			})
		}
		return "", fmt.Errorf("failed to create temp directory %s: %w", tempDir, err)
	}
	
	fmt.Printf("📁 创建临时模板目录: %s\n", tempDir)
	
	// Get source templates directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		if tm.logger != nil {
			tm.logger.Error("Failed to get user home directory", err, map[string]interface{}{
				"task_id": taskID,
			})
		}
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	templatesDir := filepath.Join(homeDir, ".wepoc", "nuclei-templates")
	
	if tm.logger != nil {
		tm.logger.Debug("Source templates directory", map[string]interface{}{
			"templates_dir": templatesDir,
			"task_id":       taskID,
		})
	}
	
	// Copy selected POC files
	copiedCount := 0
	failedCount := 0
	var copyErrors []string
	
	for _, pocPath := range selectedPOCs {
		var srcPath string
		
		// Handle both absolute and relative paths
		if filepath.IsAbs(pocPath) {
			srcPath = pocPath
		} else {
			srcPath = filepath.Join(templatesDir, pocPath)
		}
		
		// Create destination path (preserve directory structure)
		relPath := pocPath
		if filepath.IsAbs(pocPath) {
			// For absolute paths, try to make them relative to templates directory
			if strings.HasPrefix(pocPath, templatesDir) {
				relPath = strings.TrimPrefix(pocPath, templatesDir)
				relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
			} else {
				// Use just the filename for absolute paths outside templates dir
				relPath = filepath.Base(pocPath)
			}
		}
		
		dstPath := filepath.Join(tempDir, relPath)
		
		// Create destination directory if needed
		dstDir := filepath.Dir(dstPath)
		if err := os.MkdirAll(dstDir, 0755); err != nil {
			errorMsg := fmt.Sprintf("failed to create directory %s: %v", dstDir, err)
			copyErrors = append(copyErrors, errorMsg)
			failedCount++
			
			if tm.logger != nil {
				tm.logger.Error("Failed to create destination directory", err, map[string]interface{}{
					"dst_dir":  dstDir,
					"poc_path": pocPath,
					"task_id":  taskID,
				})
			}
			continue
		}
		
		// Copy the file
		if err := tm.copyFile(srcPath, dstPath); err != nil {
			errorMsg := fmt.Sprintf("failed to copy %s: %v", pocPath, err)
			copyErrors = append(copyErrors, errorMsg)
			failedCount++
			
			if tm.logger != nil {
				tm.logger.Error("Failed to copy POC file", err, map[string]interface{}{
					"src_path": srcPath,
					"dst_path": dstPath,
					"poc_path": pocPath,
					"task_id":  taskID,
				})
			}
		} else {
			copiedCount++
			if tm.logger != nil {
				tm.logger.Debug("Successfully copied POC file", map[string]interface{}{
					"src_path": srcPath,
					"dst_path": dstPath,
					"poc_path": pocPath,
					"task_id":  taskID,
				})
			}
		}
	}
	
	duration := time.Since(startTime)
	
	if tm.logger != nil {
		tm.logger.Info("Temporary template directory creation completed", map[string]interface{}{
			"task_id":       taskID,
			"temp_dir":      tempDir,
			"total_pocs":    len(selectedPOCs),
			"copied_count":  copiedCount,
			"failed_count":  failedCount,
			"duration_ms":   duration.Milliseconds(),
			"copy_errors":   copyErrors,
		})
	}
	
	fmt.Printf("📋 复制完成: %d/%d 个POC文件 (耗时: %v)\n", copiedCount, len(selectedPOCs), duration)
	
	if failedCount > 0 {
		fmt.Printf("⚠️  %d 个文件复制失败\n", failedCount)
		for _, errMsg := range copyErrors {
			fmt.Printf("   - %s\n", errMsg)
		}
	}
	
	return tempDir, nil
}

// copyFile copies a file from source to destination
func (tm *TempManager) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer sourceFile.Close()
	
	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer destFile.Close()
	
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}
	
	return nil
}

// CleanupTempDir removes the temporary directory
func (tm *TempManager) CleanupTempDir(tempDir string) error {
	startTime := time.Now()
	
	if tm.logger != nil {
		tm.logger.Info("Starting cleanup of temporary directory", map[string]interface{}{
			"temp_dir": tempDir,
		})
	}
	
	// Check if directory exists
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		if tm.logger != nil {
			tm.logger.Warn("Temporary directory does not exist, skipping cleanup", map[string]interface{}{
				"temp_dir": tempDir,
			})
		}
		fmt.Printf("📁 临时目录不存在，跳过清理: %s\n", tempDir)
		return nil
	}
	
	// Get directory size before cleanup for logging
	var dirSize int64
	var fileCount int
	if tm.logger != nil {
		filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				dirSize += info.Size()
				fileCount++
			}
			return nil
		})
	}
	
	// Remove the directory
	if err := os.RemoveAll(tempDir); err != nil {
		if tm.logger != nil {
			tm.logger.Error("Failed to cleanup temporary directory", err, map[string]interface{}{
				"temp_dir":   tempDir,
				"file_count": fileCount,
				"dir_size":   dirSize,
			})
		}
		return fmt.Errorf("failed to cleanup temp directory %s: %w", tempDir, err)
	}
	
	duration := time.Since(startTime)
	
	if tm.logger != nil {
		tm.logger.Info("Successfully cleaned up temporary directory", map[string]interface{}{
			"temp_dir":    tempDir,
			"file_count":  fileCount,
			"dir_size":    dirSize,
			"duration_ms": duration.Milliseconds(),
		})
	}
	
	fmt.Printf("🗑️  已清理临时目录: %s (文件数: %d, 耗时: %v)\n", tempDir, fileCount, duration)
	return nil
}

// CleanupOldTempDirs removes old temporary directories (older than 24 hours)
func (tm *TempManager) CleanupOldTempDirs() error {
	if _, err := os.Stat(tm.baseDir); os.IsNotExist(err) {
		return nil // Base directory doesn't exist, nothing to clean
	}
	
	entries, err := os.ReadDir(tm.baseDir)
	if err != nil {
		return fmt.Errorf("failed to read temp base directory: %w", err)
	}
	
	cutoff := time.Now().Add(-24 * time.Hour)
	cleanedCount := 0
	
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		dirPath := filepath.Join(tm.baseDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}
		
		// Remove directories older than 24 hours
		if info.ModTime().Before(cutoff) {
			if err := os.RemoveAll(dirPath); err != nil {
				fmt.Printf("⚠️  清理旧临时目录失败: %s, 错误: %v\n", dirPath, err)
			} else {
				cleanedCount++
			}
		}
	}
	
	if cleanedCount > 0 {
		fmt.Printf("🧹 清理了 %d 个旧的临时目录\n", cleanedCount)
	}
	
	return nil
}

// GetTempBaseDir returns the base temporary directory path
func (tm *TempManager) GetTempBaseDir() string {
	return tm.baseDir
}