package logger

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultLogger(t *testing.T) {
	// 创建临时目录用于测试
	tempDir := filepath.Join(os.TempDir(), "qhft-logger-test")
	defer os.RemoveAll(tempDir)
	
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	
	// 创建日志配置
	config := LogConfig{
		Level:      LogLevelDebug,
		Format:     LogFormatText,
		Output:     LogOutputFile,
		FilePath:   filepath.Join(tempDir, "test.log"),
		MaxSizeMB:  1,
		MaxBackups: 1,
		MaxAgeDays: 1,
		Compress:   false,
	}
	
	// 创建日志记录器
	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("创建日志记录器失败: %v", err)
	}
	
	// 记录不同级别的日志
	logger.Debug("这是一条调试日志")
	logger.Info("这是一条信息日志")
	logger.Warn("这是一条警告日志")
	logger.Error("这是一条错误日志")
	
	// 使用上下文
	logger.WithField("key", "value").Info("带字段的日志")
	logger.WithFields(map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}).Info("带多个字段的日志")
	
	// 关闭日志记录器
	if err := logger.Close(); err != nil {
		t.Fatalf("关闭日志记录器失败: %v", err)
	}
	
	// 检查日志文件是否存在
	if _, err := os.Stat(config.FilePath); os.IsNotExist(err) {
		t.Fatalf("日志文件不存在: %v", err)
	}
}

func TestTradeLogger(t *testing.T) {
	// 创建临时目录用于测试
	tempDir := filepath.Join(os.TempDir(), "qhft-trade-logger-test")
	defer os.RemoveAll(tempDir)
	
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	
	// 创建系统日志记录器
	sysLogger, err := NewLogger(LogConfig{
		Level:    LogLevelDebug,
		Format:   LogFormatText,
		Output:   LogOutputFile,
		FilePath: filepath.Join(tempDir, "system.log"),
	})
	if err != nil {
		t.Fatalf("创建系统日志记录器失败: %v", err)
	}
	defer sysLogger.Close()
	
	// 创建交易日志记录器
	tradeLogger, err := NewTradeLogger(filepath.Join(tempDir, "trades"), sysLogger)
	if err != nil {
		t.Fatalf("创建交易日志记录器失败: %v", err)
	}
	defer tradeLogger.Close()
	
	// 记录买入交易
	buyEntry := TradeLogEntry{
		Timestamp:  time.Now(),
		Symbol:     "AAPL",
		Quantity:   100,
		Price:      150.25,
		Amount:     15025.00,
		Commission: 5.99,
		Strategy:   "测试策略",
		OrderID:    "test-order-1",
	}
	
	if err := tradeLogger.LogBuy(buyEntry); err != nil {
		t.Fatalf("记录买入交易失败: %v", err)
	}
	
	// 记录卖出交易
	sellEntry := TradeLogEntry{
		Timestamp:   time.Now(),
		Symbol:      "AAPL",
		Quantity:    100,
		Price:       155.50,
		Amount:      15550.00,
		Commission:  5.99,
		PnL:         519.01,
		PnLPercent:  3.45,
		Strategy:    "测试策略",
		OrderID:     "test-order-2",
	}
	
	if err := tradeLogger.LogSell(sellEntry); err != nil {
		t.Fatalf("记录卖出交易失败: %v", err)
	}
	
	// 获取今天的交易日志
	entries, err := tradeLogger.GetDailyLogs(time.Now())
	if err != nil {
		t.Fatalf("获取交易日志失败: %v", err)
	}
	
	if len(entries) != 2 {
		t.Fatalf("预期有2条交易记录，实际有%d条", len(entries))
	}
	
	if entries[0].Type != "buy" || entries[1].Type != "sell" {
		t.Fatalf("交易类型不匹配")
	}
	
	if entries[0].Symbol != "AAPL" || entries[1].Symbol != "AAPL" {
		t.Fatalf("交易股票代码不匹配")
	}
} 