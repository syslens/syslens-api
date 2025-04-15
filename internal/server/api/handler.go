package api

import (
	"time"

	"github.com/syslens/syslens-api/internal/common/utils"
	"github.com/syslens/syslens-api/internal/config"
	"github.com/syslens/syslens-api/internal/server/repository"
	"go.uber.org/zap"
)

// MetricsHandler 处理指标相关的API请求
type MetricsHandler struct {
	storage        MetricsStorage            // 指标存储接口
	securityConfig *config.SecurityConfig    // 安全配置
	encryptionSvc  *utils.EncryptionService  // 加密服务
	logger         *zap.Logger               // 日志记录器
	nodeRepo       repository.NodeRepository // 节点仓库接口
}

// MetricsStorage 定义了指标存储接口
type MetricsStorage interface {
	StoreMetrics(nodeID string, metrics interface{}) error
	GetNodeMetrics(nodeID string, start, end time.Time) ([]interface{}, error)
	GetAllNodes() ([]string, error)
	GetLatestMetrics(nodeID string) (interface{}, error)
}

// NewMetricsHandler 创建新的指标处理器
func NewMetricsHandler(storage MetricsStorage) *MetricsHandler {
	return &MetricsHandler{
		storage: storage,
		securityConfig: &config.SecurityConfig{
			Encryption: config.EncryptionConfig{
				Enabled:   false,
				Algorithm: "aes-256-gcm",
				Key:       "",
			},
			Compression: config.CompressionConfig{
				Enabled:   false,
				Algorithm: "gzip",
			},
		},
		logger: zap.NewNop(), // 默认使用空日志记录器
	}
}

// WithSecurityConfig 设置安全配置
func (h *MetricsHandler) WithSecurityConfig(secConfig *config.SecurityConfig) {
	if secConfig != nil {
		h.securityConfig = secConfig
		// 初始化加密服务
		if h.securityConfig.Encryption.Enabled {
			h.encryptionSvc = utils.NewEncryptionService(h.securityConfig.Encryption.Algorithm)
		}
	}
}

// WithLogger 设置日志记录器
func (h *MetricsHandler) WithLogger(logger *zap.Logger) {
	if logger != nil {
		h.logger = logger
	}
}

// WithNodeRepository 设置节点仓库
func (h *MetricsHandler) WithNodeRepository(repo repository.NodeRepository) {
	h.nodeRepo = repo
}

// processData 处理数据：解密和解压缩
func (h *MetricsHandler) processData(data []byte, isEncrypted, isCompressed bool) ([]byte, error) {
	processedData := data
	var err error

	// 步骤1：解密（如果启用）
	if isEncrypted && h.securityConfig.Encryption.Enabled && h.encryptionSvc != nil {
		processedData, err = h.encryptionSvc.Decrypt(processedData, h.securityConfig.Encryption.Key)
		if err != nil {
			return nil, err
		}
	}

	// 步骤2：解压缩（如果启用）
	if isCompressed && h.securityConfig.Compression.Enabled {
		processedData, err = utils.DecompressData(processedData)
		if err != nil {
			return nil, err
		}
	}

	return processedData, nil
}
