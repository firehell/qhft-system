package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// PolygonDataSource 实现了Polygon.io数据源
type PolygonDataSource struct {
	config     DataSourceConfig
	httpClient *http.Client
}

// NewPolygonDataSource 创建一个新的Polygon.io数据源
func NewPolygonDataSource(config DataSourceConfig) (*PolygonDataSource, error) {
	if config.TimeoutSeconds <= 0 {
		config.TimeoutSeconds = 30 // 默认30秒超时
	}
	config.Timeout = time.Duration(config.TimeoutSeconds) * time.Second

	if config.RetryAttempts <= 0 {
		config.RetryAttempts = 3 // 默认重试3次
	}

	if config.RetryDelaySeconds <= 0 {
		config.RetryDelaySeconds = 5 // 默认延迟5秒
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	return &PolygonDataSource{
		config:     config,
		httpClient: httpClient,
	}, nil
}

// Name 返回数据源名称
func (p *PolygonDataSource) Name() string {
	return "polygon"
}

// IsEnabled 检查数据源是否启用
func (p *PolygonDataSource) IsEnabled() bool {
	return p.config.Enabled
}

// HealthCheck 检查Polygon.io API的连接状态
func (p *PolygonDataSource) HealthCheck(ctx context.Context) (bool, error) {
	// 简单调用一个轻量级API检查连接是否正常
	endpoint := fmt.Sprintf("%s/v1/marketstatus/now?apiKey=%s", 
		p.config.BaseURL, p.config.APIKey)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, &DataSourceError{
			Source:  p.Name(),
			Code:    "REQUEST_CREATION_ERROR",
			Message: fmt.Sprintf("Failed to create request: %v", err),
			Time:    time.Now(),
		}
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return false, &DataSourceError{
			Source:  p.Name(),
			Code:    "CONNECTION_ERROR",
			Message: fmt.Sprintf("Connection failed: %v", err),
			Time:    time.Now(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, &DataSourceError{
			Source:  p.Name(),
			Code:    "API_ERROR",
			Message: fmt.Sprintf("API returned status code %d", resp.StatusCode),
			Time:    time.Now(),
		}
	}

	return true, nil
}

// GetStockData 获取指定股票的价格数据
func (p *PolygonDataSource) GetStockData(ctx context.Context, symbol string, timeframe string, from, to time.Time) ([]StockData, error) {
	// 构建API URL
	endpoint := fmt.Sprintf("%s/v2/aggs/ticker/%s/range/1/%s/%s/%s?apiKey=%s",
		p.config.BaseURL,
		symbol,
		timeframe,
		from.Format("2006-01-02"),
		to.Format("2006-01-02"),
		p.config.APIKey)

	// 发送请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, &DataSourceError{
			Source:  p.Name(),
			Code:    "REQUEST_CREATION_ERROR",
			Message: fmt.Sprintf("Failed to create request: %v", err),
			Time:    time.Now(),
		}
	}

	// 发送请求并处理重试逻辑
	var resp *http.Response
	var attempt int
	for attempt = 0; attempt < p.config.RetryAttempts; attempt++ {
		resp, err = p.httpClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		
		if resp != nil {
			resp.Body.Close()
		}
		
		// 如果不是最后一次尝试，则等待后重试
		if attempt < p.config.RetryAttempts-1 {
			select {
			case <-time.After(time.Duration(p.config.RetryDelaySeconds) * time.Second):
				continue
			case <-ctx.Done():
				return nil, &DataSourceError{
					Source:  p.Name(),
					Code:    "CONTEXT_CANCELLED",
					Message: "Request cancelled by context",
					Time:    time.Now(),
				}
			}
		}
	}

	if err != nil {
		return nil, &DataSourceError{
			Source:  p.Name(),
			Code:    "CONNECTION_ERROR",
			Message: fmt.Sprintf("Connection failed after %d attempts: %v", attempt+1, err),
			Time:    time.Now(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &DataSourceError{
			Source:  p.Name(),
			Code:    "API_ERROR",
			Message: fmt.Sprintf("API returned status code %d: %s", resp.StatusCode, string(bodyBytes)),
			Time:    time.Now(),
		}
	}

	// 解析响应
	var result struct {
		Status string `json:"status"`
		Results []struct {
			V  float64 `json:"v"`  // 成交量
			O  float64 `json:"o"`  // 开盘价
			C  float64 `json:"c"`  // 收盘价
			H  float64 `json:"h"`  // 最高价
			L  float64 `json:"l"`  // 最低价
			T  int64   `json:"t"`  // 时间戳（毫秒）
			N  int     `json:"n"`  // 交易次数
			VW float64 `json:"vw"` // 成交量加权平均价
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, &DataSourceError{
			Source:  p.Name(),
			Code:    "RESPONSE_PARSE_ERROR",
			Message: fmt.Sprintf("Failed to parse response: %v", err),
			Time:    time.Now(),
		}
	}

	// 转换为标准格式
	stockData := make([]StockData, 0, len(result.Results))
	for _, bar := range result.Results {
		stockData = append(stockData, StockData{
			Symbol:    symbol,
			Timestamp: time.Unix(0, bar.T*int64(time.Millisecond)),
			Open:      bar.O,
			High:      bar.H,
			Low:       bar.L,
			Close:     bar.C,
			Volume:    int64(bar.V),
			VWAP:      bar.VW,
			TransactionID: fmt.Sprintf("polygon_%s_%d", symbol, bar.T),
		})
	}

	return stockData, nil
}

// GetMultipleStockData 批量获取多只股票的价格数据
func (p *PolygonDataSource) GetMultipleStockData(ctx context.Context, symbols []string, timeframe string, from, to time.Time) (map[string][]StockData, error) {
	result := make(map[string][]StockData)
	
	// Polygon.io API不支持批量获取，所以这里逐个调用
	for _, symbol := range symbols {
		data, err := p.GetStockData(ctx, symbol, timeframe, from, to)
		if err != nil {
			return result, err
		}
		result[symbol] = data
	}
	
	return result, nil
}

// GetRealTimeQuote 获取实时报价
func (p *PolygonDataSource) GetRealTimeQuote(ctx context.Context, symbol string) (*Quote, error) {
	// 构建API URL
	endpoint := fmt.Sprintf("%s/v2/last/nbbo/%s?apiKey=%s",
		p.config.BaseURL, symbol, p.config.APIKey)

	// 发送请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, &DataSourceError{
			Source:  p.Name(),
			Code:    "REQUEST_CREATION_ERROR",
			Message: fmt.Sprintf("Failed to create request: %v", err),
			Time:    time.Now(),
		}
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, &DataSourceError{
			Source:  p.Name(),
			Code:    "CONNECTION_ERROR",
			Message: fmt.Sprintf("Connection failed: %v", err),
			Time:    time.Now(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &DataSourceError{
			Source:  p.Name(),
			Code:    "API_ERROR",
			Message: fmt.Sprintf("API returned status code %d: %s", resp.StatusCode, string(bodyBytes)),
			Time:    time.Now(),
		}
	}

	// 解析响应
	var result struct {
		Status  string `json:"status"`
		Results struct {
			T  int64   `json:"t"`  // 时间戳（纳秒）
			P  float64 `json:"p"`  // 最后成交价
			S  int64   `json:"s"`  // 最后成交量
			AP float64 `json:"ap"` // 卖出价
			AS int64   `json:"as"` // 卖出量
			BP float64 `json:"bp"` // 买入价
			BS int64   `json:"bs"` // 买入量
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, &DataSourceError{
			Source:  p.Name(),
			Code:    "RESPONSE_PARSE_ERROR",
			Message: fmt.Sprintf("Failed to parse response: %v", err),
			Time:    time.Now(),
		}
	}

	// 转换为标准格式
	quote := &Quote{
		Symbol:        symbol,
		Timestamp:     time.Unix(0, result.Results.T),
		AskPrice:      result.Results.AP,
		AskSize:       result.Results.AS,
		BidPrice:      result.Results.BP,
		BidSize:       result.Results.BS,
		LastPrice:     result.Results.P,
		LastSize:      result.Results.S,
		TransactionID: fmt.Sprintf("polygon_%s_%d", symbol, result.Results.T),
	}

	return quote, nil
}

// GetAllStocks 获取所有可交易的股票列表
func (p *PolygonDataSource) GetAllStocks(ctx context.Context) ([]Stock, error) {
	// 构建API URL，Polygon.io的参数允许设置每页数量和市场类型
	endpoint := fmt.Sprintf("%s/v3/reference/tickers?market=stocks&active=true&limit=1000&apiKey=%s",
		p.config.BaseURL, p.config.APIKey)

	var allStocks []Stock
	var nextURL string = endpoint

	// 分页获取所有数据
	for nextURL != "" {
		// 发送请求
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, nextURL, nil)
		if err != nil {
			return nil, &DataSourceError{
				Source:  p.Name(),
				Code:    "REQUEST_CREATION_ERROR",
				Message: fmt.Sprintf("Failed to create request: %v", err),
				Time:    time.Now(),
			}
		}

		resp, err := p.httpClient.Do(req)
		if err != nil {
			return nil, &DataSourceError{
				Source:  p.Name(),
				Code:    "CONNECTION_ERROR",
				Message: fmt.Sprintf("Connection failed: %v", err),
				Time:    time.Now(),
			}
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			bodyBytes, _ := io.ReadAll(resp.Body)
			return nil, &DataSourceError{
				Source:  p.Name(),
				Code:    "API_ERROR",
				Message: fmt.Sprintf("API returned status code %d: %s", resp.StatusCode, string(bodyBytes)),
				Time:    time.Now(),
			}
		}

		// 解析响应
		var result struct {
			Status    string `json:"status"`
			NextURL   string `json:"next_url"`
			Results []struct {
				Ticker          string `json:"ticker"`
				Name            string `json:"name"`
				Market          string `json:"market"`
				PrimaryExchange string `json:"primary_exchange"`
				Type            string `json:"type"`
				Active          bool   `json:"active"`
				CurrencyName    string `json:"currency_name"`
				Description     string `json:"description"`
			} `json:"results"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, &DataSourceError{
				Source:  p.Name(),
				Code:    "RESPONSE_PARSE_ERROR",
				Message: fmt.Sprintf("Failed to parse response: %v", err),
				Time:    time.Now(),
			}
		}
		resp.Body.Close()

		// 转换并添加到结果数组
		for _, item := range result.Results {
			allStocks = append(allStocks, Stock{
				Symbol:      item.Ticker,
				Name:        item.Name,
				Exchange:    item.PrimaryExchange,
				Type:        item.Type,
				Currency:    item.CurrencyName,
				IsActive:    item.Active,
				Description: item.Description,
			})
		}

		// 检查是否有下一页
		if result.NextURL != "" {
			// 提取next_url并添加API密钥
			parsedURL, err := url.Parse(result.NextURL)
			if err != nil {
				return nil, &DataSourceError{
					Source:  p.Name(),
					Code:    "URL_PARSE_ERROR",
					Message: fmt.Sprintf("Failed to parse next_url: %v", err),
					Time:    time.Now(),
				}
			}
			
			q := parsedURL.Query()
			q.Set("apiKey", p.config.APIKey)
			parsedURL.RawQuery = q.Encode()
			
			nextURL = parsedURL.String()
		} else {
			nextURL = ""
		}
	}

	return allStocks, nil
}

// Close 关闭数据源连接
func (p *PolygonDataSource) Close() error {
	// HTTP客户端不需要显式关闭
	return nil
} 