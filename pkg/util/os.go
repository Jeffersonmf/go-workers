package util

import (
	"runtime"

	"github.com/google/uuid"
)

// NewUUID returns a new random (v4) UUID. It replaced a previous
// implementation that shelled out to the `uuidgen` CLI: that only
// works on hosts where the binary happens to be installed (notably
// absent from minimal container images), spawns a process for
// something the standard library ecosystem already does in-process,
// and silently returned an empty string on error instead of failing
// loudly.
func NewUUID() string {
	return uuid.NewString()
}

// LogMemStats logs a snapshot of the current Go runtime memory
// statistics, useful for spot-checking a worker's memory footprint
// during development.
func LogMemStats() {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	Sugar.Infow("memory stats",
		"allocBytes", mem.Alloc,
		"totalAllocBytes", mem.TotalAlloc,
		"heapAllocBytes", mem.HeapAlloc,
		"numGC", mem.NumGC,
	)
}
