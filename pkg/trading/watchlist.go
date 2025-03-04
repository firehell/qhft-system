package trading

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/yourusername/qhft-system/pkg/datasource"
	"github.com/yourusername/qhft-system/pkg/indicators"
)

// WatchlistItemStatus 表示监控项状态
type WatchlistItemStatus string

const (
	// 监控项状态常量
	WatchStatusActive    WatchlistItemStatus = "active"    // 活跃的
	WatchStatusTriggered WatchlistItemStatus = "triggered" // 已触发
	WatchStatusExpired   WatchlistItemStatus = "expired"   // 已过期
	WatchStatusInvalid   WatchlistItemStatus = "invalid"   // 无效的
)

// WatchlistItem 表示监控项
type WatchlistItem struct {
	ID            string               `json:"id"`
	Symbol        string               `json:"symbol"`
	TargetPrice   float64              `json:"target_price,omitempty"`
	StopLoss      float64              `json:"stop_loss,omitempty"`
	TakeProfit    float64              `json:"take_profit,omitempty"`
	Quantity      int64                `json:"quantity"`
	Status        WatchlistItemStatus  `json:"status"`
	AddedAt       time.Time            `json:"added_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	ExpiresAt     *time.Time           `json:"expires_at,omitempty"`
	TriggeredAt   *time.Time           `json:"triggered_at,omitempty"`
	Strategy      string               `json:"strategy,omitempty"`
	ScanResults   []indicators.ScanResult `json:"scan_results,omitempty"`
	Notes         string               `json:"notes,omitempty"`
	Tags          []string             `json:"tags,omitempty"`
	OrderID       string               `json:"order_id,omitempty"`
	IsBuyList     bool                 `json:"is_buy_list"`
}

// Watchlist 表示监控列表（买入表或卖出表）
type Watchlist struct {
	mu         sync.RWMutex
	items      map[string]WatchlistItem
	engine     TradingEngine
	dataManager *datasource.Manager
}

// NewWatchlist 创建新的监控列表
func NewWatchlist(engine TradingEngine, dataManager *datasource.Manager) *Watchlist {
	return &Watchlist{
		items:       make(map[string]WatchlistItem),
		engine:      engine,
		dataManager: dataManager,
	}
}

// AddItem 添加监控项
func (w *Watchlist) AddItem(item WatchlistItem) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 验证必填字段
	if item.Symbol == "" {
		return errors.New("symbol is required")
	}
	if item.Quantity <= 0 {
		return errors.New("quantity must be positive")
	}

	// 设置默认值
	if item.ID == "" {
		item.ID = fmt.Sprintf("watch-%s-%d", item.Symbol, time.Now().UnixNano())
	}
	if item.Status == "" {
		item.Status = WatchStatusActive
	}
	if item.AddedAt.IsZero() {
		item.AddedAt = time.Now()
	}
	item.UpdatedAt = time.Now()

	// 存储项目
	w.items[item.ID] = item

	return nil
}

// GetItem 获取监控项
func (w *Watchlist) GetItem(id string) (WatchlistItem, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	item, exists := w.items[id]
	if !exists {
		return WatchlistItem{}, fmt.Errorf("item with ID '%s' not found", id)
	}

	return item, nil
}

// GetItemBySymbol 通过股票代码获取监控项
func (w *Watchlist) GetItemBySymbol(symbol string) (WatchlistItem, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	for _, item := range w.items {
		if item.Symbol == symbol && item.Status == WatchStatusActive {
			return item, nil
		}
	}

	return WatchlistItem{}, fmt.Errorf("active item for symbol '%s' not found", symbol)
}

// UpdateItem 更新监控项
func (w *Watchlist) UpdateItem(id string, updatedItem WatchlistItem) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	item, exists := w.items[id]
	if !exists {
		return fmt.Errorf("item with ID '%s' not found", id)
	}

	// 保留不可修改的字段
	updatedItem.ID = item.ID
	updatedItem.AddedAt = item.AddedAt
	updatedItem.UpdatedAt = time.Now()

	// 存储更新后的项目
	w.items[id] = updatedItem

	return nil
}

// RemoveItem 移除监控项
func (w *Watchlist) RemoveItem(id string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.items[id]; !exists {
		return fmt.Errorf("item with ID '%s' not found", id)
	}

	delete(w.items, id)
	return nil
}

// GetAllItems 获取所有监控项
func (w *Watchlist) GetAllItems() []WatchlistItem {
	w.mu.RLock()
	defer w.mu.RUnlock()

	items := make([]WatchlistItem, 0, len(w.items))
	for _, item := range w.items {
		items = append(items, item)
	}

	return items
}

// GetActiveItems 获取所有活跃的监控项
func (w *Watchlist) GetActiveItems() []WatchlistItem {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var activeItems []WatchlistItem
	for _, item := range w.items {
		if item.Status == WatchStatusActive {
			activeItems = append(activeItems, item)
		}
	}

	return activeItems
}

// ScanWatchlist 扫描监控列表中的股票
func (w *Watchlist) ScanWatchlist(ctx context.Context) ([]WatchlistItem, error) {
	// 获取活跃的监控项
	activeItems := w.GetActiveItems()
	
	// 用于存储需要更新的项目
	var updatedItems []WatchlistItem
	var triggeredItems []WatchlistItem
	
	// 逐个检查监控项
	for _, item := range activeItems {
		// 跳过已过期的项目
		if item.ExpiresAt != nil && item.ExpiresAt.Before(time.Now()) {
			item.Status = WatchStatusExpired
			item.UpdatedAt = time.Now()
			updatedItems = append(updatedItems, item)
			continue
		}
		
		// 获取最新价格
		ds, err := w.dataManager.GetPrimaryDataSource()
		if err != nil {
			continue // 跳过无法获取数据源的项目
		}
		
		quote, err := ds.GetRealTimeQuote(ctx, item.Symbol)
		if err != nil {
			continue // 跳过无法获取报价的项目
		}
		
		lastPrice := quote.LastPrice
		
		// 检查是否触发条件
		triggered := false
		
		if item.IsBuyList {
			// 买入表逻辑
			if item.TargetPrice > 0 && lastPrice <= item.TargetPrice {
				// 价格低于目标价格，可以买入
				triggered = true
			}
		} else {
			// 卖出表逻辑
			if (item.StopLoss > 0 && lastPrice <= item.StopLoss) || 
			   (item.TakeProfit > 0 && lastPrice >= item.TakeProfit) {
				// 触发止损或止盈，可以卖出
				triggered = true
			}
		}
		
		if triggered {
			now := time.Now()
			item.Status = WatchStatusTriggered
			item.TriggeredAt = &now
			item.UpdatedAt = now
			
			triggeredItems = append(triggeredItems, item)
			updatedItems = append(updatedItems, item)
		}
	}
	
	// 更新状态已改变的项目
	for _, item := range updatedItems {
		w.mu.Lock()
		w.items[item.ID] = item
		w.mu.Unlock()
	}
	
	return triggeredItems, nil
}

// ExecuteWatchlistItems 执行触发的监控项交易
func (w *Watchlist) ExecuteWatchlistItems(ctx context.Context, triggeredItems []WatchlistItem) []error {
	var errors []error
	
	for _, item := range triggeredItems {
		var err error
		var order *Order
		
		if item.IsBuyList {
			// 买入表项目，执行买入
			order, err = w.engine.SubmitOrder(ctx, item.Symbol, item.Quantity, 0, OrderTypeMarket, OrderSideBuy)
		} else {
			// 卖出表项目，执行卖出
			order, err = w.engine.SubmitOrder(ctx, item.Symbol, item.Quantity, 0, OrderTypeMarket, OrderSideSell)
		}
		
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to execute order for %s: %v", item.Symbol, err))
			continue
		}
		
		// 更新监控项状态
		item.OrderID = order.ID
		item.UpdatedAt = time.Now()
		
		w.mu.Lock()
		w.items[item.ID] = item
		w.mu.Unlock()
	}
	
	return errors
}

// StartWatchlistMonitor 启动监控列表的定期扫描
func (w *Watchlist) StartWatchlistMonitor(ctx context.Context, scanInterval time.Duration) {
	ticker := time.NewTicker(scanInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 扫描监控列表
			triggeredItems, err := w.ScanWatchlist(ctx)
			if err != nil {
				fmt.Printf("Error scanning watchlist: %v\n", err)
				continue
			}
			
			// 执行触发的项目
			if len(triggeredItems) > 0 {
				errors := w.ExecuteWatchlistItems(ctx, triggeredItems)
				for _, err := range errors {
					fmt.Printf("Error executing watchlist item: %v\n", err)
				}
			}
		}
	}
} 