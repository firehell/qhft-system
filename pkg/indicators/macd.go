package indicators

import (
	"fmt"
	"time"

	"github.com/yourusername/qhft-system/pkg/datasource"
)

// MACD 指标结构体
type MACD struct {
	fastPeriod   int
	slowPeriod   int
	signalPeriod int
}

// NewMACD 创建一个新的MACD指标
func NewMACD(params IndicatorParams) (Indicator, error) {
	// 设置默认参数
	fastPeriod := params.GetInt("fast_period", 12)
	slowPeriod := params.GetInt("slow_period", 26)
	signalPeriod := params.GetInt("signal_period", 9)

	// 验证参数
	if fastPeriod <= 0 || slowPeriod <= 0 || signalPeriod <= 0 {
		return nil, fmt.Errorf("periods must be positive integers")
	}

	if fastPeriod >= slowPeriod {
		return nil, fmt.Errorf("fast period must be less than slow period")
	}

	return &MACD{
		fastPeriod:   fastPeriod,
		slowPeriod:   slowPeriod,
		signalPeriod: signalPeriod,
	}, nil
}

// Name 返回指标名称
func (m *MACD) Name() string {
	return IndicatorTypeMACD
}

// Calculate 计算MACD指标值
func (m *MACD) Calculate(data []datasource.StockData) (IndicatorResult, error) {
	if len(data) < m.slowPeriod + m.signalPeriod {
		return IndicatorResult{}, fmt.Errorf("not enough data points for MACD calculation (minimum: %d, got: %d)", 
			m.slowPeriod+m.signalPeriod, len(data))
	}

	// 提取收盘价
	prices := make([]float64, len(data))
	dates := make([]string, len(data))
	for i, bar := range data {
		prices[i] = bar.Close
		dates[i] = bar.Timestamp.Format(time.RFC3339)
	}

	// 计算EMA
	fastEMA := calculateEMA(prices, m.fastPeriod)
	slowEMA := calculateEMA(prices, m.slowPeriod)

	// 计算MACD线 = 快速EMA - 慢速EMA
	macdLine := make([]float64, len(prices))
	for i := 0; i < len(prices); i++ {
		if i < m.slowPeriod-1 {
			macdLine[i] = 0
		} else {
			macdLine[i] = fastEMA[i] - slowEMA[i]
		}
	}

	// 计算信号线 = MACD的EMA
	signalLine := calculateEMA(macdLine[m.slowPeriod-1:], m.signalPeriod)

	// 补全信号线前面的0值
	fullSignalLine := make([]float64, len(prices))
	for i := 0; i < m.slowPeriod+m.signalPeriod-2; i++ {
		fullSignalLine[i] = 0
	}
	copy(fullSignalLine[m.slowPeriod+m.signalPeriod-2:], signalLine)

	// 计算柱状图 = MACD线 - 信号线
	histogram := make([]float64, len(prices))
	for i := 0; i < len(prices); i++ {
		if i < m.slowPeriod+m.signalPeriod-2 {
			histogram[i] = 0
		} else {
			histogram[i] = macdLine[i] - fullSignalLine[i]
		}
	}

	// 创建结果
	result := IndicatorResult{
		Name: m.Name(),
		Values: map[string][]float64{
			"macd":      macdLine,
			"signal":    fullSignalLine,
			"histogram": histogram,
		},
		Dates: dates,
	}

	return result, nil
}

// EvaluateCondition 评估MACD指标条件
func (m *MACD) EvaluateCondition(result IndicatorResult, condition string, threshold float64) (bool, error) {
	if len(result.Values["macd"]) == 0 || len(result.Values["signal"]) == 0 {
		return false, fmt.Errorf("MACD result is empty")
	}

	// 获取最新的值
	idx := len(result.Values["macd"]) - 1
	prevIdx := idx - 1
	if prevIdx < 0 {
		return false, fmt.Errorf("not enough data points for MACD condition evaluation")
	}

	macd := result.Values["macd"][idx]
	prevMacd := result.Values["macd"][prevIdx]
	signal := result.Values["signal"][idx]
	prevSignal := result.Values["signal"][prevIdx]
	histogram := result.Values["histogram"][idx]
	prevHistogram := result.Values["histogram"][prevIdx]

	switch condition {
	case ConditionCrossAbove:
		// MACD线上穿信号线
		return prevMacd < prevSignal && macd > signal, nil
	case ConditionCrossBelow:
		// MACD线下穿信号线
		return prevMacd > prevSignal && macd < signal, nil
	case ConditionAboveThreshold:
		// MACD值高于阈值
		return macd > threshold, nil
	case ConditionBelowThreshold:
		// MACD值低于阈值
		return macd < threshold, nil
	case ConditionIncreasing:
		// MACD值增加
		return macd > prevMacd, nil
	case ConditionDecreasing:
		// MACD值减少
		return macd < prevMacd, nil
	default:
		return false, fmt.Errorf("unsupported condition for MACD: %s", condition)
	}
}

// calculateEMA 计算指数移动平均线
func calculateEMA(prices []float64, period int) []float64 {
	ema := make([]float64, len(prices))
	k := 2.0 / float64(period+1)

	// 第一个EMA值使用简单移动平均值
	var sum float64
	for i := 0; i < period && i < len(prices); i++ {
		sum += prices[i]
	}
	ema[period-1] = sum / float64(period)

	// 计算后续的EMA值
	for i := period; i < len(prices); i++ {
		ema[i] = prices[i]*k + ema[i-1]*(1-k)
	}

	return ema
} 