package main

import (
	"fmt"
	"os"
	"strings"
)

// envOrFile returns the value of key, or the contents of $key_FILE when the
// direct env var is empty. Empty string and nil error means neither is set.
// Docker Compose secrets commonly use the *_FILE convention.
func envOrFile(key string) (string, error) {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v, nil
	}
	path := strings.TrimSpace(os.Getenv(key + "_FILE"))
	if path == "" {
		return "", nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%s_FILE: %w", key, err)
	}
	v := strings.TrimSpace(string(b))
	if v == "" {
		return "", fmt.Errorf("%s_FILE %q is empty", key, path)
	}
	return v, nil
}

// accessSecretFromEnv resolves the HTTP API/UI shared secret from environment.
// Checked in order: KEIMLING_ACCESS_SECRET, KEIMLING_SECRET, ACCESS_SECRET
// (each with optional _FILE companion).
func accessSecretFromEnv() (string, error) {
	for _, key := range []string{"KEIMLING_ACCESS_SECRET", "KEIMLING_SECRET", "ACCESS_SECRET"} {
		v, err := envOrFile(key)
		if err != nil {
			return "", err
		}
		if v != "" {
			return v, nil
		}
	}
	return "", nil
}
