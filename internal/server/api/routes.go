package api

import (
	"log"
	"net/http"
	"strings"
	"time"
)

// SetupRoutes 配置API路由
func SetupRoutes(handler *MetricsHandler) *http.ServeMux {
	mux := http.NewServeMux()

	// 节点上报指标 - 使用通配符路径
	mux.HandleFunc("/api/v1/nodes/", func(w http.ResponseWriter, r *http.Request) {
		// 解析URL路径，提取nodeID
		path := r.URL.Path

		// 检查路径是否符合 /api/v1/nodes/{nodeID}/metrics 格式
		if strings.HasSuffix(path, "/metrics") {
			parts := strings.Split(path, "/")
			if len(parts) == 5 && parts[1] == "api" && parts[2] == "v1" && parts[3] == "nodes" {
				nodeID := parts[4]
				log.Printf("[路由] 匹配到节点指标上报请求 - 路径: %s, 节点ID: %s, 方法: %s", path, nodeID, r.Method)

				// 保存nodeID到请求上下文，以便在处理函数中使用
				r.Header.Set("X-Node-ID", nodeID)
				// 调用指标处理函数
				handler.HandleMetricsSubmit(w, r)
				return
			}
		}

		// 如果路径不匹配指标上报格式，则尝试其他API
		if r.URL.Path == "/api/v1/nodes/metrics" {
			log.Printf("[路由] 匹配到查询节点指标请求 - 方法: %s, 查询参数: %s", r.Method, r.URL.RawQuery)
			// 获取节点指标
			handler.HandleGetNodeMetrics(w, r)
		} else if r.URL.Path == "/api/v1/nodes" {
			log.Printf("[路由] 匹配到获取所有节点请求 - 方法: %s", r.Method)
			// 获取所有节点列表
			handler.HandleGetAllNodes(w, r)
		} else {
			log.Printf("[路由] 未匹配任何API路径 - 路径: %s, 方法: %s", r.URL.Path, r.Method)
			// 不匹配任何API路径
			http.NotFound(w, r)
		}
	})

	// 健康检查路由
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// 返回简单的健康状态
		log.Printf("[路由] 处理健康检查请求 - 来源: %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","time":"` + time.Now().Format(time.RFC3339) + `"}`))
	})

	// 请求日志中间件
	loggingMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// 记录请求开始
			log.Printf("[HTTP请求] 开始 - %s %s %s, 来源: %s, 用户代理: %s",
				r.Method, r.URL.Path, r.Proto, r.RemoteAddr, r.UserAgent())

			// 创建自定义响应写入器以捕获状态码
			writer := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // 默认为200
			}

			// 处理请求
			next.ServeHTTP(writer, r)

			// 计算处理时间
			duration := time.Since(start)

			// 获取查询参数（去除敏感信息）
			query := r.URL.RawQuery
			if query != "" {
				query = "?" + query
			}

			// 判断请求是否成功
			statusText := "成功"
			if writer.statusCode >= 400 {
				statusText = "失败"
			}

			// 记录请求完成
			log.Printf("[HTTP请求] 完成 - %s %s %s, 状态: %d, 耗时: %v, 结果: %s",
				r.Method, r.URL.Path, query, writer.statusCode, duration, statusText)

			// 记录慢请求
			if duration > 500*time.Millisecond {
				log.Printf("[警告] 慢请求 - %s %s, 耗时: %v", r.Method, r.URL.Path, duration)
			}
		})
	}

	// 应用中间件
	return applyMiddleware(mux, loggingMiddleware)
}

// 自定义响应写入器以捕获状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// 重写WriteHeader方法以捕获状态码
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// 重写Write方法以确保状态码被设置
func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
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
