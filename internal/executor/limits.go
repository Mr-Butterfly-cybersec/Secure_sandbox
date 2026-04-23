package executor

import "time"

// Limits defines the resource constraints for a sandbox container.
type Limits struct {
	MemoryUsageBytes int64         // Maximum memory allowed (e.g., 50MB = 50 * 1024 * 1024)
	NanoCPUs         int64         // CPU limit in nanos (e.g., 0.5 CPU = 500000000)
	PIDsLimit        int64         // Maximum number of processes (prevent fork bombs)
	Timeout          time.Duration // Maximum execution time
}

// DefaultLimits returns a safe set of default constraints.
func DefaultLimits() Limits {
	return Limits{
		MemoryUsageBytes: 64 * 1024 * 1024, // 64 MB
		NanoCPUs:         500000000,        // 0.5 CPU
		PIDsLimit:        20,               // 20 processes
		Timeout:          5 * time.Second,  // 5 seconds
	}
}
