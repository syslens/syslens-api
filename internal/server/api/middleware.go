package api

import (
	"net/http"
	"strings"

	"github.com/syslens/syslens-api/internal/config"
)

// AuthMiddleware 创建一个认证中间件，用于验证聚合服务器的请求
func AuthMiddleware(cfg *config.ServerConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 检查是否是聚合服务器的请求路径
			if isAggregatorPath(r.URL.Path) {
				// 从请求头中获取认证令牌
				authToken := r.Header.Get("Authorization")
				if authToken == "" {
					http.Error(w, "未提供认证令牌", http.StatusUnauthorized)
					return
				}

				// 移除可能的 "Bearer " 前缀
				authToken = strings.TrimPrefix(authToken, "Bearer ")

				// 验证令牌
				if authToken != cfg.Aggregator.AuthToken {
					http.Error(w, "无效的认证令牌", http.StatusUnauthorized)
					return
				}
			}

			// 继续处理请求
			next.ServeHTTP(w, r)
		})
	}
}

// isAggregatorPath 检查请求路径是否需要聚合服务器认证
func isAggregatorPath(path string) bool {
	// 这里定义需要认证的路径前缀
	aggregatorPaths := []string{
		"/api/v1/metrics",
		"/api/v1/nodes/metrics",
	}

	for _, prefix := range aggregatorPaths {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
