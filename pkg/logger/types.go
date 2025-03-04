package logger

import (
	"time"
)

// LogLevel 表示日志级别
type LogLevel string

// 日志级别常量
const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)

// LogFormat 表示日志格式
type LogFormat string

// 日志格式常量
const (
	LogFormatText LogFormat = "text"
	LogFormatJSON LogFormat = "json"
)

// LogOutput 表示日志输出目标
type LogOutput string

// 日志输出目标常量
const (
	LogOutputConsole LogOutput = "console"
	LogOutputFile    LogOutput = "file"
	LogOutputBoth    LogOutput = "both"
)

// LogEntry 表示一条日志记录
type LogEntry struct {
	Level     LogLevel   `json:"level"`
	Message   string     `json:"message"`
	Timestamp time.Time  `json:"timestamp"`
	Module    string     `json:"module,omitempty"`
	Function  string     `json:"function,omitempty"`
	File      string     `json:"file,omitempty"`
	Line      int        `json:"line,omitempty"`
	Context   LogContext `json:"context,omitempty"`
}

// LogContext 表示日志上下文
type LogContext map[string]interface{}

// LogConfig 表示日志配置
type LogConfig struct {
	Level      LogLevel  `json:"level" yaml:"level"`
	Format     LogFormat `json:"format" yaml:"format"`
	Output     LogOutput `json:"output" yaml:"output"`
	FilePath   string    `json:"file_path" yaml:"file_path"`
	MaxSizeMB  int       `json:"max_size_mb" yaml:"max_size_mb"`
	MaxBackups int       `json:"max_backups" yaml:"max_backups"`
	MaxAgeDays int       `json:"max_age_days" yaml:"max_age_days"`
	Compress   bool      `json:"compress" yaml:"compress"`
}

// TradeLogEntry 表示交易日志记录
type TradeLogEntry struct {
	Type           string    `json:"type"`            // "buy", "sell", "position", "summary"
	Timestamp      time.Time `json:"timestamp"`
	Symbol         string    `json:"symbol,omitempty"`
	Quantity       int64     `json:"quantity,omitempty"`
	Price          float64   `json:"price,omitempty"`
	Amount         float64   `json:"amount,omitempty"`        // 交易金额
	Commission     float64   `json:"commission,omitempty"`    // 手续费
	PnL            float64   `json:"pnl,omitempty"`           // 盈亏
	PnLPercent     float64   `json:"pnl_percent,omitempty"`   // 盈亏百分比
	Position       int64     `json:"position,omitempty"`      // 持仓数量
	EntryPrice     float64   `json:"entry_price,omitempty"`   // 平均成本
	HoldTime       float64   `json:"hold_time,omitempty"`     // 持仓时间（小时）
	Strategy       string    `json:"strategy,omitempty"`      // 交易策略
	OrderID        string    `json:"order_id,omitempty"`      // 订单ID
	ExecutionID    string    `json:"execution_id,omitempty"`  // 执行ID
	Notes          string    `json:"notes,omitempty"`         // 备注
	Tags           []string  `json:"tags,omitempty"`          // 标签
}

// DailySummary 表示每日交易汇总
type DailySummary struct {
	Date               time.Time `json:"date"`
	TotalTrades        int       `json:"total_trades"`
	BuyTrades          int       `json:"buy_trades"`
	SellTrades         int       `json:"sell_trades"`
	WinningTrades      int       `json:"winning_trades"`
	LosingTrades       int       `json:"losing_trades"`
	WinRate            float64   `json:"win_rate"`
	GrossProfit        float64   `json:"gross_profit"`
	GrossLoss          float64   `json:"gross_loss"`
	NetProfit          float64   `json:"net_profit"`
	TotalCommission    float64   `json:"total_commission"`
	LargestWin         float64   `json:"largest_win"`
	LargestLoss        float64   `json:"largest_loss"`
	AverageTrade       float64   `json:"average_trade"`
	AverageWin         float64   `json:"average_win"`
	AverageLoss        float64   `json:"average_loss"`
	ProfitFactor       float64   `json:"profit_factor"`
	AverageHoldingTime float64   `json:"average_holding_time"`
	FinalEquity        float64   `json:"final_equity"`
	DailyReturn        float64   `json:"daily_return"`
}

// Logger 接口定义了日志记录器的方法
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Fatal(msg string, args ...interface{})
	
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
	WithContext(ctx LogContext) Logger
	
	SetLevel(level LogLevel)
	GetLevel() LogLevel
	
	Close() error
}

// TradeLogger 接口定义了交易日志记录器的方法
type TradeLogger interface {
	LogBuy(entry TradeLogEntry) error
	LogSell(entry TradeLogEntry) error
	LogPosition(entry TradeLogEntry) error
	LogSummary(summary DailySummary) error
	
	GetDailyLogs(date time.Time) ([]TradeLogEntry, error)
	GetDateRange(start, end time.Time) ([]TradeLogEntry, error)
	ExportToExcel(date time.Time, filePath string) error
	
	Close() error
} 