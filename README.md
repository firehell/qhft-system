# 量化高频交易与监控系统 (QHFT System)

这是一个面向美股市场的量化高频交易系统，通过技术分析指标筛选潜在交易机会，并实时监控和执行交易操作。

## 系统特点

- 使用Go语言实现核心交易引擎，确保高性能和低延迟
- 使用Python进行数据分析和技术指标计算
- 支持多数据源配置和健康检查
- 自定义技术指标和交易策略
- 实时监控和自动交易执行
- 完整的交易日志和报告功能
- 系统性能监控和异常告警
- 安全性和合规性保障

## 系统架构

系统采用模块化设计，主要包括以下模块：

1. 数据源管理模块
2. 技术指标筛选模块
3. 核心交易引擎模块
4. 交易日志管理模块
5. 数据分析和辅助模块
6. 系统监控和告警模块
7. 安全性模块
8. 合规性模块

## 目录结构

```
qhft-system/
├── cmd/                # 命令行工具和服务入口
│   ├── api/            # API服务
│   └── worker/         # 后台工作进程
├── pkg/                # Go语言包
│   ├── config/         # 配置管理
│   ├── datasource/     # 数据源管理
│   ├── indicators/     # 技术指标计算
│   ├── trading/        # 交易引擎
│   ├── logger/         # 日志管理
│   ├── monitoring/     # 系统监控
│   ├── security/       # 安全性功能
│   ├── compliance/     # 合规性功能
│   ├── database/       # 数据库操作
│   └── models/         # 数据模型
├── python/             # Python模块
│   ├── analysis/       # 数据分析
│   ├── backtesting/    # 策略回测
│   ├── optimization/   # 策略优化
│   └── visualization/  # 数据可视化
├── web/                # Web界面
│   ├── frontend/       # 前端界面
│   └── backend/        # 后端API
└── docs/               # 文档
```

## 模块说明

### 日志管理模块 (pkg/logger)

日志管理模块提供了全面的日志记录功能，包括系统日志和交易日志：

#### 系统日志功能
- 支持多种日志级别：Debug、Info、Warn、Error、Fatal
- 支持多种输出格式：文本格式和JSON格式
- 支持多种输出目标：控制台、文件或两者同时
- 支持日志文件轮转和压缩
- 支持上下文日志，可添加额外字段
- 提供全局默认日志记录器和自定义日志记录器

#### 交易日志功能
- 记录买入、卖出、持仓变动等交易操作
- 按日期组织交易日志，便于查询
- 支持导出交易日志到Excel文件
- 支持记录每日交易汇总数据
- 提供交易统计和分析功能

#### 使用示例
```go
// 初始化系统日志
logConfig := logger.LogConfig{
    Level:      logger.LogLevelInfo,
    Format:     logger.LogFormatText,
    Output:     logger.LogOutputBoth,
    FilePath:   "logs/app.log",
    MaxSizeMB:  100,
    MaxBackups: 5,
    MaxAgeDays: 30,
    Compress:   true,
}
logger.InitDefaultLogger(logConfig)
log := logger.GetDefaultLogger()

// 记录日志
log.Info("系统启动成功")
log.WithField("module", "api").Info("API服务启动")

// 初始化交易日志
tradeLogger := logger.GetDefaultTradeLogger()

// 记录买入交易
buyEntry := logger.TradeLogEntry{
    Symbol:     "AAPL",
    Quantity:   100,
    Price:      150.25,
    Strategy:   "均线突破",
}
tradeLogger.LogBuy(buyEntry)
```

## 安装要求

### Go开发环境
- Go 1.16+
- 依赖管理：go modules

### Python开发环境
- Python 3.8+
- 依赖包：pandas, numpy, ta-lib, sklearn等

### 数据库
- PostgreSQL 12+

### API密钥
- 需要Polygon.io API密钥

## 快速开始

1. 克隆仓库
```bash
git clone https://github.com/yourusername/qhft-system.git
cd qhft-system
```

2. 安装Go依赖
```bash
go mod tidy
```

3. 安装Python依赖
```bash
cd python
pip install -r requirements.txt
```

4. 配置数据源
```bash
cp config.example.yaml config.yaml
# 编辑config.yaml，添加你的Polygon.io API密钥
```

5. 启动系统
```bash
go run cmd/api/main.go
```

## 许可证

MIT
