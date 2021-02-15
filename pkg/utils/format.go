package utils

import (
	"fmt"
	"time"
)

// AddTimestamp wraps string with time
func AddTimestamp(inp string) string {
	return fmt.Sprintf("time: %s out: %s", time.Now().Format(time.RFC3339Nano), inp)
}
