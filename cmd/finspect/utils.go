package main

import (
	"fmt"
	"os"

	"go.uber.org/zap"
)

// handleCommandError provides standardized error handling for CLI commands
func handleCommandError(cmd, operation, path string, err error) {
	if logger != nil {
		logger.Error(fmt.Sprintf("Failed to %s", operation),
			zap.String("path", path),
			zap.Error(err))
	}
	fmt.Fprintf(os.Stderr, "%s: %s: %v\n", cmd, path, err)
	os.Exit(1)
}

// handleCommandErrorWithFields provides error handling with additional fields
func handleCommandErrorWithFields(cmd, operation string, err error, fields ...zap.Field) {
	if logger != nil {
		logger.Error(fmt.Sprintf("Failed to %s", operation), append(fields, zap.Error(err))...)
	}
	fmt.Fprintf(os.Stderr, "%s: %v\n", cmd, err)
	os.Exit(1)
}

// handleNonFatalError provides standardized error handling for non-fatal errors in multi-file operations
func handleNonFatalError(cmd, path string, err error) {
	if logger != nil {
		logger.Error(fmt.Sprintf("%s failed", cmd),
			zap.String("path", path),
			zap.Error(err))
	}
	fmt.Fprintf(os.Stderr, "%s: %s: %v\n", cmd, path, err)
}

// resolvePaths resolves multiple paths using the common resolvePath function
func resolvePaths(paths ...string) []string {
	resolved := make([]string, len(paths))
	for i, p := range paths {
		resolved[i] = resolvePath(p)
	}
	return resolved
}
