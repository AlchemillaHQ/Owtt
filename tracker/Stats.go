package tracker

import (
	"runtime"

	"github.com/gofiber/fiber/v2"
)

func StatsHandler(c *fiber.Ctx, wt *WebtorrentTracker) error {

	peersCount := 0

	swarms := wt.Tracker.GetSwarms()
	torrentsCount := len(swarms)

	for _, swarm := range swarms {
		peersCount += swarm.GetPeersCount()
	}

	serverStats := map[string]interface{}{
		"host":           wt.Host,
		"port":           wt.Port,
		"websocketCount": wt.WebSocketCount,
	}

	memory := GetMemoryUsage()

	responseData := map[string]interface{}{
		"torrentsCount": torrentsCount,
		"peersCount":    peersCount,
		"serverStats":   serverStats,
		"memory":        memory,
	}

	c.Set("Content-Type", "application/json")
	return c.JSON(responseData)
}

func GetMemoryUsage() uint64 {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return memStats.Alloc
}
