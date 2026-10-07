package plan

import "time"

// now returns the current Unix millisecond timestamp, matching the session package convention.
func now() int64 {
	return time.Now().UnixMilli()
}

// Now returns the current Unix millisecond timestamp.
func Now() int64 {
	return now()
}