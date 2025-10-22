package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"wepoc/internal/models"
)

const (
	DefaultPOCDir        = "nuclei-templates"
	DefaultResultsDir    = "results"
	DefaultDatabaseName  = "wepoc.db"
	DefaultMaxConcurrency = 3
	DefaultTimeout       = 10
)

// GetUserHomeDir returns the user's home directory
func GetUserHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return home, nil
}

// GetWepocDir returns the wepoc application directory
func GetWepocDir() (string, error) {
	home, err := GetUserHomeDir()
	if err != nil {
		return "", err
	}
	wepocDir := filepath.Join(home, ".wepoc")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(wepocDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create wepoc directory: %w", err)
	}

	return wepocDir, nil
}

// GetDefaultConfig returns the default configuration
func GetDefaultConfig() (*models.Config, error) {
	wepocDir, err := GetWepocDir()
	if err != nil {
		return nil, err
	}

	nucleiPath, err := findNucleiBinary()
	if err != nil {
		nucleiPath = "nuclei" // Fallback to PATH
	}

	return &models.Config{
		POCDirectory:   filepath.Join(wepocDir, DefaultPOCDir),
		ResultsDir:     filepath.Join(wepocDir, DefaultResultsDir),
		DatabasePath:   filepath.Join(wepocDir, DefaultDatabaseName),
		NucleiPath:     nucleiPath,
		MaxConcurrency: DefaultMaxConcurrency,
		Timeout:        DefaultTimeout,
	}, nil
}

// ValidateNucleiPath validates and potentially updates the nuclei path
func ValidateNucleiPath(config *models.Config) error {
	// Check if current path is valid
	if isNucleiPathValid(config.NucleiPath) {
		return nil
	}
	
	// Try to find a valid nuclei binary
	newPath, err := findNucleiBinary()
	if err != nil {
		return fmt.Errorf("nuclei not found: %w. Please install nuclei or set the correct path in settings", err)
	}
	
	// Update the config with the found path
	config.NucleiPath = newPath
	return nil
}

// isNucleiPathValid checks if the nuclei path is valid and executable
func isNucleiPathValid(path string) bool {
	if path == "" {
		return false
	}
	
	// Check if file exists and is executable
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	
	return isExecutable(path)
}

// LoadConfig loads configuration from file
func LoadConfig() (*models.Config, error) {
	wepocDir, err := GetWepocDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(wepocDir, "config.json")

	// If config file doesn't exist, create default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config, err := GetDefaultConfig()
		if err != nil {
			return nil, err
		}
		if err := SaveConfig(config); err != nil {
			return nil, err
		}
		return config, nil
	}

	// Read existing config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config models.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *models.Config) error {
	wepocDir, err := GetWepocDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(wepocDir, "config.json")

	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// EnsureDirectories ensures all required directories exist
func EnsureDirectories(config *models.Config) error {
	dirs := []string{
		config.POCDirectory,
		config.ResultsDir,
		filepath.Dir(config.DatabasePath),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// findNucleiBinary tries to find the nuclei binary
func findNucleiBinary() (string, error) {
	// Try common locations
	locations := []string{
		"nuclei",                      // In PATH
		"/usr/local/bin/nuclei",       // Common Linux/Mac location
		"/usr/bin/nuclei",             // Common Linux location
		filepath.Join(os.Getenv("HOME"), "go", "bin", "nuclei"), // Go bin
		filepath.Join(os.Getenv("HOME"), ".local", "bin", "nuclei"), // User local bin
		filepath.Join(os.Getenv("HOME"), "bin", "nuclei"), // User bin
	}

	for _, path := range locations {
		if _, err := os.Stat(path); err == nil {
			// Verify it's executable
			if isExecutable(path) {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("nuclei binary not found in common locations")
}

// isExecutable checks if a file is executable
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	
	// Check if it's a regular file
	if info.IsDir() {
		return false
	}
	
	// On Windows, check for .exe extension or executable permissions
	if runtime.GOOS == "windows" {
		ext := strings.ToLower(filepath.Ext(path))
		return ext == ".exe" || (info.Mode()&0111) != 0
	}
	
	// On Unix-like systems, check executable permissions
	return (info.Mode()&0111) != 0
}

// CheckNucleiInstalled checks if nuclei is installed and returns version
func CheckNucleiInstalled(nucleiPath string) (bool, string, error) {
	// Check if file exists
	if _, err := os.Stat(nucleiPath); os.IsNotExist(err) {
		return false, "", fmt.Errorf("nuclei not found at: %s", nucleiPath)
	}
	
	// Check if it's executable
	if !isExecutable(nucleiPath) {
		return false, "", fmt.Errorf("nuclei at %s is not executable", nucleiPath)
	}
	
	// Try to get version by running nuclei --version
	version, err := getNucleiVersion(nucleiPath)
	if err != nil {
		return false, "", fmt.Errorf("nuclei at %s is not working: %w", nucleiPath, err)
	}
	
	return true, version, nil
}

// getNucleiVersion gets the version of nuclei by running it
func getNucleiVersion(nucleiPath string) (string, error) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	
	// Normalize the path
	normalizedPath := filepath.Clean(nucleiPath)
	
	// Get absolute path
	absPath, err := filepath.Abs(normalizedPath)
	if err != nil {
		absPath = normalizedPath
	}
	
	// Try to run nuclei --version
	cmd := exec.CommandContext(ctx, absPath, "--version")
	
	// Set working directory to avoid path issues
	cmd.Dir = filepath.Dir(absPath)
	
	// Set environment variables for Windows
	if runtime.GOOS == "windows" {
		cmd.Env = append(os.Environ(), "PATH="+filepath.Dir(absPath)+";"+os.Getenv("PATH"))
		// Hide the command window on Windows
		hideWindowOnWindows(cmd)
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute nuclei at %s: %w (output: %s)", absPath, err, string(output))
	}
	
	// Parse version from output
	outputStr := strings.TrimSpace(string(output))
	lines := strings.Split(outputStr, "\n")
	
	// Look for version line that contains "Version:"
	var version string
	for _, line := range lines {
		if strings.Contains(line, "Version:") {
			// Extract version from line like "[INF] Nuclei Engine Version: v3.4.10"
			parts := strings.Split(line, "Version:")
			if len(parts) > 1 {
				version = strings.TrimSpace(parts[1])
				break
			}
		}
	}
	
	if version == "" {
		// If no version found in structured output, try to extract from first line
		if len(lines) > 0 {
			firstLine := strings.TrimSpace(lines[0])
			// Look for version pattern like "v3.4.10"
			if strings.Contains(firstLine, "v") {
				version = firstLine
			}
		}
	}
	
	if version == "" {
		version = "unknown"
	}
	
	return version, nil
}

// ValidateUserNucleiPath validates a user-specified nuclei path
func ValidateUserNucleiPath(userPath string) (bool, string, error) {
	if userPath == "" {
		return false, "", fmt.Errorf("nuclei path cannot be empty")
	}
	
	// Normalize the path
	normalizedPath := filepath.Clean(userPath)
	
	// Get absolute path
	absPath, err := filepath.Abs(normalizedPath)
	if err != nil {
		absPath = normalizedPath
	}
	
	// Check if path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return false, "", fmt.Errorf("nuclei not found at: %s", absPath)
	}
	
	// Check if it's executable
	if !isExecutable(absPath) {
		return false, "", fmt.Errorf("nuclei at %s is not executable", absPath)
	}
	
	// Try to run nuclei to verify it works
	version, err := testNucleiExecution(absPath)
	if err != nil {
		return false, "", fmt.Errorf("nuclei at %s is not working: %w", absPath, err)
	}
	
	return true, version, nil
}

// testNucleiExecution tests if nuclei can be executed and returns version
func testNucleiExecution(nucleiPath string) (string, error) {
	// Normalize path for Windows
	normalizedPath := filepath.Clean(nucleiPath)
	
	// Get absolute path
	absPath, err := filepath.Abs(normalizedPath)
	if err != nil {
		absPath = normalizedPath
	}
	
	// Check if path exists and is accessible
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return "", fmt.Errorf("nuclei executable not found at: %s", absPath)
	}
	
	return getNucleiVersion(absPath)
}

// hideWindowOnWindows is a placeholder function that will be implemented by build tags
func hideWindowOnWindows(cmd *exec.Cmd) {
	// This function is implemented in windows_hide.go and unix_hide.go
}
