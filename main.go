package main

import (
	"github.com/gin-gonic/gin"
	"github.com/syslens/syslens-api/monitor"
)

func main() {
	r := gin.Default()

	r.GET("/api/metrics", monitor.GetMetrics)

	r.Run(":8080") // 启动在 8080 端口
}
