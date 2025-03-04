package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
)

// defaultTradeLogger 是默认的交易日志实现
type defaultTradeLogger struct {
	mu         sync.Mutex
	baseDir    string
	currentDay time.Time
	jsonFile   *os.File
	logger     Logger
}

// NewTradeLogger 创建一个新的交易日志记录器
func NewTradeLogger(baseDir string, logger Logger) (TradeLogger, error) {
	// 确保日志目录存在
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("创建交易日志目录失败: %v", err)
	}

	if logger == nil {
		logger = GetDefaultLogger()
	}

	tl := &defaultTradeLogger{
		baseDir: baseDir,
		logger:  logger,
	}

	// 初始化为今天的日志
	if err := tl.setCurrentDay(time.Now()); err != nil {
		return nil, err
	}

	return tl, nil
}

// setCurrentDay 设置当前日期并打开相应的日志文件
func (tl *defaultTradeLogger) setCurrentDay(day time.Time) error {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	// 如果日期没变且文件已打开，不做任何操作
	if tl.currentDay.Format("2006-01-02") == day.Format("2006-01-02") && tl.jsonFile != nil {
		return nil
	}

	// 关闭旧的文件（如果有）
	if tl.jsonFile != nil {
		if err := tl.jsonFile.Close(); err != nil {
			tl.logger.Error("关闭交易日志文件失败: %v", err)
		}
		tl.jsonFile = nil
	}

	// 更新当前日期
	tl.currentDay = day

	// 创建新的日志文件
	logDir := filepath.Join(tl.baseDir, tl.currentDay.Format("2006/01"))
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建交易日志目录失败: %v", err)
	}

	logPath := filepath.Join(logDir, fmt.Sprintf("trades_%s.json", tl.currentDay.Format("2006-01-02")))
	var err error
	tl.jsonFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开交易日志文件失败: %v", err)
	}

	return nil
}

// logEntry 记录一条交易日志
func (tl *defaultTradeLogger) logEntry(entry TradeLogEntry) error {
	// 确保日期被设置
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// 确保使用正确的日期日志文件
	if err := tl.setCurrentDay(entry.Timestamp); err != nil {
		return err
	}

	tl.mu.Lock()
	defer tl.mu.Unlock()

	// 序列化并写入日志
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("序列化交易日志失败: %v", err)
	}

	if _, err := tl.jsonFile.Write(jsonBytes); err != nil {
		return fmt.Errorf("写入交易日志失败: %v", err)
	}
	if _, err := tl.jsonFile.WriteString("\n"); err != nil {
		return fmt.Errorf("写入交易日志失败: %v", err)
	}

	// 同时记录到标准日志
	logMsg := fmt.Sprintf("交易日志: %s %s 数量:%d 价格:%.2f 金额:%.2f", 
		entry.Type, entry.Symbol, entry.Quantity, entry.Price, entry.Amount)
	
	switch entry.Type {
	case "buy":
		tl.logger.Info(logMsg)
	case "sell":
		if entry.PnL > 0 {
			tl.logger.Info("%s 盈利:%.2f(%.2f%%)", logMsg, entry.PnL, entry.PnLPercent)
		} else {
			tl.logger.Warn("%s 亏损:%.2f(%.2f%%)", logMsg, entry.PnL, entry.PnLPercent)
		}
	case "position":
		tl.logger.Info(logMsg)
	case "summary":
		tl.logger.Info("每日总结: %s 交易:%d 胜率:%.2f%% 净利润:%.2f", 
			entry.Timestamp.Format("2006-01-02"), entry.Quantity, entry.PnLPercent, entry.PnL)
	}

	return nil
}

// LogBuy 记录买入操作
func (tl *defaultTradeLogger) LogBuy(entry TradeLogEntry) error {
	entry.Type = "buy"
	return tl.logEntry(entry)
}

// LogSell 记录卖出操作
func (tl *defaultTradeLogger) LogSell(entry TradeLogEntry) error {
	entry.Type = "sell"
	return tl.logEntry(entry)
}

// LogPosition 记录持仓变动
func (tl *defaultTradeLogger) LogPosition(entry TradeLogEntry) error {
	entry.Type = "position"
	return tl.logEntry(entry)
}

// LogSummary 记录每日交易汇总
func (tl *defaultTradeLogger) LogSummary(summary DailySummary) error {
	entry := TradeLogEntry{
		Type:       "summary",
		Timestamp:  summary.Date,
		Quantity:   int64(summary.TotalTrades),
		PnL:        summary.NetProfit,
		PnLPercent: summary.WinRate,
	}

	// 将汇总详情序列化为JSON并存储在元数据中
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return fmt.Errorf("序列化交易汇总失败: %v", err)
	}

	// 写入汇总日志文件
	summaryDir := filepath.Join(tl.baseDir, "summaries", summary.Date.Format("2006/01"))
	if err := os.MkdirAll(summaryDir, 0755); err != nil {
		return fmt.Errorf("创建交易汇总目录失败: %v", err)
	}

	summaryPath := filepath.Join(summaryDir, fmt.Sprintf("summary_%s.json", summary.Date.Format("2006-01-02")))
	if err := os.WriteFile(summaryPath, summaryJSON, 0644); err != nil {
		return fmt.Errorf("写入交易汇总失败: %v", err)
	}

	return tl.logEntry(entry)
}

// GetDailyLogs 获取特定日期的交易日志
func (tl *defaultTradeLogger) GetDailyLogs(date time.Time) ([]TradeLogEntry, error) {
	logDir := filepath.Join(tl.baseDir, date.Format("2006/01"))
	logPath := filepath.Join(logDir, fmt.Sprintf("trades_%s.json", date.Format("2006-01-02")))

	// 检查文件是否存在
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return []TradeLogEntry{}, nil
	}

	// 读取文件内容
	content, err := os.ReadFile(logPath)
	if err != nil {
		return nil, fmt.Errorf("读取交易日志失败: %v", err)
	}

	// 解析每一行为一个日志条目
	var entries []TradeLogEntry
	lines := splitLines(string(content))
	for _, line := range lines {
		if line == "" {
			continue
		}

		var entry TradeLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			tl.logger.Error("解析交易日志条目失败: %v", err)
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// GetDateRange 获取日期范围内的所有交易日志
func (tl *defaultTradeLogger) GetDateRange(start, end time.Time) ([]TradeLogEntry, error) {
	var allEntries []TradeLogEntry

	// 遍历日期范围
	for d := truncateToDay(start); !d.After(truncateToDay(end)); d = d.AddDate(0, 0, 1) {
		entries, err := tl.GetDailyLogs(d)
		if err != nil {
			tl.logger.Error("获取日期 %s 的交易日志失败: %v", d.Format("2006-01-02"), err)
			continue
		}
		allEntries = append(allEntries, entries...)
	}

	return allEntries, nil
}

// ExportToExcel 将特定日期的交易日志导出为Excel文件
func (tl *defaultTradeLogger) ExportToExcel(date time.Time, filePath string) error {
	// 获取日志数据
	entries, err := tl.GetDailyLogs(date)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		return fmt.Errorf("日期 %s 没有交易记录", date.Format("2006-01-02"))
	}

	// 创建一个新的Excel文件
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			tl.logger.Error("关闭Excel文件失败: %v", err)
		}
	}()

	// 创建交易表格
	sheetName := "交易记录"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("创建Excel表格失败: %v", err)
	}
	f.SetActiveSheet(index)

	// 设置表头
	headers := []string{"时间", "类型", "股票代码", "数量", "价格", "金额", "手续费", "盈亏", "盈亏%", "持仓", "成本", "持有时间", "策略", "订单ID", "备注"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// 填充数据
	for i, entry := range entries {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), entry.Timestamp.Format("2006-01-02 15:04:05"))
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), entry.Type)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), entry.Symbol)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), entry.Quantity)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), entry.Price)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), entry.Amount)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), entry.Commission)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), entry.PnL)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), entry.PnLPercent)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), entry.Position)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), entry.EntryPrice)
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), entry.HoldTime)
		f.SetCellValue(sheetName, fmt.Sprintf("M%d", row), entry.Strategy)
		f.SetCellValue(sheetName, fmt.Sprintf("N%d", row), entry.OrderID)
		f.SetCellValue(sheetName, fmt.Sprintf("O%d", row), entry.Notes)
	}

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 20)
	f.SetColWidth(sheetName, "B", "C", 12)
	f.SetColWidth(sheetName, "D", "L", 12)
	f.SetColWidth(sheetName, "M", "O", 20)

	// 保存Excel文件
	if err := f.SaveAs(filePath); err != nil {
		return fmt.Errorf("保存Excel文件失败: %v", err)
	}

	return nil
}

// Close 关闭交易日志记录器
func (tl *defaultTradeLogger) Close() error {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if tl.jsonFile != nil {
		return tl.jsonFile.Close()
	}
	return nil
}

// 辅助函数

// splitLines 将字符串按行分割
func splitLines(s string) []string {
	var lines []string
	var line string
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, line)
			line = ""
		} else {
			line += string(r)
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// truncateToDay 将时间截断至日期
func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// 全局默认交易日志记录器
var (
	defaultTradeLoggerInstance TradeLogger
	tradeLoggerOnce            sync.Once
)

// GetDefaultTradeLogger 获取全局默认交易日志记录器
func GetDefaultTradeLogger() TradeLogger {
	tradeLoggerOnce.Do(func() {
		logger, err := NewTradeLogger("logs/trades", GetDefaultLogger())
		if err != nil {
			fmt.Fprintf(os.Stderr, "初始化默认交易日志记录器失败: %v\n", err)
			os.Exit(1)
		}
		defaultTradeLoggerInstance = logger
	})

	return defaultTradeLoggerInstance
}

// InitDefaultTradeLogger 初始化全局默认交易日志记录器
func InitDefaultTradeLogger(baseDir string, logger Logger) {
	tradeLogger, err := NewTradeLogger(baseDir, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化默认交易日志记录器失败: %v\n", err)
		os.Exit(1)
	}
	defaultTradeLoggerInstance = tradeLogger
} 