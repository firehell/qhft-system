package indicators

import (
	"fmt"
	"time"

	"github.com/yourusername/qhft-system/pkg/datasource"
)

// RSI 指标结构体
type RSI struct {
	period int
}

// NewRSI 创建一个新的RSI指标
func NewRSI(params IndicatorParams) (Indicator, error) {
	period := params.GetInt("period", 14)

	// 验证参数
	if period <= 0 {
		return nil, fmt.Errorf("period must be a positive integer")
	}

	return &RSI{
		period: period,
	}, nil
}

// Name 返回指标名称
func (r *RSI) Name() string {
	return IndicatorTypeRSI
}

// Calculate 计算RSI指标值
func (r *RSI) Calculate(data []datasource.StockData) (IndicatorResult, error) {
	if len(data) < r.period + 1 {
		return IndicatorResult{}, fmt.Errorf("not enough data points for RSI calculation (minimum: %d, got: %d)", 
			r.period+1, len(data))
	}

	// 提取收盘价
	prices := make([]float64, len(data))
	dates := make([]string, len(data))
	for i, bar := range data {
		prices[i] = bar.Close
		dates[i] = bar.Timestamp.Format(time.RFC3339)
	}

	// 计算价格变动
	changes := make([]float64, len(prices)-1)
	for i := 0; i < len(changes); i++ {
		changes[i] = prices[i+1] - prices[i]
	}

	// 初始化
	rsiValues := make([]float64, len(prices))
	for i := 0; i < r.period; i++ {
		rsiValues[i] = 0
	}

	// 为第一个RSI值计算平均增长和下跌
	var sumGain, sumLoss float64
	for i := 0; i < r.period; i++ {
		if changes[i] > 0 {
			sumGain += changes[i]
		} else {
			sumLoss -= changes[i]
		}
	}

	// 计算首个平均值
	avgGain := sumGain / float64(r.period)
	avgLoss := sumLoss / float64(r.period)

	// 避免除以零
	if avgLoss == 0 {
		rsiValues[r.period] = 100
	} else {
		rs := avgGain / avgLoss
		rsiValues[r.period] = 100 - (100 / (1 + rs))
	}

	// 计算剩余的RSI值
	for i := r.period + 1; i < len(prices); i++ {
		change := changes[i-1]
		var currentGain, currentLoss float64
		if change > 0 {
			currentGain = change
			currentLoss = 0
		} else {
			currentGain = 0
			currentLoss = -change
		}

		// 使用前一个平均值的平滑计算
		avgGain = (avgGain*float64(r.period-1) + currentGain) / float64(r.period)
		avgLoss = (avgLoss*float64(r.period-1) + currentLoss) / float64(r.period)

		// 避免除以零
		if avgLoss == 0 {
			rsiValues[i] = 100
		} else {
			rs := avgGain / avgLoss
			rsiValues[i] = 100 - (100 / (1 + rs))
		}
	}

	// 创建结果
	result := IndicatorResult{
		Name: r.Name(),
		Values: map[string][]float64{
			"rsi": rsiValues,
		},
		Dates: dates,
	}

	return result, nil
}

// EvaluateCondition 评估RSI指标条件
func (r *RSI) EvaluateCondition(result IndicatorResult, condition string, threshold float64) (bool, error) {
	if len(result.Values["rsi"]) == 0 {
		return false, fmt.Errorf("RSI result is empty")
	}

	// 获取最新的RSI值
	idx := len(result.Values["rsi"]) - 1
	prevIdx := idx - 1
	if prevIdx < 0 {
		return false, fmt.Errorf("not enough data points for RSI condition evaluation")
	}

	rsi := result.Values["rsi"][idx]
	prevRsi := result.Values["rsi"][prevIdx]

	switch condition {
	case ConditionAboveThreshold:
		// RSI高于阈值
		return rsi > threshold, nil
	case ConditionBelowThreshold:
		// RSI低于阈值
		return rsi < threshold, nil
	case ConditionCrossAbove:
		// RSI上穿阈值
		return prevRsi < threshold && rsi > threshold, nil
	case ConditionCrossBelow:
		// RSI下穿阈值
		return prevRsi > threshold && rsi < threshold, nil
	case ConditionIncreasing:
		// RSI值增加
		return rsi > prevRsi, nil
	case ConditionDecreasing:
		// RSI值减少
		return rsi < prevRsi, nil
	default:
		return false, fmt.Errorf("unsupported condition for RSI: %s", condition)
	}
} 