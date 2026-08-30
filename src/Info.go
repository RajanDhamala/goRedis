package src

import (
	"fmt"
	"runtime"
)

func formatBytes(b uint64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case b >= GB:
		return fmt.Sprintf("%.2f GB", float64(b)/GB)
	case b >= MB:
		return fmt.Sprintf("%.2f MB", float64(b)/MB)
	case b >= KB:
		return fmt.Sprintf("%.2f KB", float64(b)/KB)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func GetRuntimeInfo(msg []string, client *Client) (string, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	info := fmt.Sprintf(
		`Runtime Info
	Goroutines: %d
	CPUs: %d
	GOMAXPROCS: %d
	Go Version: %s
	GOOS: %s
	GOARCH: %s

	Memory
	Alloc: %s
	TotalAlloc: %s
	Sys: %s
	HeapAlloc: %s
	HeapSys: %s
	HeapIdle: %s
	HeapInuse: %s
	HeapReleased: %s
	HeapObjects: %d

	GC
	NumGC: %d
	PauseTotal: %.2f ms
	LastGC: %d
	NextGC: %s
`,
		runtime.NumGoroutine(),
		runtime.NumCPU(),
		runtime.GOMAXPROCS(0),
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,

		formatBytes(m.Alloc),
		formatBytes(m.TotalAlloc),
		formatBytes(m.Sys),
		formatBytes(m.HeapAlloc),
		formatBytes(m.HeapSys),
		formatBytes(m.HeapIdle),
		formatBytes(m.HeapInuse),
		formatBytes(m.HeapReleased),
		m.HeapObjects,

		m.NumGC,
		float64(m.PauseTotalNs)/1_000_000,
		m.LastGC,
		formatBytes(m.NextGC),
	)

	fmt.Println(info)

	return info, nil
}
