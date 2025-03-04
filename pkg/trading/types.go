package trading

import (
	"time"
)

// OrderStatus 表示订单状态
type OrderStatus string

// 订单状态常量
const (
	OrderStatusPending   OrderStatus = "pending"   // 待处理
	OrderStatusSubmitted OrderStatus = "submitted" // 已提交
	OrderStatusAccepted  OrderStatus = "accepted"  // 已接受
	OrderStatusRejected  OrderStatus = "rejected"  // 已拒绝
	OrderStatusFilled    OrderStatus = "filled"    // 已成交
	OrderStatusPartial   OrderStatus = "partial"   // 部分成交
	OrderStatusCanceled  OrderStatus = "canceled"  // 已取消
	OrderStatusExpired   OrderStatus = "expired"   // 已过期
)

// OrderType 表示订单类型
type OrderType string

// 订单类型常量
const (
	OrderTypeMarket OrderType = "market" // 市价单
	OrderTypeLimit  OrderType = "limit"  // 限价单
	OrderTypeStop   OrderType = "stop"   // 止损单
)

// OrderSide 表示订单方向
type OrderSide string

// 订单方向常量
const (
	OrderSideBuy  OrderSide = "buy"  // 买入
	OrderSideSell OrderSide = "sell" // 卖出
)

// Order 表示交易订单
type Order struct {
	ID            string      `json:"id"`
	Symbol        string      `json:"symbol"`
	Quantity      int64       `json:"quantity"`
	FilledQty     int64       `json:"filled_qty"`
	Price         float64     `json:"price"`
	StopPrice     float64     `json:"stop_price,omitempty"`
	Type          OrderType   `json:"type"`
	Side          OrderSide   `json:"side"`
	Status        OrderStatus `json:"status"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	FilledAt      *time.Time  `json:"filled_at,omitempty"`
	AvgFillPrice  float64     `json:"avg_fill_price,omitempty"`
	Commission    float64     `json:"commission,omitempty"`
	RejectReason  string      `json:"reject_reason,omitempty"`
	ClientOrderID string      `json:"client_order_id,omitempty"`
	BrokerOrderID string      `json:"broker_order_id,omitempty"`
	Tags          []string    `json:"tags,omitempty"`
}

// Position 表示持仓
type Position struct {
	Symbol        string    `json:"symbol"`
	Quantity      int64     `json:"quantity"`
	EntryPrice    float64   `json:"entry_price"`
	CurrentPrice  float64   `json:"current_price"`
	MarketValue   float64   `json:"market_value"`
	Cost          float64   `json:"cost"`
	UnrealizedPnL float64   `json:"unrealized_pnl"`
	PnLPercent    float64   `json:"pnl_percent"`
	OpenedAt      time.Time `json:"opened_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	StopLoss      float64   `json:"stop_loss,omitempty"`
	TakeProfit    float64   `json:"take_profit,omitempty"`
	Tags          []string  `json:"tags,omitempty"`
}

// Account 表示交易账户
type Account struct {
	ID                     string    `json:"id"`
	BrokerID               string    `json:"broker_id"`
	Cash                   float64   `json:"cash"`
	BuyingPower            float64   `json:"buying_power"`
	Equity                 float64   `json:"equity"`
	MarginUsed             float64   `json:"margin_used"`
	InitialMargin          float64   `json:"initial_margin"`
	MaintenanceMargin      float64   `json:"maintenance_margin"`
	DayTradeCount          int       `json:"day_trade_count"`
	LastEquity             float64   `json:"last_equity"`
	RealizedPnL            float64   `json:"realized_pnl"`
	UnrealizedPnL          float64   `json:"unrealized_pnl"`
	TotalPnL               float64   `json:"total_pnl"`
	PnLPercent             float64   `json:"pnl_percent"`
	UpdatedAt              time.Time `json:"updated_at"`
	IsLocked               bool      `json:"is_locked"`
	IsPatternDayTrader     bool      `json:"is_pattern_day_trader"`
	IsDayTradingCalls      bool      `json:"is_day_trading_calls"`
	IsMarginCalls          bool      `json:"is_margin_calls"`
	MaxPositionSize        int64     `json:"max_position_size"`
	MaxPositionValuePercent float64   `json:"max_position_value_percent"`
	MaxDailyTrades         int       `json:"max_daily_trades"`
}

// Execution 表示交易执行记录
type Execution struct {
	ID           string    `json:"id"`
	OrderID      string    `json:"order_id"`
	Symbol       string    `json:"symbol"`
	Quantity     int64     `json:"quantity"`
	Price        float64   `json:"price"`
	Side         OrderSide `json:"side"`
	ExecutedAt   time.Time `json:"executed_at"`
	Commission   float64   `json:"commission"`
	BrokerExecID string    `json:"broker_exec_id,omitempty"`
}

// TradeStats 表示交易统计
type TradeStats struct {
	TotalTrades      int     `json:"total_trades"`
	WinningTrades    int     `json:"winning_trades"`
	LosingTrades     int     `json:"losing_trades"`
	WinRate          float64 `json:"win_rate"`
	AverageProfit    float64 `json:"average_profit"`
	AverageLoss      float64 `json:"average_loss"`
	ProfitFactor     float64 `json:"profit_factor"`
	LargestWin       float64 `json:"largest_win"`
	LargestLoss      float64 `json:"largest_loss"`
	AverageHoldTime  float64 `json:"average_hold_time"`
	SharpRatio       float64 `json:"sharpe_ratio"`
	MaxDrawdownValue float64 `json:"max_drawdown_value"`
	MaxDrawdownPercent float64 `json:"max_drawdown_percent"`
}

// Trade 表示一个完整的交易（开仓和平仓）
type Trade struct {
	ID             string     `json:"id"`
	Symbol         string     `json:"symbol"`
	EntryOrder     Order      `json:"entry_order"`
	ExitOrder      *Order     `json:"exit_order,omitempty"`
	EntryPrice     float64    `json:"entry_price"`
	ExitPrice      float64    `json:"exit_price,omitempty"`
	Quantity       int64      `json:"quantity"`
	RealizedPnL    float64    `json:"realized_pnl"`
	RealizedPnLPercent float64 `json:"realized_pnl_percent"`
	Commission     float64    `json:"commission"`
	OpenedAt       time.Time  `json:"opened_at"`
	ClosedAt       *time.Time `json:"closed_at,omitempty"`
	HoldTime       float64    `json:"hold_time,omitempty"` // 以小时为单位
	Tags           []string   `json:"tags,omitempty"`
	Notes          string     `json:"notes,omitempty"`
	Strategy       string     `json:"strategy,omitempty"`
}

// BrokerConfig 表示券商配置
type BrokerConfig struct {
	Name         string `json:"name" yaml:"name"`
	APIKey       string `json:"api_key" yaml:"api_key"`
	APISecret    string `json:"api_secret" yaml:"api_secret"`
	AccountID    string `json:"account_id" yaml:"account_id"`
	IsPaperTrading bool   `json:"is_paper_trading" yaml:"is_paper_trading"`
	BaseURL      string `json:"base_url" yaml:"base_url"`
}

// TradingLimits 表示交易限制
type TradingLimits struct {
	MaxPositions          int     `json:"max_positions" yaml:"max_positions"`
	MaxPositionSizePercent float64 `json:"max_position_size_percent" yaml:"max_position_size_percent"`
	MaxDailyTrades        int     `json:"max_daily_trades" yaml:"max_daily_trades"`
	StopLossPercent       float64 `json:"stop_loss_percent" yaml:"stop_loss_percent"`
	TakeProfitPercent     float64 `json:"take_profit_percent" yaml:"take_profit_percent"`
} 