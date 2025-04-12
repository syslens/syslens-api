package utils

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// EncryptionService 提供加密和解密功能
type EncryptionService struct {
	algorithm string
}

// NewEncryptionService 创建新的加密服务
func NewEncryptionService(algorithm string) *EncryptionService {
	if algorithm == "" {
		algorithm = "aes-256-gcm" // 默认算法
	}
	return &EncryptionService{
		algorithm: algorithm,
	}
}

// Encrypt 使用提供的密钥加密数据
func (s *EncryptionService) Encrypt(data []byte, key string) ([]byte, error) {
	if len(key) == 0 {
		return data, nil // 如果没有提供密钥，则不加密
	}

	keyBytes := []byte(key)
	// 确保密钥长度为32字节 (AES-256)
	if len(keyBytes) < 32 {
		newKey := make([]byte, 32)
		copy(newKey, keyBytes)
		keyBytes = newKey
	} else if len(keyBytes) > 32 {
		keyBytes = keyBytes[:32]
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}

	// GCM模式提供认证加密
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 创建随机nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// 加密数据
	ciphertext := aesGCM.Seal(nonce, nonce, data, nil)

	// base64编码便于传输
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(ciphertext)))
	base64.StdEncoding.Encode(encoded, ciphertext)

	return encoded, nil
}

// Decrypt 使用提供的密钥解密数据
func (s *EncryptionService) Decrypt(data []byte, key string) ([]byte, error) {
	if len(key) == 0 {
		return data, nil // 如果没有提供密钥，则假设数据未加密
	}

	// base64解码
	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
	n, err := base64.StdEncoding.Decode(decoded, data)
	if err != nil {
		return nil, err
	}
	decoded = decoded[:n]

	keyBytes := []byte(key)
	// 确保密钥长度为32字节 (AES-256)
	if len(keyBytes) < 32 {
		newKey := make([]byte, 32)
		copy(newKey, keyBytes)
		keyBytes = newKey
	} else if len(keyBytes) > 32 {
		keyBytes = keyBytes[:32]
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesGCM.NonceSize()
	if len(decoded) < nonceSize {
		return nil, errors.New("密文太短")
	}

	nonce, ciphertext := decoded[:nonceSize], decoded[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// CompressData 压缩数据
func CompressData(data []byte, level int) ([]byte, error) {
	if level < 1 || level > 9 {
		level = 6 // 默认压缩级别
	}

	var compressedBuf bytes.Buffer
	compressor, err := gzip.NewWriterLevel(&compressedBuf, level)
	if err != nil {
		return nil, err
	}

	if _, err := compressor.Write(data); err != nil {
		return nil, err
	}
	if err := compressor.Close(); err != nil {
		return nil, err
	}

	return compressedBuf.Bytes(), nil
}

// DecompressData 解压数据
func DecompressData(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}
