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

func isDebugEnabled() bool {
	debugOnce.Do(func() {
		v := strings.TrimSpace(strings.ToLower(os.Getenv("IPK_DEBUG")))
		debugEnabled = v == "1" || v == "true" || v == "yes" || v == "on"
	})
	return debugEnabled
}

func debugf(format string, args ...any) {
	if !isDebugEnabled() {
		return
	}

	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "[DEBUG %s] %s\n", time.Now().Format("15:04:05.000"), msg)
}
