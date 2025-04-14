package api

import (
	"log"
	"net/http"
)

// SetupRoutes 配置API路由
func SetupRoutes(handler *MetricsHandler) *http.ServeMux {
	mux := http.NewServeMux()

	// 节点上报指标
	mux.HandleFunc("/api/v1/nodes/:node_id/metrics", handler.HandleMetricsSubmit)

	// 获取节点指标
	mux.HandleFunc("/api/v1/nodes/metrics", handler.HandleGetNodeMetrics)

	// 获取所有节点列表
	mux.HandleFunc("/api/v1/nodes", handler.HandleGetAllNodes)

	// 请求日志中间件
	loggingMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}

	// 应用中间件
	return applyMiddleware(mux, loggingMiddleware)
}

// applyMiddleware 应用中间件到路由
func applyMiddleware(handler http.Handler, middlewares ...func(http.Handler) http.Handler) *http.ServeMux {
	for _, middleware := range middlewares {
		handler = middleware(handler)
	}

	// 将处理后的handler包装到一个新的ServeMux中
	newMux := http.NewServeMux()
	newMux.Handle("/", handler)
	return newMux
}
