package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// GenerateRandomString 生成指定长度的随机字符串
func GenerateRandomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)

	// 使用加密安全的随机数生成器
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			// 如果随机生成失败，回退到 base64 编码的随机字节
			b := make([]byte, length*2)
			rand.Read(b)
			return base64.URLEncoding.EncodeToString(b)[:length]
		}
		result[i] = chars[num.Int64()]
	}

	return string(result)
}

// HashPassword 使用 bcrypt 计算密码的哈希值
func HashPassword(password string) (string, error) {
	// 使用 bcrypt 的默认成本因子
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("生成密码哈希失败: %w", err)
	}

	return string(hashedBytes), nil
}

// ComparePasswordAndHash 验证密码与哈希值是否匹配
func ComparePasswordAndHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateSecureToken 生成安全的随机令牌
func GenerateSecureToken(length int) string {
	if length < 16 {
		length = 16 // 最小长度16
	}

	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		// 如果随机生成失败，回退到时间戳和随机字符串组合
		return fmt.Sprintf("%d-%s",
			GetCurrentTimestampMs(),
			GenerateRandomString(length-8))
	}

	// 使用 URL 安全的 base64 编码
	return base64.URLEncoding.EncodeToString(b)[:length]
}

// GetCurrentTimestampMs 获取当前时间戳（毫秒）
func GetCurrentTimestampMs() int64 {
	return GetTimeNow().UnixNano() / int64(1e6)
}

// 这个函数可以在测试中被替换，使测试更容易
var GetTimeNow = getTimeNow

// 内部函数，用于获取当前时间
// 在测试中可以被替换
func getTimeNow() time.Time {
	return time.Now()
}
