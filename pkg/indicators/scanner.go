package indicators

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yourusername/qhft-system/pkg/datasource"
)

// ScanResult 表示扫描结果
type ScanResult struct {
	Symbol        string    `json:"symbol"`
	Timestamp     time.Time `json:"timestamp"`
	IndicatorName string    `json:"indicator_name"`
	Condition     string    `json:"condition"`
	Value         float64   `json:"value"`
	Threshold     float64   `json:"threshold"`
	IsBuySignal   bool      `json:"is_buy_signal"`
	IsSellSignal  bool      `json:"is_sell_signal"`
	Score         float64   `json:"score"` // 组合策略中的得分
}

// Scanner 指标扫描器
type Scanner struct {
	registry         *IndicatorRegistry
	dataManager      *datasource.Manager
	strategies       map[string]Strategy
	defaultTimeframe string
}

// NewScanner 创建一个新的指标扫描器
func NewScanner(registry *IndicatorRegistry, dataManager *datasource.Manager) *Scanner {
	return &Scanner{
		registry:         registry,
		dataManager:      dataManager,
		strategies:       make(map[string]Strategy),
		defaultTimeframe: "day",
	}
}

// AddStrategy 添加策略
func (s *Scanner) AddStrategy(strategy Strategy) error {
	if _, exists := s.strategies[strategy.Name]; exists {
		return fmt.Errorf("strategy '%s' already exists", strategy.Name)
	}

	s.strategies[strategy.Name] = strategy
	return nil
}

// RemoveStrategy 移除策略
func (s *Scanner) RemoveStrategy(name string) error {
	if _, exists := s.strategies[name]; !exists {
		return fmt.Errorf("strategy '%s' does not exist", name)
	}

	delete(s.strategies, name)
	return nil
}

// GetStrategy 获取指定名称的策略
func (s *Scanner) GetStrategy(name string) (Strategy, error) {
	strategy, exists := s.strategies[name]
	if !exists {
		return Strategy{}, fmt.Errorf("strategy '%s' does not exist", name)
	}

	return strategy, nil
}

// GetAllStrategies 获取所有策略
func (s *Scanner) GetAllStrategies() map[string]Strategy {
	return s.strategies
}

// SetDefaultTimeframe 设置默认时间周期
func (s *Scanner) SetDefaultTimeframe(timeframe string) {
	s.defaultTimeframe = timeframe
}

// ScanSymbol 扫描单个股票
func (s *Scanner) ScanSymbol(ctx context.Context, symbol string, strategyName string, from, to time.Time, timeframe string) ([]ScanResult, error) {
	if timeframe == "" {
		timeframe = s.defaultTimeframe
	}

	// 获取策略
	strategy, err := s.GetStrategy(strategyName)
	if err != nil {
		return nil, err
	}

	if !strategy.Enabled {
		return nil, fmt.Errorf("strategy '%s' is disabled", strategyName)
	}

	// 获取股票数据
	stockData, err := s.dataManager.GetStockData(ctx, symbol, timeframe, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get stock data: %v", err)
	}

	if len(stockData) == 0 {
		return nil, fmt.Errorf("no stock data available for symbol '%s'", symbol)
	}

	var results []ScanResult
	var totalWeight float64

	// 计算所有指标的权重总和
	for _, indConfig := range strategy.Indicators {
		totalWeight += indConfig.Weight
	}

	// 如果总权重为0，平均分配权重
	if totalWeight == 0 {
		for i := range strategy.Indicators {
			strategy.Indicators[i].Weight = 1.0 / float64(len(strategy.Indicators))
		}
		totalWeight = 1.0
	}

	// 评估每个指标
	for _, indConfig := range strategy.Indicators {
		// 创建指标
		indicator, err := s.registry.CreateIndicator(indConfig.Type, indConfig.Parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to create indicator '%s': %v", indConfig.Type, err)
		}

		// 计算指标值
		result, err := indicator.Calculate(stockData)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate indicator '%s': %v", indConfig.Type, err)
		}

		// 最新价格用于评估条件
		latestPrice := stockData[len(stockData)-1].Close

		// 评估买入条件
		if indConfig.BuyCondition != "" {
			isBuySignal, err := indicator.EvaluateCondition(result, indConfig.BuyCondition, indConfig.BuyThreshold)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate buy condition for indicator '%s': %v", indConfig.Type, err)
			}

			if isBuySignal {
				scanResult := ScanResult{
					Symbol:        symbol,
					Timestamp:     stockData[len(stockData)-1].Timestamp,
					IndicatorName: indConfig.Type,
					Condition:     indConfig.BuyCondition,
					Value:         latestPrice,
					Threshold:     indConfig.BuyThreshold,
					IsBuySignal:   true,
					IsSellSignal:  false,
					Score:         indConfig.Weight / totalWeight,
				}
				results = append(results, scanResult)
			}
		}

		// 评估卖出条件
		if indConfig.SellCondition != "" {
			isSellSignal, err := indicator.EvaluateCondition(result, indConfig.SellCondition, indConfig.SellThreshold)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate sell condition for indicator '%s': %v", indConfig.Type, err)
			}

			if isSellSignal {
				scanResult := ScanResult{
					Symbol:        symbol,
					Timestamp:     stockData[len(stockData)-1].Timestamp,
					IndicatorName: indConfig.Type,
					Condition:     indConfig.SellCondition,
					Value:         latestPrice,
					Threshold:     indConfig.SellThreshold,
					IsBuySignal:   false,
					IsSellSignal:  true,
					Score:         indConfig.Weight / totalWeight,
				}
				results = append(results, scanResult)
			}
		}
	}

	return results, nil
}

// ScanMultipleSymbols 批量扫描多个股票
func (s *Scanner) ScanMultipleSymbols(ctx context.Context, symbols []string, strategyName string, from, to time.Time, timeframe string) (map[string][]ScanResult, error) {
	results := make(map[string][]ScanResult)
	var mu sync.Mutex
	var wg sync.WaitGroup
	errorsChan := make(chan error, len(symbols))

	// 创建一个工作池
	workers := make(chan struct{}, 10) // 最多10个并发工作
	
	for _, symbol := range symbols {
		wg.Add(1)
		workers <- struct{}{} // 获取工作槽
		
		go func(symbol string) {
			defer wg.Done()
			defer func() { <-workers }() // 释放工作槽
			
			// 扫描单个股票
			symbolResults, err := s.ScanSymbol(ctx, symbol, strategyName, from, to, timeframe)
			if err != nil {
				select {
				case errorsChan <- fmt.Errorf("failed to scan symbol '%s': %v", symbol, err):
				default:
					// 如果错误通道已满，忽略错误
				}
				return
			}
			
			if len(symbolResults) > 0 {
				mu.Lock()
				results[symbol] = symbolResults
				mu.Unlock()
			}
		}(symbol)
	}
	
	wg.Wait()
	close(errorsChan)
	
	// 检查是否有错误发生
	select {
	case err := <-errorsChan:
		return results, err
	default:
		return results, nil
	}
}

// CalculateStrategyScore 计算策略得分
func (s *Scanner) CalculateStrategyScore(results []ScanResult, isBuy bool) float64 {
	var totalScore float64
	
	for _, result := range results {
		if (isBuy && result.IsBuySignal) || (!isBuy && result.IsSellSignal) {
			totalScore += result.Score
		}
	}
	
	return totalScore
} 