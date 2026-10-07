package task

import "time"

// now returns the current Unix millisecond timestamp.
func now() int64 {
	return time.Now().UnixMilli()
}

// Now returns the current Unix millisecond timestamp.
func Now() int64 {
	return now()
}