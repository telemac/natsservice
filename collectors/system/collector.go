package system

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

// Collector implements system metrics collection using gopsutil
type Collector struct{}

// CollectAllMetrics collects all available system metrics
// Returns partial metrics if some collection fails, with aggregated errors
func (c *Collector) CollectAllMetrics(ctx context.Context) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})
	var errors []string

	// CPU: 1 second sampling for accuracy
	if cpuPercent, err := cpu.PercentWithContext(ctx, 1*time.Second, false); err == nil && len(cpuPercent) > 0 {
		cores, _ := cpu.CountsWithContext(ctx, false)
		metrics["cpu"] = map[string]interface{}{
			"usage_percent": cpuPercent[0],
			"cores":         cores,
		}
	} else {
		errors = append(errors, fmt.Sprintf("cpu: %v", err))
	}

	// Memory
	if memStats, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		metrics["memory"] = map[string]interface{}{
			"used_bytes":    memStats.Used,
			"total_bytes":   memStats.Total,
			"usage_percent": memStats.UsedPercent,
		}
	} else {
		errors = append(errors, fmt.Sprintf("memory: %v", err))
	}

	// Disk: platform-specific path
	diskPath := getDiskPath()
	if diskStats, err := disk.UsageWithContext(ctx, diskPath); err == nil {
		metrics["disk"] = map[string]interface{}{
			"used_bytes":    diskStats.Used,
			"total_bytes":   diskStats.Total,
			"usage_percent": diskStats.UsedPercent,
		}
	} else {
		errors = append(errors, fmt.Sprintf("disk: %v", err))
	}

	// Uptime
	if uptime, err := host.UptimeWithContext(ctx); err == nil {
		metrics["uptime"] = map[string]interface{}{
			"seconds": uptime,
		}
	} else {
		errors = append(errors, fmt.Sprintf("uptime: %v", err))
	}

	// Return partial metrics with aggregated errors
	if len(errors) > 0 {
		return metrics, fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	return metrics, nil
}

// getDiskPath returns the appropriate disk path for the platform
func getDiskPath() string {
	if runtime.GOOS == "windows" {
		// Use system drive (typically C:)
		systemRoot := os.Getenv("SystemRoot")
		if systemRoot != "" {
			return filepath.VolumeName(systemRoot) + "\\"
		}
		return "C:\\"
	}
	return "/"
}
