package trading

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/yourusername/qhft-system/pkg/datasource"
)

// 错误常量
var (
	ErrTradeDisabled    = errors.New("trading is disabled")
	ErrInvalidSymbol    = errors.New("invalid symbol")
	ErrInvalidQuantity  = errors.New("invalid quantity")
	ErrInvalidPrice     = errors.New("invalid price")
	ErrInvalidOrderType = errors.New("invalid order type")
	ErrInvalidOrderSide = errors.New("invalid order side")
	ErrOrderNotFound    = errors.New("order not found")
	ErrTradeLimitExceeded = errors.New("trading limit exceeded")
	ErrAccountNotFound  = errors.New("account not found")
	ErrBrokerNotAvailable = errors.New("broker not available")
)

// TradingEngine 定义了交易引擎的接口
type TradingEngine interface {
	// 订单操作
	SubmitOrder(ctx context.Context, symbol string, quantity int64, price float64, orderType OrderType, orderSide OrderSide) (*Order, error)
	CancelOrder(ctx context.Context, orderID string) error
	GetOrder(ctx context.Context, orderID string) (*Order, error)
	GetOpenOrders(ctx context.Context) ([]Order, error)
	GetOrderHistory(ctx context.Context, symbol string, startTime, endTime time.Time) ([]Order, error)
	
	// 持仓操作
	GetPositions(ctx context.Context) ([]Position, error)
	GetPosition(ctx context.Context, symbol string) (*Position, error)
	ClosePosition(ctx context.Context, symbol string, quantity int64) (*Order, error)
	
	// 账户操作
	GetAccount(ctx context.Context) (*Account, error)
	
	// 交易统计
	GetTradeStats(ctx context.Context, startTime, endTime time.Time) (*TradeStats, error)
	GetTrades(ctx context.Context, symbol string, startTime, endTime time.Time) ([]Trade, error)
	
	// 引擎控制
	IsEnabled() bool
	Enable() error
	Disable() error
	GetLimits() TradingLimits
	SetLimits(limits TradingLimits) error
}

// BaseTradingEngine 提供基本的交易引擎实现
type BaseTradingEngine struct {
	mu            sync.RWMutex
	enabled       bool
	dataManager   *datasource.Manager
	limits        TradingLimits
	brokerConfig  BrokerConfig
	orders        map[string]Order
	positions     map[string]Position
	account       Account
	trades        []Trade
	executionChan chan Execution
	errorChan     chan error
}

// NewBaseTradingEngine 创建基本交易引擎
func NewBaseTradingEngine(dataManager *datasource.Manager, brokerConfig BrokerConfig, limits TradingLimits) *BaseTradingEngine {
	return &BaseTradingEngine{
		enabled:       false,
		dataManager:   dataManager,
		limits:        limits,
		brokerConfig:  brokerConfig,
		orders:        make(map[string]Order),
		positions:     make(map[string]Position),
		executionChan: make(chan Execution, 100), // 缓冲通道，避免阻塞
		errorChan:     make(chan error, 100),
	}
}

// IsEnabled 检查交易引擎是否启用
func (e *BaseTradingEngine) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enabled
}

// Enable 启用交易引擎
func (e *BaseTradingEngine) Enable() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = true
	return nil
}

// Disable 禁用交易引擎
func (e *BaseTradingEngine) Disable() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = false
	return nil
}

// GetLimits 获取交易限制
func (e *BaseTradingEngine) GetLimits() TradingLimits {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.limits
}

// SetLimits 设置交易限制
func (e *BaseTradingEngine) SetLimits(limits TradingLimits) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.limits = limits
	return nil
}

// SubmitOrder 提交订单
func (e *BaseTradingEngine) SubmitOrder(ctx context.Context, symbol string, quantity int64, price float64, orderType OrderType, orderSide OrderSide) (*Order, error) {
	if !e.IsEnabled() {
		return nil, ErrTradeDisabled
	}
	
	e.mu.Lock()
	defer e.mu.Unlock()
	
	// 检查参数
	if symbol == "" {
		return nil, ErrInvalidSymbol
	}
	if quantity <= 0 {
		return nil, ErrInvalidQuantity
	}
	if price < 0 && orderType != OrderTypeMarket {
		return nil, ErrInvalidPrice
	}
	
	// 验证订单类型
	switch orderType {
	case OrderTypeMarket, OrderTypeLimit, OrderTypeStop:
		// 有效的订单类型
	default:
		return nil, ErrInvalidOrderType
	}
	
	// 验证订单方向
	switch orderSide {
	case OrderSideBuy, OrderSideSell:
		// 有效的订单方向
	default:
		return nil, ErrInvalidOrderSide
	}
	
	// 检查交易限制
	positionCount := len(e.positions)
	if orderSide == OrderSideBuy && positionCount >= e.limits.MaxPositions {
		return nil, fmt.Errorf("%w: maximum positions reached (%d)", ErrTradeLimitExceeded, e.limits.MaxPositions)
	}
	
	// TODO: 实现更多限制检查...
	
	// 创建新订单
	now := time.Now()
	order := Order{
		ID:        fmt.Sprintf("order-%d", now.UnixNano()),
		Symbol:    symbol,
		Quantity:  quantity,
		Price:     price,
		Type:      orderType,
		Side:      orderSide,
		Status:    OrderStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	
	// 在实际系统中，这里应该调用券商API提交订单
	// 这里我们假设订单已提交并接受
	order.Status = OrderStatusAccepted
	
	// 保存订单
	e.orders[order.ID] = order
	
	// 如果是市价单，假设立即成交
	if orderType == OrderTypeMarket {
		// 获取最新价格
		quote, err := e.dataManager.GetPrimaryDataSource()
		if err == nil {
			realTimequote, err := quote.GetRealTimeQuote(ctx, symbol)
			if err == nil {
				fillPrice := realTimequote.LastPrice
				filledTime := time.Now()
				
				// 更新订单
				order.Status = OrderStatusFilled
				order.FilledQty = quantity
				order.AvgFillPrice = fillPrice
				order.FilledAt = &filledTime
				order.UpdatedAt = filledTime
				
				// 更新持仓
				e.updatePosition(order)
				
				// 更新订单保存
				e.orders[order.ID] = order
			}
		}
	}
	
	return &order, nil
}

// CancelOrder 取消订单
func (e *BaseTradingEngine) CancelOrder(ctx context.Context, orderID string) error {
	if !e.IsEnabled() {
		return ErrTradeDisabled
	}
	
	e.mu.Lock()
	defer e.mu.Unlock()
	
	order, exists := e.orders[orderID]
	if !exists {
		return ErrOrderNotFound
	}
	
	// 检查订单状态是否可取消
	if order.Status == OrderStatusFilled || order.Status == OrderStatusCanceled || order.Status == OrderStatusRejected {
		return fmt.Errorf("cannot cancel order with status %s", order.Status)
	}
	
	// 在实际系统中，这里应该调用券商API取消订单
	// 这里我们假设订单已取消
	order.Status = OrderStatusCanceled
	order.UpdatedAt = time.Now()
	
	// 更新订单
	e.orders[orderID] = order
	
	return nil
}

// GetOrder 获取订单
func (e *BaseTradingEngine) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	order, exists := e.orders[orderID]
	if !exists {
		return nil, ErrOrderNotFound
	}
	
	return &order, nil
}

// GetOpenOrders 获取所有未成交的订单
func (e *BaseTradingEngine) GetOpenOrders(ctx context.Context) ([]Order, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	var openOrders []Order
	for _, order := range e.orders {
		if order.Status == OrderStatusPending || order.Status == OrderStatusSubmitted || order.Status == OrderStatusAccepted || order.Status == OrderStatusPartial {
			openOrders = append(openOrders, order)
		}
	}
	
	return openOrders, nil
}

// GetOrderHistory 获取历史订单
func (e *BaseTradingEngine) GetOrderHistory(ctx context.Context, symbol string, startTime, endTime time.Time) ([]Order, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	var orderHistory []Order
	for _, order := range e.orders {
		// 检查时间范围
		if order.CreatedAt.Before(startTime) || order.CreatedAt.After(endTime) {
			continue
		}
		
		// 检查股票代码
		if symbol != "" && order.Symbol != symbol {
			continue
		}
		
		orderHistory = append(orderHistory, order)
	}
	
	return orderHistory, nil
}

// GetPositions 获取所有持仓
func (e *BaseTradingEngine) GetPositions(ctx context.Context) ([]Position, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	positions := make([]Position, 0, len(e.positions))
	for _, pos := range e.positions {
		positions = append(positions, pos)
	}
	
	return positions, nil
}

// GetPosition 获取特定股票的持仓
func (e *BaseTradingEngine) GetPosition(ctx context.Context, symbol string) (*Position, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	pos, exists := e.positions[symbol]
	if !exists {
		return nil, fmt.Errorf("no position found for symbol %s", symbol)
	}
	
	return &pos, nil
}

// ClosePosition 平仓
func (e *BaseTradingEngine) ClosePosition(ctx context.Context, symbol string, quantity int64) (*Order, error) {
	if !e.IsEnabled() {
		return nil, ErrTradeDisabled
	}
	
	e.mu.Lock()
	pos, exists := e.positions[symbol]
	if !exists {
		e.mu.Unlock()
		return nil, fmt.Errorf("no position found for symbol %s", symbol)
	}
	
	// 如果数量为0或大于持仓量，则平仓全部
	if quantity <= 0 || quantity > pos.Quantity {
		quantity = pos.Quantity
	}
	
	e.mu.Unlock()
	
	// 创建市价卖单
	return e.SubmitOrder(ctx, symbol, quantity, 0, OrderTypeMarket, OrderSideSell)
}

// GetAccount 获取账户信息
func (e *BaseTradingEngine) GetAccount(ctx context.Context) (*Account, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	// 在实际系统中，这里应该调用券商API获取最新账户信息
	// 这里我们简单返回当前账户
	
	// 计算未实现盈亏
	var unrealizedPnL float64
	for _, pos := range e.positions {
		unrealizedPnL += pos.UnrealizedPnL
	}
	
	e.account.UnrealizedPnL = unrealizedPnL
	e.account.TotalPnL = e.account.RealizedPnL + unrealizedPnL
	e.account.UpdatedAt = time.Now()
	
	// 如果初始账户为空，创建一个默认账户
	if e.account.ID == "" {
		e.account.ID = "default-account"
		e.account.BrokerID = e.brokerConfig.Name
		e.account.Cash = 100000 // 默认10万美元
		e.account.BuyingPower = e.account.Cash * 2 // 假设2倍杠杆
		e.account.Equity = e.account.Cash + e.account.UnrealizedPnL
		e.account.UpdatedAt = time.Now()
		e.account.MaxPositionSize = 1000
		e.account.MaxPositionValuePercent = e.limits.MaxPositionSizePercent
		e.account.MaxDailyTrades = e.limits.MaxDailyTrades
	}
	
	return &e.account, nil
}

// GetTradeStats 获取交易统计
func (e *BaseTradingEngine) GetTradeStats(ctx context.Context, startTime, endTime time.Time) (*TradeStats, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	// 筛选时间范围内的交易
	var filteredTrades []Trade
	for _, trade := range e.trades {
		if trade.OpenedAt.After(startTime) && (trade.ClosedAt == nil || trade.ClosedAt.Before(endTime)) {
			filteredTrades = append(filteredTrades, trade)
		}
	}
	
	// 计算统计数据
	stats := TradeStats{
		TotalTrades: len(filteredTrades),
	}
	
	if len(filteredTrades) == 0 {
		return &stats, nil
	}
	
	var totalProfit, totalLoss, sumProfits, sumLosses float64
	var winCount, lossCount int
	var largestWin, largestLoss float64
	var totalHoldTime float64
	
	for _, trade := range filteredTrades {
		// 仅计算已平仓的交易
		if trade.ClosedAt != nil {
			pnl := trade.RealizedPnL
			holdTime := trade.HoldTime
			
			if pnl > 0 {
				winCount++
				sumProfits += pnl
				totalProfit += pnl
				if pnl > largestWin {
					largestWin = pnl
				}
			} else if pnl < 0 {
				lossCount++
				sumLosses += (-pnl) // 取绝对值
				totalLoss += pnl
				if pnl < largestLoss {
					largestLoss = pnl
				}
			}
			
			totalHoldTime += holdTime
		}
	}
	
	stats.WinningTrades = winCount
	stats.LosingTrades = lossCount
	
	if winCount > 0 {
		stats.AverageProfit = sumProfits / float64(winCount)
	}
	
	if lossCount > 0 {
		stats.AverageLoss = sumLosses / float64(lossCount)
	}
	
	if stats.TotalTrades > 0 {
		stats.WinRate = float64(winCount) / float64(stats.TotalTrades)
		stats.AverageHoldTime = totalHoldTime / float64(stats.TotalTrades)
	}
	
	if sumLosses > 0 {
		stats.ProfitFactor = sumProfits / sumLosses
	}
	
	stats.LargestWin = largestWin
	stats.LargestLoss = largestLoss
	
	// TODO: 计算夏普比率和最大回撤
	
	return &stats, nil
}

// GetTrades 获取交易记录
func (e *BaseTradingEngine) GetTrades(ctx context.Context, symbol string, startTime, endTime time.Time) ([]Trade, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	var filteredTrades []Trade
	for _, trade := range e.trades {
		// 检查股票代码
		if symbol != "" && trade.Symbol != symbol {
			continue
		}
		
		// 检查时间范围
		if trade.OpenedAt.Before(startTime) || (trade.ClosedAt != nil && trade.ClosedAt.After(endTime)) {
			continue
		}
		
		filteredTrades = append(filteredTrades, trade)
	}
	
	return filteredTrades, nil
}

// updatePosition 更新持仓（内部方法）
func (e *BaseTradingEngine) updatePosition(order Order) {
	if order.Status != OrderStatusFilled {
		return
	}
	
	// 更新现有持仓或创建新持仓
	symbol := order.Symbol
	pos, exists := e.positions[symbol]
	
	if order.Side == OrderSideBuy {
		// 买入
		if !exists {
			// 创建新持仓
			pos = Position{
				Symbol:       symbol,
				Quantity:     order.FilledQty,
				EntryPrice:   order.AvgFillPrice,
				CurrentPrice: order.AvgFillPrice,
				Cost:         float64(order.FilledQty) * order.AvgFillPrice,
				OpenedAt:     *order.FilledAt,
				UpdatedAt:    time.Now(),
			}
			
			// 设置止损和止盈
			if e.limits.StopLossPercent > 0 {
				pos.StopLoss = pos.EntryPrice * (1 - e.limits.StopLossPercent/100)
			}
			
			if e.limits.TakeProfitPercent > 0 {
				pos.TakeProfit = pos.EntryPrice * (1 + e.limits.TakeProfitPercent/100)
			}
		} else {
			// 加仓，计算平均成本
			totalQuantity := pos.Quantity + order.FilledQty
			totalCost := pos.Cost + float64(order.FilledQty)*order.AvgFillPrice
			pos.Quantity = totalQuantity
			pos.Cost = totalCost
			pos.EntryPrice = totalCost / float64(totalQuantity)
			pos.CurrentPrice = order.AvgFillPrice
			pos.UpdatedAt = time.Now()
			
			// 更新止损和止盈
			if e.limits.StopLossPercent > 0 {
				pos.StopLoss = pos.EntryPrice * (1 - e.limits.StopLossPercent/100)
			}
			
			if e.limits.TakeProfitPercent > 0 {
				pos.TakeProfit = pos.EntryPrice * (1 + e.limits.TakeProfitPercent/100)
			}
		}
	} else {
		// 卖出
		if !exists {
			// 没有持仓可卖，这应该是一个错误
			return
		}
		
		// 减仓
		pos.Quantity -= order.FilledQty
		pos.CurrentPrice = order.AvgFillPrice
		pos.UpdatedAt = time.Now()
		
		// 计算实现盈亏
		realizedPnL := float64(order.FilledQty) * (order.AvgFillPrice - pos.EntryPrice)
		
		// 更新账户
		e.account.RealizedPnL += realizedPnL
		
		// 如果完全平仓，则删除持仓
		if pos.Quantity <= 0 {
			// 创建交易记录
			closedTime := order.FilledAt
			holdTimeHours := closedTime.Sub(pos.OpenedAt).Hours()
			
			trade := Trade{
				ID:                 fmt.Sprintf("trade-%d", time.Now().UnixNano()),
				Symbol:             symbol,
				EntryOrder:         e.orders[order.ID], // 这里应该是开仓订单ID
				ExitOrder:          &order,
				EntryPrice:         pos.EntryPrice,
				ExitPrice:          order.AvgFillPrice,
				Quantity:           order.FilledQty,
				RealizedPnL:        realizedPnL,
				RealizedPnLPercent: (order.AvgFillPrice/pos.EntryPrice - 1) * 100,
				Commission:         order.Commission,
				OpenedAt:           pos.OpenedAt,
				ClosedAt:           closedTime,
				HoldTime:           holdTimeHours,
			}
			
			e.trades = append(e.trades, trade)
			
			// 删除持仓
			delete(e.positions, symbol)
		} else {
			// 更新持仓
			e.positions[symbol] = pos
		}
	}
	
	// 如果没有完全平仓，更新持仓
	if pos.Quantity > 0 {
		pos.MarketValue = float64(pos.Quantity) * pos.CurrentPrice
		pos.UnrealizedPnL = pos.MarketValue - pos.Cost
		pos.PnLPercent = (pos.CurrentPrice/pos.EntryPrice - 1) * 100
		e.positions[symbol] = pos
	}
} 