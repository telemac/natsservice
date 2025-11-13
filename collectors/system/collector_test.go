package system

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollectAllMetrics(t *testing.T) {
	assert := assert.New(t)

	collector := &Collector{}
	metrics, _ := collector.CollectAllMetrics(context.Background())

	// Should collect at least some metrics (may not get all on every platform)
	assert.NotNil(metrics)

	// Check for expected metric keys
	if cpu, ok := metrics["cpu"]; ok {
		cpuMap := cpu.(map[string]interface{})
		assert.Contains(cpuMap, "usage_percent")
		assert.Contains(cpuMap, "cores")
	}

	if memory, ok := metrics["memory"]; ok {
		memMap := memory.(map[string]interface{})
		assert.Contains(memMap, "used_bytes")
		assert.Contains(memMap, "total_bytes")
		assert.Contains(memMap, "usage_percent")
	}

	if disk, ok := metrics["disk"]; ok {
		diskMap := disk.(map[string]interface{})
		assert.Contains(diskMap, "used_bytes")
		assert.Contains(diskMap, "total_bytes")
		assert.Contains(diskMap, "usage_percent")
	}

	if uptime, ok := metrics["uptime"]; ok {
		uptimeMap := uptime.(map[string]interface{})
		assert.Contains(uptimeMap, "seconds")
	}
}

func TestGetDiskPath(t *testing.T) {
	assert := assert.New(t)

	path := getDiskPath()

	// Should return a non-empty path
	assert.NotEmpty(path)
}
