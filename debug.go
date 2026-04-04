package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	debugEnabled bool
	debugOnce    sync.Once
)

func isDebugEnabled() bool { // Check if debug mode is enabled by reading the IPK_DEBUG environment variable once
	debugOnce.Do(func() {
		v := strings.TrimSpace(strings.ToLower(os.Getenv("IPK_DEBUG")))
		debugEnabled = v == "1" || v == "true" || v == "yes" || v == "on"
	})
	return debugEnabled
}

func debugf(format string, args ...any) { // Print debug messages if debug mode is enabled
	if !isDebugEnabled() {
		return
	}

	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "[DEBUG %s] %s\n", time.Now().Format("15:04:05.000"), msg)
}
