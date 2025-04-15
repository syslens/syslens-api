package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

var (
	// ErrMissingToken 表示缺少Token
	ErrMissingToken = errors.New("缺少访问令牌")
	// ErrInvalidToken 表示Token无效
	ErrInvalidToken = errors.New("无效的访问令牌")
	// ErrExpiredToken 表示Token已过期
	ErrExpiredToken = errors.New("访问令牌已过期")
)

// TokenClaims 是JWT令牌的声明结构
type TokenClaims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// AuthConfig 是身份验证中间件的配置
type AuthConfig struct {
	Secret            string        // JWT密钥
	TokenExpiry       time.Duration // 令牌过期时间
	SkipPaths         []string      // 不需要认证的路径
	ExcludeAPIKeyAuth []string      // 排除API密钥认证的路径
}

// JWTAuth 是JWT身份验证中间件
// 从请求头中获取并验证JWT令牌
func JWTAuth(config AuthConfig, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查当前路径是否在跳过认证的路径列表中
		for _, path := range config.SkipPaths {
			if strings.HasPrefix(c.Request.URL.Path, path) {
				c.Next()
				return
			}
		}

		// 从Authorization标头获取令牌
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Debug("请求缺少Authorization标头")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   ErrMissingToken.Error(),
				"code":    http.StatusUnauthorized,
				"message": "请提供有效的身份验证凭证",
			})
			return
		}

		// 格式必须是"Bearer {token}"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			logger.Debug("无效的Authorization标头格式",
				zap.String("header", authHeader))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   ErrInvalidToken.Error(),
				"code":    http.StatusUnauthorized,
				"message": "请提供有效的Bearer令牌",
			})
			return
		}

		tokenString := parts[1]
		claims := &TokenClaims{}

		// 解析和验证令牌
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// 确保签名方法是HMAC
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("意外的签名方法: %v", token.Header["alg"])
			}
			return []byte(config.Secret), nil
		})

		// 处理令牌解析错误
		if err != nil {
			var validationErr *jwt.ValidationError
			if errors.As(err, &validationErr) {
				if validationErr.Errors&jwt.ValidationErrorExpired != 0 {
					logger.Debug("令牌已过期")
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
						"error":   ErrExpiredToken.Error(),
						"code":    http.StatusUnauthorized,
						"message": "身份验证已过期，请重新登录",
					})
					return
				}
			}
			logger.Debug("无效令牌", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   ErrInvalidToken.Error(),
				"code":    http.StatusUnauthorized,
				"message": "提供的身份验证令牌无效",
			})
			return
		}

		// 确保令牌有效
		if !token.Valid {
			logger.Debug("令牌无效")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   ErrInvalidToken.Error(),
				"code":    http.StatusUnauthorized,
				"message": "提供的身份验证令牌无效",
			})
			return
		}

		// 将令牌声明存储在上下文中以供后续处理使用
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("roles", claims.Roles)
		c.Set("claims", claims)

		// 处理请求
		c.Next()
	}
}

// GenerateToken 生成新的JWT令牌
func GenerateToken(userID, username, email string, roles []string, secret string, expiry time.Duration) (string, error) {
	// 创建令牌声明
	claims := TokenClaims{
		UserID:   userID,
		Username: username,
		Email:    email,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "syslens-api",
			Subject:   userID,
		},
	}

	// 创建令牌
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 签名令牌
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("签署令牌错误: %w", err)
	}

	return tokenString, nil
}

// RoleRequired 是基于角色的授权中间件
// 验证用户是否具有所需的角色
func RoleRequired(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文中获取角色
		rolesValue, exists := c.Get("roles")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "未授权",
				"code":    http.StatusForbidden,
				"message": "请先进行身份验证",
			})
			return
		}

		// 转换角色为字符串切片
		roles, ok := rolesValue.([]string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error":   "内部服务器错误",
				"code":    http.StatusInternalServerError,
				"message": "角色格式错误",
			})
			return
		}

		// 检查用户是否具有所需角色
		authorized := false
		for _, requiredRole := range requiredRoles {
			for _, role := range roles {
				if role == requiredRole {
					authorized = true
					break
				}
			}
			if authorized {
				break
			}
		}

		if !authorized {
			logger := c.MustGet("logger").(*zap.Logger)
			logger.Debug("用户缺少所需的角色",
				zap.Strings("required_roles", requiredRoles),
				zap.Strings("user_roles", roles))

			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "拒绝访问",
				"code":    http.StatusForbidden,
				"message": "您没有访问此资源的权限",
			})
			return
		}

		// 继续处理请求
		c.Next()
	}
}
