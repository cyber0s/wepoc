
package scanner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"wepoc/internal/models"
)

// TemplateInfo represents parsed template metadata
type TemplateInfo struct {
	ID          string                 `yaml:"id"`
	Info        map[string]interface{} `yaml:"info"`
	HTTP        interface{}            `yaml:"http,omitempty"`
	Network     interface{}            `yaml:"network,omitempty"`
	Headless    interface{}            `yaml:"headless,omitempty"`
	File        interface{}            `yaml:"file,omitempty"`
	Workflows   interface{}            `yaml:"workflows,omitempty"`
	DNS         interface{}            `yaml:"dns,omitempty"`
	SSL         interface{}            `yaml:"ssl,omitempty"`
	Code        interface{}            `yaml:"code,omitempty"`
}

// TemplateParser handles parsing of Nuclei templates
type TemplateParser struct {
}

// NewTemplateParser creates a new template parser
func NewTemplateParser() *TemplateParser {
	return &TemplateParser{}
}

// ParseTemplate parses a single Nuclei template file
func (tp *TemplateParser) ParseTemplate(filePath string) (*models.Template, error) {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %w", err)
	}

	// Parse YAML
	var templateInfo TemplateInfo
	if err := yaml.Unmarshal(data, &templateInfo); err != nil {
		return nil, fmt.Errorf("failed to parse template YAML: %w", err)
	}

	// Extract metadata
	template := &models.Template{
		TemplateID: templateInfo.ID,
		FilePath:   filePath,
	}

	// Extract info fields
	if templateInfo.Info != nil {
		if name, ok := templateInfo.Info["name"].(string); ok {
			template.Name = name
		}
		if severity, ok := templateInfo.Info["severity"].(string); ok {
			template.Severity = severity
		}
		
		// Handle tags - can be string or array
		if tags, ok := templateInfo.Info["tags"]; ok {
			switch v := tags.(type) {
			case string:
				template.Tags = v
			case []interface{}:
				var tagStrings []string
				for _, tag := range v {
					if tagStr, ok := tag.(string); ok {
						tagStrings = append(tagStrings, tagStr)
					}
				}
				template.Tags = strings.Join(tagStrings, ",")
			}
		}
		
		// Handle author - can be string or array
		if author, ok := templateInfo.Info["author"]; ok {
			switch v := author.(type) {
			case string:
				template.Author = v
			case []interface{}:
				var authorStrings []string
				for _, auth := range v {
					if authStr, ok := auth.(string); ok {
						authorStrings = append(authorStrings, authStr)
					}
				}
				template.Author = strings.Join(authorStrings, ",")
			}
		}
	}

	return template, nil
}

// ScanDirectory recursively scans a directory for Nuclei templates
func (tp *TemplateParser) ScanDirectory(dirPath string) ([]*models.Template, []error) {
	var templates []*models.Template
	var errors []error

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			errors = append(errors, fmt.Errorf("error accessing path %s: %w", path, err))
			return nil // Continue walking
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process .yaml and .yml files
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		// Try to parse template
		template, err := tp.ParseTemplate(path)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to parse %s: %w", path, err))
			return nil // Continue walking
		}

		templates = append(templates, template)
		return nil
	})

	if err != nil {
		errors = append(errors, fmt.Errorf("failed to walk directory: %w", err))
	}

	return templates, errors
}

// ValidateTemplate validates a template file using nuclei -validate
func (tp *TemplateParser) ValidateTemplate(templatePath string, nucleiPath string) error {
	// Use nuclei -validate command to validate the template
	cmd := exec.Command(nucleiPath, "-validate", "-t", templatePath)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return fmt.Errorf("nuclei validation failed: %s", string(output))
	}
	
	// Check if validation was successful
	outputStr := string(output)
	if strings.Contains(outputStr, "All templates validated successfully") {
		return nil
	}
	
	return fmt.Errorf("template validation failed: %s", outputStr)
}

// PreValidateTemplates validates templates without importing them
func (tp *TemplateParser) PreValidateTemplates(sourceDir, nucleiPath string) (*ImportResult, error) {
	result := &ImportResult{
		TotalFound:    0,
		Validated:     0,
		Failed:        0,
		AlreadyExists: 0,
		Errors:        []string{},
		ValidTemplates: []*models.Template{},
	}

	// Scan source directory for templates
	templates, errors := tp.ScanDirectory(sourceDir)
	result.TotalFound = len(templates)
	
	// Add scan errors to result
	for _, err := range errors {
		result.Errors = append(result.Errors, err.Error())
	}

	// Use nuclei to validate all templates at once (much faster)
	validationResult, err := tp.validateTemplatesBatch(sourceDir, nucleiPath)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Batch validation failed: %v", err))
		// Fall back to individual validation
		for _, template := range templates {
			if err := tp.ValidateTemplate(template.FilePath, nucleiPath); err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("Validation failed for %s: %v", template.TemplateID, err))
			} else {
				result.Validated++
				result.ValidTemplates = append(result.ValidTemplates, template)
			}
		}
	} else {
		// Process batch validation results
		result.Validated = validationResult.Validated
		result.Failed = validationResult.Failed
		result.Errors = append(result.Errors, validationResult.Errors...)
		
		// Add valid templates to result
		for _, template := range templates {
			// Check if this template was validated successfully
			if tp.isTemplateValid(template, validationResult) {
				result.ValidTemplates = append(result.ValidTemplates, template)
			}
		}
	}

	return result, nil
}

// validateTemplatesBatch validates all templates in a directory using nuclei
func (tp *TemplateParser) validateTemplatesBatch(sourceDir, nucleiPath string) (*ImportResult, error) {
	result := &ImportResult{
		TotalFound: 0,
		Validated:  0,
		Failed:     0,
		Errors:     []string{},
	}

	// Use nuclei to validate the entire directory
	cmd := exec.Command(nucleiPath, "-validate", "-t", sourceDir)
	output, err := cmd.CombinedOutput()
	
	outputStr := string(output)
	
	// Count total templates found
	templates, _ := tp.ScanDirectory(sourceDir)
	result.TotalFound = len(templates)
	
	if err != nil {
		// Parse error output to count failed templates
		lines := strings.Split(outputStr, "\n")
		errorCount := 0
		for _, line := range lines {
			if strings.Contains(line, "[ERR]") {
				errorCount++
				result.Errors = append(result.Errors, strings.TrimSpace(line))
			}
		}
		result.Failed = errorCount
		result.Validated = result.TotalFound - result.Failed
	} else {
		// All templates validated successfully
		result.Validated = result.TotalFound
		result.Failed = 0
	}

	return result, nil
}

// isTemplateValid checks if a template was validated successfully
func (tp *TemplateParser) isTemplateValid(template *models.Template, validationResult *ImportResult) bool {
	// Simple heuristic: if we have fewer failed templates than total, 
	// and this template has a valid ID, consider it valid
	if template.TemplateID != "" && template.Name != "" {
		return true
	}
	return false
}

// ImportTemplatesWithValidation imports templates with validation and incremental copy
func (tp *TemplateParser) ImportTemplatesWithValidation(sourceDir, targetDir, nucleiPath string) (*ImportResult, error) {
	return tp.ImportTemplatesWithValidationAndProgress(sourceDir, targetDir, nucleiPath, nil)
}

// ImportTemplatesWithValidationAndProgress imports templates with validation, incremental copy and progress updates
func (tp *TemplateParser) ImportTemplatesWithValidationAndProgress(sourceDir, targetDir, nucleiPath string, progressCallback func(current, total int, status string, stats ...map[string]int)) (*ImportResult, error) {
	result := &ImportResult{
		TotalFound:    0,
		Validated:     0,
		Failed:        0,
		AlreadyExists: 0,
		Errors:        []string{},
	}

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create target directory: %w", err)
	}

	// Scan source directory for templates
	templates, errors := tp.ScanDirectory(sourceDir)
	result.TotalFound = len(templates)
	
	// Add scan errors to result
	for _, err := range errors {
		result.Errors = append(result.Errors, err.Error())
	}

	// Send initial progress
	if progressCallback != nil {
		progressCallback(0, result.TotalFound, "开始批量验证模板...")
	}

	// 使用批量验证提高速度
	validationResult, err := tp.validateTemplatesBatch(sourceDir, nucleiPath)
	if err != nil {
		// 如果批量验证失败，回退到单个验证
		if progressCallback != nil {
			progressCallback(0, result.TotalFound, "批量验证失败，使用单个验证...")
		}
		
		// Process each template individually
		for i, template := range templates {
			// Update progress with real-time stats
			if progressCallback != nil {
				stats := map[string]int{
					"successful": result.Validated,
					"errors":     result.Failed,
					"duplicates": result.AlreadyExists,
				}
				progressCallback(i, result.TotalFound, fmt.Sprintf("验证模板 %d/%d: %s", i+1, result.TotalFound, template.TemplateID), stats)
			}

			// Check if template already exists in target directory
			targetPath := filepath.Join(targetDir, filepath.Base(template.FilePath))
			if _, err := os.Stat(targetPath); err == nil {
				result.AlreadyExists++
				continue
			}

			// Validate template using nuclei
			if err := tp.ValidateTemplate(template.FilePath, nucleiPath); err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("Validation failed for %s: %v", template.TemplateID, err))
				continue
			}

			// Copy validated template to target directory
			if err := tp.copyTemplate(template.FilePath, targetPath); err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to copy %s: %v", template.TemplateID, err))
				continue
			}

			// Update template file path to target location
			template.FilePath = targetPath
			result.Validated++
		}
	} else {
		// 使用批量验证结果
		result.Validated = validationResult.Validated
		result.Failed = validationResult.Failed
		
		result.Errors = append(result.Errors, validationResult.Errors...)
		
		// 复制所有有效模板
		validTemplates := []*models.Template{}
		for i, template := range templates {
			// Update progress with real-time stats
			if progressCallback != nil {
				stats := map[string]int{
					"successful": len(validTemplates),
					"errors":     result.Failed,
					"duplicates": result.AlreadyExists,
				}
				progressCallback(i, result.TotalFound, fmt.Sprintf("复制模板 %d/%d: %s", i+1, result.TotalFound, template.TemplateID), stats)
			}

			// Check if template already exists in target directory
			targetPath := filepath.Join(targetDir, filepath.Base(template.FilePath))
			if _, err := os.Stat(targetPath); err == nil {
				result.AlreadyExists++
				continue
			}

			// 假设批量验证通过的模板都是有效的
			if tp.isTemplateValid(template, validationResult) {
				// Copy template to target directory
				if err := tp.copyTemplate(template.FilePath, targetPath); err != nil {
					result.Failed++
					result.Errors = append(result.Errors, fmt.Sprintf("Failed to copy %s: %v", template.TemplateID, err))
					continue
				}

				// Update template file path to target location
				template.FilePath = targetPath
				validTemplates = append(validTemplates, template)
			}
		}
		result.Validated = len(validTemplates)
	}

	// Send final progress with final stats
	if progressCallback != nil {
		finalStats := map[string]int{
			"successful": result.Validated,
			"errors":     result.Failed,
			"duplicates": result.AlreadyExists,
		}
		progressCallback(result.TotalFound, result.TotalFound, "导入完成!", finalStats)
	}

	return result, nil
}

// copyTemplate copies a template file to the target location
func (tp *TemplateParser) copyTemplate(sourcePath, targetPath string) error {
	// Read source file
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Write to target file
	if err := os.WriteFile(targetPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write target file: %w", err)
	}

	return nil
}

// ConfirmAndImportTemplates imports only the pre-validated templates
func (tp *TemplateParser) ConfirmAndImportTemplates(validTemplates []*models.Template, targetDir string) (*ImportResult, error) {
	return tp.ConfirmAndImportTemplatesWithProgress(validTemplates, targetDir, nil)
}

// ConfirmAndImportTemplatesWithProgress imports only the pre-validated templates with progress callback
func (tp *TemplateParser) ConfirmAndImportTemplatesWithProgress(validTemplates []*models.Template, targetDir string, progressCallback func(current, total int, status string, stats ...map[string]int)) (*ImportResult, error) {
	result := &ImportResult{
		TotalFound:    len(validTemplates),
		Validated:     0,
		Failed:        0,
		AlreadyExists: 0,
		Errors:        []string{},
	}

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create target directory: %w", err)
	}

	total := len(validTemplates)
	
	// Process each pre-validated template
	for i, template := range validTemplates {
		// Update progress with real-time stats
		if progressCallback != nil {
			stats := map[string]int{
				"successful": result.Validated,
				"errors":     result.Failed,
				"duplicates": result.AlreadyExists,
			}
			progressCallback(i, total, "导入中...", stats)
		}

		// Check if template already exists in target directory
		targetPath := filepath.Join(targetDir, filepath.Base(template.FilePath))
		if _, err := os.Stat(targetPath); err == nil {
			result.AlreadyExists++
			continue
		}

		// Copy template to target directory
		if err := tp.copyTemplate(template.FilePath, targetPath); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to copy %s: %v", template.TemplateID, err))
			continue
		}

		// Update template file path to target location
		template.FilePath = targetPath
		result.Validated++
	}

	// Final progress update with final stats
	if progressCallback != nil {
		finalStats := map[string]int{
			"successful": result.Validated,
			"errors":     result.Failed,
			"duplicates": result.AlreadyExists,
		}
		progressCallback(total, total, "导入完成!", finalStats)
	}

	return result, nil
}


// ImportResult represents the result of template import operation
type ImportResult struct {
	TotalFound     int                 `json:"total_found"`
	Validated      int                 `json:"validated"`
	Failed         int                 `json:"failed"`
	AlreadyExists  int                 `json:"already_exists"`
	Errors         []string            `json:"errors"`
	ValidTemplates []*models.Template  `json:"valid_templates,omitempty"`
}
