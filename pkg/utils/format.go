package utils

import (
	"fmt"
	"time"
)

// AddTimestamp wraps string with time
func AddTimestamp(inp string) string {
	return fmt.Sprintf("time: %s out: %s", time.Now().Format(time.RFC3339Nano), inp)
}

// AddTimestampf works like AddTimestamp, but accepts formatting arguments
func AddTimestampf(format string, args ...interface{}) string {
	argsWithTimestamp := append([]interface{}{time.Now().Format(time.RFC3339Nano)}, args...)
	format = "time: %s out: " + format
	return fmt.Sprintf(format, argsWithTimestamp...)
}
