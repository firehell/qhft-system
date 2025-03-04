package indicators

import (
	"fmt"
	"sync"
)

// IndicatorRegistry 是指标工厂的注册表
type IndicatorRegistry struct {
	mu      sync.RWMutex
	factories map[string]IndicatorFactory
}

// NewIndicatorRegistry 创建一个新的指标注册表
func NewIndicatorRegistry() *IndicatorRegistry {
	registry := &IndicatorRegistry{
		factories: make(map[string]IndicatorFactory),
	}
	
	// 注册默认指标
	registry.RegisterIndicator(IndicatorTypeMACD, NewMACD)
	registry.RegisterIndicator(IndicatorTypeRSI, NewRSI)
	registry.RegisterIndicator(IndicatorTypeBollinger, NewBollingerBands)
	registry.RegisterIndicator(IndicatorTypeEMA, NewEMA)
	registry.RegisterIndicator(IndicatorTypeSMA, NewSMA)
	
	return registry
}

// RegisterIndicator 注册一个指标工厂
func (r *IndicatorRegistry) RegisterIndicator(name string, factory IndicatorFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[name] = factory
}

// CreateIndicator 创建一个指标
func (r *IndicatorRegistry) CreateIndicator(name string, params IndicatorParams) (Indicator, error) {
	r.mu.RLock()
	factory, exists := r.factories[name]
	r.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("indicator '%s' not registered", name)
	}
	
	return factory(params)
}

// GetAvailableIndicators 获取所有可用的指标类型
func (r *IndicatorRegistry) GetAvailableIndicators() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	indicators := make([]string, 0, len(r.factories))
	for name := range r.factories {
		indicators = append(indicators, name)
	}
	
	return indicators
} 