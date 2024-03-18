package util

import (
	"context"
	"os/exec"
	"runtime"
	"time"
)

func UUIDGenerate() string {
	newUUID, err := exec.Command("uuidgen").Output()
	if err != nil {
		Sugar.Infof(err.Error())
	}

	return string(newUUID)
}

// any potentially blocking task should take a context
// style: context should be the first passed in parameter
func PipesTask(ctx context.Context, poll time.Duration) {
	Sugar.Infof("Running...")
	select {
	case <-ctx.Done():
		Sugar.Infof("Done...")
		break
	default:
		for {
			time.Sleep(poll * time.Millisecond)
		}
	}
}

func PrintStats(mem runtime.MemStats) {
	runtime.ReadMemStats(&mem)
	Sugar.Infof("mem.Alloc:", mem.Alloc)
	Sugar.Infof("mem.TotalAlloc:", mem.TotalAlloc)
	Sugar.Infof("mem.HeapAlloc:", mem.HeapAlloc)
	Sugar.Infof("mem.NumGC:", mem.NumGC)
	Sugar.Infof("-----")
}
