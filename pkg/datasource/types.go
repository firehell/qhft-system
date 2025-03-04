package datasource

import (
	"context"
	"time"
)

// DataSource 接口定义了所有数据源必须实现的方法
type DataSource interface {
	// Name 返回数据源的名称
	Name() string
	
	// IsEnabled 检查数据源是否启用
	IsEnabled() bool
	
	// HealthCheck 检查数据源的健康状态
	HealthCheck(ctx context.Context) (bool, error)
	
	// GetStockData 获取指定股票的价格数据
	GetStockData(ctx context.Context, symbol string, timeframe string, from, to time.Time) ([]StockData, error)
	
	// GetMultipleStockData 批量获取多只股票的价格数据
	GetMultipleStockData(ctx context.Context, symbols []string, timeframe string, from, to time.Time) (map[string][]StockData, error)
	
	// GetRealTimeQuote 获取实时报价
	GetRealTimeQuote(ctx context.Context, symbol string) (*Quote, error)
	
	// GetAllStocks 获取所有可交易的股票列表
	GetAllStocks(ctx context.Context) ([]Stock, error)
	
	// Close 关闭数据源连接
	Close() error
}

// StockData 定义了股票价格数据的结构
type StockData struct {
	Symbol        string    `json:"symbol"`
	Timestamp     time.Time `json:"timestamp"`
	Open          float64   `json:"open"`
	High          float64   `json:"high"`
	Low           float64   `json:"low"`
	Close         float64   `json:"close"`
	Volume        int64     `json:"volume"`
	VWAP          float64   `json:"vwap,omitempty"`         // 成交量加权平均价
	TransactionID string    `json:"transaction_id,omitempty"` // 交易ID，用于追踪数据来源
}

// Quote 定义了实时报价的结构
type Quote struct {
	Symbol        string    `json:"symbol"`
	Timestamp     time.Time `json:"timestamp"`
	AskPrice      float64   `json:"ask_price"`
	AskSize       int64     `json:"ask_size"`
	BidPrice      float64   `json:"bid_price"`
	BidSize       int64     `json:"bid_size"`
	LastPrice     float64   `json:"last_price"`
	LastSize      int64     `json:"last_size"`
	TransactionID string    `json:"transaction_id,omitempty"`
}

// Stock 定义了股票基本信息的结构
type Stock struct {
	Symbol      string `json:"symbol"`
	Name        string `json:"name"`
	Exchange    string `json:"exchange"`
	Type        string `json:"type"` // 如：common stock, etf等
	Currency    string `json:"currency"`
	IsActive    bool   `json:"is_active"`
	Description string `json:"description,omitempty"`
}

// DataSourceConfig 定义了数据源配置的结构
type DataSourceConfig struct {
	Name              string        `json:"name" yaml:"name"`
	Enabled           bool          `json:"enabled" yaml:"enabled"`
	APIKey            string        `json:"api_key" yaml:"api_key"`
	BaseURL           string        `json:"base_url" yaml:"base_url"`
	TimeoutSeconds    int           `json:"timeout_seconds" yaml:"timeout_seconds"`
	RetryAttempts     int           `json:"retry_attempts" yaml:"retry_attempts"`
	RetryDelaySeconds int           `json:"retry_delay_seconds" yaml:"retry_delay_seconds"`
	Timeout           time.Duration `json:"-" yaml:"-"` // 在初始化时根据TimeoutSeconds计算
}

// DataSourceError 定义了数据源错误的结构
type DataSourceError struct {
	Source  string `json:"source"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Time    time.Time `json:"time"`
}

func (e *DataSourceError) Error() string {
	return e.Message
} 