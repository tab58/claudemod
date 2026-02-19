package app

import (
	"errors"
	"fmt"
	"os"
)

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true // File or directory exists
	}
	// Check if the error is specifically due to the file not existing
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	// For other errors (e.g., permission issues), it's unknown or a different problem
	return false
}

func dirExists(path string) bool {
	// check if path exists and is a directory
	info, err := os.Stat(path)
	if err == nil {
		return info.IsDir()
	}

	// if the directory does not exist, return false
	if errors.Is(err, os.ErrNotExist) {
		return false
	}

	fmt.Printf("error checking if directory exists: %v\n", err)
	return false
}

func copyFile(src, dst string) error {
	srcData, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	err = os.WriteFile(dst, srcData, 0644)
	if err != nil {
		return err
	}
	return nil
}
