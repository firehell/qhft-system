package datasource

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Manager 数据源管理器，管理多个数据源
type Manager struct {
	mu         sync.RWMutex
	dataSources map[string]DataSource
	primary    string // 主数据源名称
}

// NewManager 创建一个新的数据源管理器
func NewManager() *Manager {
	return &Manager{
		dataSources: make(map[string]DataSource),
	}
}

// AddDataSource 添加一个数据源
func (m *Manager) AddDataSource(ds DataSource) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := ds.Name()
	if _, exists := m.dataSources[name]; exists {
		return fmt.Errorf("data source '%s' already exists", name)
	}

	m.dataSources[name] = ds

	// 如果是第一个数据源，设置为主数据源
	if len(m.dataSources) == 1 {
		m.primary = name
	}

	return nil
}

// RemoveDataSource 移除数据源
func (m *Manager) RemoveDataSource(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ds, exists := m.dataSources[name]
	if !exists {
		return fmt.Errorf("data source '%s' does not exist", name)
	}

	// 关闭数据源
	if err := ds.Close(); err != nil {
		return fmt.Errorf("failed to close data source '%s': %v", name, err)
	}

	delete(m.dataSources, name)

	// 如果删除的是主数据源，那么需要选择新的主数据源
	if m.primary == name {
		if len(m.dataSources) > 0 {
			for n := range m.dataSources {
				m.primary = n
				break
			}
		} else {
			m.primary = ""
		}
	}

	return nil
}

// GetDataSource 获取指定名称的数据源
func (m *Manager) GetDataSource(name string) (DataSource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ds, exists := m.dataSources[name]
	if !exists {
		return nil, fmt.Errorf("data source '%s' does not exist", name)
	}

	return ds, nil
}

// GetPrimaryDataSource 获取主数据源
func (m *Manager) GetPrimaryDataSource() (DataSource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.primary == "" {
		return nil, fmt.Errorf("no primary data source set")
	}

	return m.dataSources[m.primary], nil
}

// SetPrimaryDataSource 设置主数据源
func (m *Manager) SetPrimaryDataSource(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.dataSources[name]; !exists {
		return fmt.Errorf("data source '%s' does not exist", name)
	}

	m.primary = name
	return nil
}

// GetAllDataSources 获取所有数据源
func (m *Manager) GetAllDataSources() map[string]DataSource {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 创建副本以避免外部修改
	result := make(map[string]DataSource, len(m.dataSources))
	for name, ds := range m.dataSources {
		result[name] = ds
	}

	return result
}

// HealthCheckAll 检查所有数据源的健康状态
func (m *Manager) HealthCheckAll(ctx context.Context) map[string]error {
	m.mu.RLock()
	dataSources := make(map[string]DataSource, len(m.dataSources))
	for name, ds := range m.dataSources {
		dataSources[name] = ds
	}
	m.mu.RUnlock()

	results := make(map[string]error)
	var wg sync.WaitGroup

	// 并行检查所有数据源
	for name, ds := range dataSources {
		wg.Add(1)
		go func(name string, ds DataSource) {
			defer wg.Done()

			// 检查数据源是否启用
			if !ds.IsEnabled() {
				results[name] = fmt.Errorf("data source is disabled")
				return
			}

			// 设置超时上下文
			checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			// 执行健康检查
			_, err := ds.HealthCheck(checkCtx)
			results[name] = err
		}(name, ds)
	}

	wg.Wait()
	return results
}

// Close 关闭所有数据源连接
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for name, ds := range m.dataSources {
		if err := ds.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close data source '%s': %v", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing data sources: %v", errs)
	}

	return nil
}

// GetStockData 从主数据源获取股票数据，如果失败则尝试备用数据源
func (m *Manager) GetStockData(ctx context.Context, symbol string, timeframe string, from, to time.Time) ([]StockData, error) {
	m.mu.RLock()
	primary := m.primary
	dataSources := make(map[string]DataSource, len(m.dataSources))
	for name, ds := range m.dataSources {
		dataSources[name] = ds
	}
	m.mu.RUnlock()

	// 首先尝试主数据源
	if primaryDS, exists := dataSources[primary]; exists && primaryDS.IsEnabled() {
		data, err := primaryDS.GetStockData(ctx, symbol, timeframe, from, to)
		if err == nil {
			return data, nil
		}

		// 记录主数据源错误，但继续尝试备用数据源
		fmt.Printf("Primary data source '%s' failed: %v\n", primary, err)
	}

	// 尝试其他数据源
	var lastErr error
	for name, ds := range dataSources {
		if name == primary || !ds.IsEnabled() {
			continue
		}

		data, err := ds.GetStockData(ctx, symbol, timeframe, from, to)
		if err == nil {
			return data, nil
		}

		lastErr = err
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all data sources failed, last error: %v", lastErr)
	}

	return nil, fmt.Errorf("no data sources available")
}

// CreatePolygonDataSource 创建一个Polygon.io数据源并添加到管理器
func (m *Manager) CreatePolygonDataSource(config DataSourceConfig) error {
	ds, err := NewPolygonDataSource(config)
	if err != nil {
		return err
	}

	return m.AddDataSource(ds)
} 