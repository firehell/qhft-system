package indicators

import (
	"github.com/yourusername/qhft-system/pkg/datasource"
)

// 指标类型常量
const (
	IndicatorTypeMACD     = "MACD"
	IndicatorTypeRSI      = "RSI"
	IndicatorTypeBollinger = "BollingerBands"
	IndicatorTypeEMA      = "EMA"
	IndicatorTypeSMA      = "SMA"
	IndicatorTypeStochastic = "Stochastic"
	IndicatorTypeKDJ      = "KDJ"
	IndicatorTypeATR      = "ATR"
	IndicatorTypeVWAP     = "VWAP"
)

// 条件类型常量
const (
	ConditionCrossAbove      = "cross_above"
	ConditionCrossBelow      = "cross_below"
	ConditionAboveThreshold  = "above_threshold"
	ConditionBelowThreshold  = "below_threshold"
	ConditionPriceBelowLower = "price_below_lower"
	ConditionPriceAboveUpper = "price_above_upper"
	ConditionPriceWithinBands = "price_within_bands"
	ConditionIncreasing      = "increasing"
	ConditionDecreasing      = "decreasing"
)

// Indicator 定义了一个技术指标的接口
type Indicator interface {
	// Name 返回指标名称
	Name() string
	
	// Calculate 计算指标值
	Calculate(data []datasource.StockData) (IndicatorResult, error)
	
	// EvaluateCondition 评估指标值是否满足指定条件
	EvaluateCondition(result IndicatorResult, condition string, threshold float64) (bool, error)
}

// IndicatorResult 表示指标计算结果
type IndicatorResult struct {
	Name   string             `json:"name"`
	Values map[string][]float64 `json:"values"` // 键是值名称，如"macd", "signal", "histogram"
	Dates  []string           `json:"dates"`
}

// IndicatorParams 表示指标参数
type IndicatorParams map[string]interface{}

// 从map中获取整数参数，带默认值
func (p IndicatorParams) GetInt(key string, defaultValue int) int {
	if val, ok := p[key]; ok {
		if intVal, ok := val.(int); ok {
			return intVal
		}
		// 尝试从其他类型转换
		if floatVal, ok := val.(float64); ok {
			return int(floatVal)
		}
	}
	return defaultValue
}

// 从map中获取浮点数参数，带默认值
func (p IndicatorParams) GetFloat(key string, defaultValue float64) float64 {
	if val, ok := p[key]; ok {
		if floatVal, ok := val.(float64); ok {
			return floatVal
		}
		// 尝试从其他类型转换
		if intVal, ok := val.(int); ok {
			return float64(intVal)
		}
	}
	return defaultValue
}

// 从map中获取字符串参数，带默认值
func (p IndicatorParams) GetString(key string, defaultValue string) string {
	if val, ok := p[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return defaultValue
}

// 从map中获取布尔参数，带默认值
func (p IndicatorParams) GetBool(key string, defaultValue bool) bool {
	if val, ok := p[key]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return defaultValue
}

// 指标配置结构体
type IndicatorConfig struct {
	Name           string         `json:"name" yaml:"name"`
	Type           string         `json:"type" yaml:"type"`
	Parameters     IndicatorParams `json:"parameters" yaml:"parameters"`
	BuyCondition   string         `json:"buy_condition" yaml:"buy_condition"`
	BuyThreshold   float64        `json:"buy_threshold" yaml:"buy_threshold"`
	SellCondition  string         `json:"sell_condition" yaml:"sell_condition"`
	SellThreshold  float64        `json:"sell_threshold" yaml:"sell_threshold"`
	Weight         float64        `json:"weight" yaml:"weight"` // 在组合策略中的权重
}

// 策略结构体
type Strategy struct {
	Name       string            `json:"name" yaml:"name"`
	Enabled    bool              `json:"enabled" yaml:"enabled"`
	Indicators []IndicatorConfig `json:"indicators" yaml:"indicators"`
}

// IndicatorFactory 创建指标的工厂函数类型
type IndicatorFactory func(params IndicatorParams) (Indicator, error) 