package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"
)

func editFile(path, oldString, newString string, replaceAll bool) (string, error) {
	if privatePath(path) {
		return "", fmt.Errorf("access denied: protected path")
	}
	if !insideWorkingDir(path) {
		return "", fmt.Errorf("write outside working directory")
	}
	if oldString == "" {
		return "", fmt.Errorf("old_string must not be empty; use write_file to create or overwrite")
	}
	if oldString == newString {
		return "", fmt.Errorf("old_string and new_string must differ")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(data) > 8<<20 {
		return "", fmt.Errorf("file too large")
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("file is not valid UTF-8")
	}
	needle := []byte(oldString)
	count := bytes.Count(data, needle)
	if count == 0 {
		return "", fmt.Errorf("old_string not found; read the file again")
	}
	if !replaceAll && count > 1 {
		return "", fmt.Errorf("multiple matches (%d); provide more context in old_string or set replace_all=true", count)
	}
	if len(newString) > 1<<20 {
		return "", fmt.Errorf("new_string too large")
	}
	var next []byte
	if replaceAll {
		next = bytes.ReplaceAll(data, needle, []byte(newString))
	} else {
		next = bytes.Replace(data, needle, []byte(newString), 1)
	}
	if err := atomicWrite(path, next); err != nil {
		return "", err
	}
	if replaceAll {
		return fmt.Sprintf("edited successfully (%d replacement(s))", count), nil
	}
	return "edited successfully", nil
}

func writeFile(path, content string) (string, error) {
	if privatePath(path) {
		return "", fmt.Errorf("access denied: protected path")
	}
	if !insideWorkingDir(path) {
		return "", fmt.Errorf("write outside working directory")
	}
	if len(content) > 1<<20 {
		return "", fmt.Errorf("content too large")
	}
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
	}
	if err := atomicWrite(path, []byte(content)); err != nil {
		return "", err
	}
	return "wrote successfully", nil
}
