package main

import (
	"fmt"
	"os"
	"time"

	"github.com/username/qhft-system/pkg/logger"
)

func main() {
	// 初始化系统日志
	logConfig := logger.LogConfig{
		Level:      logger.LogLevelDebug,
		Format:     logger.LogFormatText,
		Output:     logger.LogOutputBoth,
		FilePath:   "logs/app.log",
		MaxSizeMB:  100,
		MaxBackups: 5,
		MaxAgeDays: 30,
		Compress:   true,
	}

	err := os.MkdirAll("logs", 0755)
	if err != nil {
		fmt.Printf("创建日志目录失败: %v\n", err)
		return
	}

	logger.InitDefaultLogger(logConfig)
	log := logger.GetDefaultLogger()
	defer log.Close()

	// 记录不同级别的日志
	log.Debug("这是一条调试日志")
	log.Info("系统启动成功")
	log.Warn("内存使用率高于80%%")
	log.Error("请求API失败: %s", "连接超时")

	// 使用上下文日志
	log.WithField("module", "api").Info("API服务启动")
	log.WithFields(map[string]interface{}{
		"user":   "admin",
		"action": "login",
		"ip":     "192.168.1.1",
	}).Info("用户登录")

	// 初始化交易日志记录器
	tradeLogger, err := logger.NewTradeLogger("logs/trades", log)
	if err != nil {
		log.Error("初始化交易日志失败: %v", err)
		return
	}
	defer tradeLogger.Close()

	// 记录买入交易
	buyEntry := logger.TradeLogEntry{
		Timestamp:  time.Now(),
		Symbol:     "AAPL",
		Quantity:   100,
		Price:      150.25,
		Amount:     15025.00,
		Commission: 5.99,
		Strategy:   "均线突破",
		OrderID:    "order-123456",
		Notes:      "突破20日均线",
		Tags:       []string{"技术分析", "短线"},
	}
	
	if err := tradeLogger.LogBuy(buyEntry); err != nil {
		log.Error("记录买入失败: %v", err)
	}

	// 等待一会儿(模拟持仓时间)
	time.Sleep(2 * time.Second)

	// 记录卖出交易
	sellEntry := logger.TradeLogEntry{
		Timestamp:   time.Now(),
		Symbol:      "AAPL",
		Quantity:    100,
		Price:       155.50,
		Amount:      15550.00,
		Commission:  5.99,
		PnL:         519.01, // 155.50*100 - 150.25*100 - 5.99*2
		PnLPercent:  3.45,   // (155.50 - 150.25) / 150.25 * 100
		HoldTime:    0.05,   // 小时
		Strategy:    "均线突破",
		OrderID:     "order-123457",
		Notes:       "目标价位达成",
		Tags:        []string{"技术分析", "短线"},
	}
	
	if err := tradeLogger.LogSell(sellEntry); err != nil {
		log.Error("记录卖出失败: %v", err)
	}

	// 记录持仓信息
	posEntry := logger.TradeLogEntry{
		Timestamp:  time.Now(),
		Symbol:     "MSFT",
		Quantity:   50,
		EntryPrice: 290.75,
		Position:   50,
		Notes:      "长期持有",
	}
	
	if err := tradeLogger.LogPosition(posEntry); err != nil {
		log.Error("记录持仓失败: %v", err)
	}

	// 记录每日汇总
	summary := logger.DailySummary{
		Date:               time.Now(),
		TotalTrades:        10,
		BuyTrades:          6,
		SellTrades:         4,
		WinningTrades:      7,
		LosingTrades:       3,
		WinRate:            70.0,
		GrossProfit:        1250.50,
		GrossLoss:          -320.75,
		NetProfit:          929.75,
		TotalCommission:    59.90,
		LargestWin:         519.01,
		LargestLoss:        -180.25,
		AverageTrade:       92.97,
		AverageWin:         178.64,
		AverageLoss:        -106.92,
		ProfitFactor:       3.90,
		AverageHoldingTime: 1.25,
		FinalEquity:        25929.75,
		DailyReturn:        3.72,
	}
	
	if err := tradeLogger.LogSummary(summary); err != nil {
		log.Error("记录日汇总失败: %v", err)
	}

	// 导出到Excel
	excelPath := fmt.Sprintf("logs/trades_%s.xlsx", time.Now().Format("2006-01-02"))
	err = tradeLogger.ExportToExcel(time.Now(), excelPath)
	if err != nil {
		log.Error("导出Excel失败: %v", err)
	} else {
		log.Info("成功导出交易日志到Excel: %s", excelPath)
	}

	log.Info("示例程序运行完成")
} 