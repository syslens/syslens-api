package monitor

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

func GetMetrics(c *gin.Context) {
	cpuPercent, _ := cpu.Percent(0, false)
	vmStat, _ := mem.VirtualMemory()
	diskStat, _ := disk.Usage("/")

	c.JSON(http.StatusOK, gin.H{
		"cpu":    cpuPercent[0],
		"memory": vmStat.UsedPercent,
		"disk":   diskStat.UsedPercent,
	})
}
