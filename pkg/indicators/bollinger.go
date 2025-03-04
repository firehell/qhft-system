package indicators

import (
	"fmt"
	"math"
	"time"

	"github.com/yourusername/qhft-system/pkg/datasource"
)

// BollingerBands 布林带指标结构体
type BollingerBands struct {
	period int
	stdDev float64
}

// NewBollingerBands 创建一个新的布林带指标
func NewBollingerBands(params IndicatorParams) (Indicator, error) {
	period := params.GetInt("period", 20)
	stdDev := params.GetFloat("std_dev", 2.0)

	// 验证参数
	if period <= 0 {
		return nil, fmt.Errorf("period must be a positive integer")
	}

	if stdDev <= 0 {
		return nil, fmt.Errorf("standard deviation must be positive")
	}

	return &BollingerBands{
		period: period,
		stdDev: stdDev,
	}, nil
}

// Name 返回指标名称
func (b *BollingerBands) Name() string {
	return IndicatorTypeBollinger
}

// Calculate 计算布林带指标值
func (b *BollingerBands) Calculate(data []datasource.StockData) (IndicatorResult, error) {
	if len(data) < b.period {
		return IndicatorResult{}, fmt.Errorf("not enough data points for Bollinger Bands calculation (minimum: %d, got: %d)", 
			b.period, len(data))
	}

	// 提取收盘价
	prices := make([]float64, len(data))
	dates := make([]string, len(data))
	for i, bar := range data {
		prices[i] = bar.Close
		dates[i] = bar.Timestamp.Format(time.RFC3339)
	}

	// 计算移动平均线 (SMA)
	sma := make([]float64, len(prices))
	for i := 0; i < b.period-1; i++ {
		sma[i] = 0
	}

	for i := b.period - 1; i < len(prices); i++ {
		var sum float64
		for j := i - (b.period - 1); j <= i; j++ {
			sum += prices[j]
		}
		sma[i] = sum / float64(b.period)
	}

	// 计算标准差
	stdDevValues := make([]float64, len(prices))
	for i := 0; i < b.period-1; i++ {
		stdDevValues[i] = 0
	}

	for i := b.period - 1; i < len(prices); i++ {
		var sumSquaredDev float64
		for j := i - (b.period - 1); j <= i; j++ {
			dev := prices[j] - sma[i]
			sumSquaredDev += dev * dev
		}
		stdDevValues[i] = math.Sqrt(sumSquaredDev / float64(b.period))
	}

	// 计算上轨和下轨
	upperBand := make([]float64, len(prices))
	lowerBand := make([]float64, len(prices))
	for i := 0; i < len(prices); i++ {
		if i < b.period-1 {
			upperBand[i] = 0
			lowerBand[i] = 0
		} else {
			upperBand[i] = sma[i] + b.stdDev*stdDevValues[i]
			lowerBand[i] = sma[i] - b.stdDev*stdDevValues[i]
		}
	}

	// 计算带宽和百分比带宽
	bandwidth := make([]float64, len(prices))
	bPercent := make([]float64, len(prices))
	for i := 0; i < len(prices); i++ {
		if i < b.period-1 {
			bandwidth[i] = 0
			bPercent[i] = 0
		} else {
			bandwidth[i] = upperBand[i] - lowerBand[i]
			if sma[i] != 0 {
				bPercent[i] = bandwidth[i] / sma[i] * 100
			} else {
				bPercent[i] = 0
			}
		}
	}

	// 创建结果
	result := IndicatorResult{
		Name: b.Name(),
		Values: map[string][]float64{
			"middle":      sma,
			"upper":       upperBand,
			"lower":       lowerBand,
			"bandwidth":   bandwidth,
			"b_percent":   bPercent,
			"std_dev":     stdDevValues,
		},
		Dates: dates,
	}

	return result, nil
}

// EvaluateCondition 评估布林带指标条件
func (b *BollingerBands) EvaluateCondition(result IndicatorResult, condition string, threshold float64) (bool, error) {
	if len(result.Values["middle"]) == 0 || len(result.Values["upper"]) == 0 || len(result.Values["lower"]) == 0 {
		return false, fmt.Errorf("Bollinger Bands result is empty")
	}

	// 获取最新的值和价格
	idx := len(result.Values["middle"]) - 1
	prevIdx := idx - 1
	if prevIdx < 0 {
		return false, fmt.Errorf("not enough data points for Bollinger Bands condition evaluation")
	}

	middle := result.Values["middle"][idx]
	upper := result.Values["upper"][idx]
	lower := result.Values["lower"][idx]
	
	// 假设价格是第一个输入参数
	price := threshold

	switch condition {
	case ConditionPriceAboveUpper:
		// 价格高于上轨
		return price > upper, nil
	case ConditionPriceBelowLower:
		// 价格低于下轨
		return price < lower, nil
	case ConditionPriceWithinBands:
		// 价格在上下轨之间
		return price >= lower && price <= upper, nil
	case ConditionAboveThreshold:
		// 带宽百分比高于阈值
		bPercent := result.Values["b_percent"][idx]
		return bPercent > threshold, nil
	case ConditionBelowThreshold:
		// 带宽百分比低于阈值
		bPercent := result.Values["b_percent"][idx]
		return bPercent < threshold, nil
	default:
		return false, fmt.Errorf("unsupported condition for Bollinger Bands: %s", condition)
	}
} 