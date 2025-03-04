package indicators

import (
	"fmt"
	"time"

	"github.com/yourusername/qhft-system/pkg/datasource"
)

// SMA 简单移动平均指标结构体
type SMA struct {
	period int
}

// NewSMA 创建一个新的SMA指标
func NewSMA(params IndicatorParams) (Indicator, error) {
	period := params.GetInt("period", 20)

	// 验证参数
	if period <= 0 {
		return nil, fmt.Errorf("period must be a positive integer")
	}

	return &SMA{
		period: period,
	}, nil
}

// Name 返回指标名称
func (s *SMA) Name() string {
	return IndicatorTypeSMA
}

// Calculate 计算SMA指标值
func (s *SMA) Calculate(data []datasource.StockData) (IndicatorResult, error) {
	if len(data) < s.period {
		return IndicatorResult{}, fmt.Errorf("not enough data points for SMA calculation (minimum: %d, got: %d)", 
			s.period, len(data))
	}

	// 提取收盘价
	prices := make([]float64, len(data))
	dates := make([]string, len(data))
	for i, bar := range data {
		prices[i] = bar.Close
		dates[i] = bar.Timestamp.Format(time.RFC3339)
	}

	// 计算移动平均线
	smaValues := make([]float64, len(prices))
	for i := 0; i < s.period-1; i++ {
		smaValues[i] = 0
	}

	for i := s.period - 1; i < len(prices); i++ {
		var sum float64
		for j := i - (s.period - 1); j <= i; j++ {
			sum += prices[j]
		}
		smaValues[i] = sum / float64(s.period)
	}

	// 创建结果
	result := IndicatorResult{
		Name: s.Name(),
		Values: map[string][]float64{
			"sma": smaValues,
		},
		Dates: dates,
	}

	return result, nil
}

// EvaluateCondition 评估SMA指标条件
func (s *SMA) EvaluateCondition(result IndicatorResult, condition string, threshold float64) (bool, error) {
	if len(result.Values["sma"]) == 0 {
		return false, fmt.Errorf("SMA result is empty")
	}

	// 获取最新的SMA值
	idx := len(result.Values["sma"]) - 1
	prevIdx := idx - 1
	if prevIdx < 0 {
		return false, fmt.Errorf("not enough data points for SMA condition evaluation")
	}

	sma := result.Values["sma"][idx]
	prevSma := result.Values["sma"][prevIdx]

	// 假设价格是第一个输入参数
	price := threshold

	switch condition {
	case ConditionAboveThreshold:
		// 价格高于SMA
		return price > sma, nil
	case ConditionBelowThreshold:
		// 价格低于SMA
		return price < sma, nil
	case ConditionCrossAbove:
		// 价格上穿SMA
		return price > sma && threshold < prevSma, nil
	case ConditionCrossBelow:
		// 价格下穿SMA
		return price < sma && threshold > prevSma, nil
	case ConditionIncreasing:
		// SMA值增加
		return sma > prevSma, nil
	case ConditionDecreasing:
		// SMA值减少
		return sma < prevSma, nil
	default:
		return false, fmt.Errorf("unsupported condition for SMA: %s", condition)
	}
}

// EMA 指数移动平均指标结构体
type EMA struct {
	period int
}

// NewEMA 创建一个新的EMA指标
func NewEMA(params IndicatorParams) (Indicator, error) {
	period := params.GetInt("period", 20)

	// 验证参数
	if period <= 0 {
		return nil, fmt.Errorf("period must be a positive integer")
	}

	return &EMA{
		period: period,
	}, nil
}

// Name 返回指标名称
func (e *EMA) Name() string {
	return IndicatorTypeEMA
}

// Calculate 计算EMA指标值
func (e *EMA) Calculate(data []datasource.StockData) (IndicatorResult, error) {
	if len(data) < e.period {
		return IndicatorResult{}, fmt.Errorf("not enough data points for EMA calculation (minimum: %d, got: %d)", 
			e.period, len(data))
	}

	// 提取收盘价
	prices := make([]float64, len(data))
	dates := make([]string, len(data))
	for i, bar := range data {
		prices[i] = bar.Close
		dates[i] = bar.Timestamp.Format(time.RFC3339)
	}

	// 计算EMA
	emaValues := calculateEMA(prices, e.period)

	// 创建结果
	result := IndicatorResult{
		Name: e.Name(),
		Values: map[string][]float64{
			"ema": emaValues,
		},
		Dates: dates,
	}

	return result, nil
}

// EvaluateCondition 评估EMA指标条件
func (e *EMA) EvaluateCondition(result IndicatorResult, condition string, threshold float64) (bool, error) {
	if len(result.Values["ema"]) == 0 {
		return false, fmt.Errorf("EMA result is empty")
	}

	// 获取最新的EMA值
	idx := len(result.Values["ema"]) - 1
	prevIdx := idx - 1
	if prevIdx < 0 {
		return false, fmt.Errorf("not enough data points for EMA condition evaluation")
	}

	ema := result.Values["ema"][idx]
	prevEma := result.Values["ema"][prevIdx]

	// 假设价格是第一个输入参数
	price := threshold

	switch condition {
	case ConditionAboveThreshold:
		// 价格高于EMA
		return price > ema, nil
	case ConditionBelowThreshold:
		// 价格低于EMA
		return price < ema, nil
	case ConditionCrossAbove:
		// 价格上穿EMA
		return price > ema && threshold < prevEma, nil
	case ConditionCrossBelow:
		// 价格下穿EMA
		return price < ema && threshold > prevEma, nil
	case ConditionIncreasing:
		// EMA值增加
		return ema > prevEma, nil
	case ConditionDecreasing:
		// EMA值减少
		return ema < prevEma, nil
	default:
		return false, fmt.Errorf("unsupported condition for EMA: %s", condition)
	}
} 