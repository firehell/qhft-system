package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/natefinch/lumberjack"
)

// defaultLogger 是默认的日志实现
type defaultLogger struct {
	mu        sync.Mutex
	config    LogConfig
	context   LogContext
	writer    io.Writer
	fileLog   *lumberjack.Logger
	stdoutLog io.Writer
}

// NewLogger 创建一个新的日志记录器
func NewLogger(config LogConfig) (Logger, error) {
	logger := &defaultLogger{
		config:  config,
		context: make(LogContext),
	}

	// 如果需要文件日志，初始化文件日志记录器
	if config.Output == LogOutputFile || config.Output == LogOutputBoth {
		// 确保日志目录存在
		if err := os.MkdirAll(filepath.Dir(config.FilePath), 0755); err != nil {
			return nil, fmt.Errorf("创建日志目录失败: %v", err)
		}

		logger.fileLog = &lumberjack.Logger{
			Filename:   config.FilePath,
			MaxSize:    config.MaxSizeMB,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAgeDays,
			Compress:   config.Compress,
		}
	}

	// 设置输出writer
	switch config.Output {
	case LogOutputConsole:
		logger.writer = os.Stdout
		logger.stdoutLog = os.Stdout
	case LogOutputFile:
		logger.writer = logger.fileLog
	case LogOutputBoth:
		logger.stdoutLog = os.Stdout
		logger.writer = io.MultiWriter(os.Stdout, logger.fileLog)
	default:
		logger.writer = os.Stdout
		logger.stdoutLog = os.Stdout
	}

	return logger, nil
}

// log 输出日志
func (l *defaultLogger) log(level LogLevel, msg string, args ...interface{}) {
	if !l.shouldLog(level) {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 格式化消息
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	// 创建日志条目
	entry := LogEntry{
		Level:     level,
		Message:   msg,
		Timestamp: time.Now(),
		Context:   l.context,
	}

	// 添加源代码位置信息
	if level == LogLevelError || level == LogLevelFatal {
		_, file, line, ok := runtime.Caller(2)
		if ok {
			entry.File = file
			entry.Line = line
		}
	}

	// 根据格式输出日志
	if l.config.Format == LogFormatJSON {
		l.writeJSONLog(entry)
	} else {
		l.writeTextLog(entry)
	}

	// 如果是fatal级别，程序终止
	if level == LogLevelFatal {
		os.Exit(1)
	}
}

// writeJSONLog 以JSON格式输出日志
func (l *defaultLogger) writeJSONLog(entry LogEntry) {
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "无法序列化日志条目: %v\n", err)
		return
	}
	fmt.Fprintln(l.writer, string(jsonBytes))
}

// writeTextLog 以文本格式输出日志
func (l *defaultLogger) writeTextLog(entry LogEntry) {
	// 基本日志格式：[时间] [级别] 消息
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05.000")
	levelStr := fmt.Sprintf("%-5s", entry.Level)
	logLine := fmt.Sprintf("[%s] [%s] %s", timestamp, levelStr, entry.Message)

	// 添加源代码位置信息（如果有）
	if entry.File != "" {
		logLine += fmt.Sprintf(" (%s:%d)", filepath.Base(entry.File), entry.Line)
	}

	// 添加上下文信息（如果有）
	if len(entry.Context) > 0 {
		contextStr, _ := json.Marshal(entry.Context)
		logLine += fmt.Sprintf(" %s", string(contextStr))
	}

	fmt.Fprintln(l.writer, logLine)
}

// shouldLog 检查是否应该记录这个级别的日志
func (l *defaultLogger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		LogLevelDebug: 0,
		LogLevelInfo:  1,
		LogLevelWarn:  2,
		LogLevelError: 3,
		LogLevelFatal: 4,
	}

	return levels[level] >= levels[l.config.Level]
}

// Debug 记录debug级别日志
func (l *defaultLogger) Debug(msg string, args ...interface{}) {
	l.log(LogLevelDebug, msg, args...)
}

// Info 记录info级别日志
func (l *defaultLogger) Info(msg string, args ...interface{}) {
	l.log(LogLevelInfo, msg, args...)
}

// Warn 记录warn级别日志
func (l *defaultLogger) Warn(msg string, args ...interface{}) {
	l.log(LogLevelWarn, msg, args...)
}

// Error 记录error级别日志
func (l *defaultLogger) Error(msg string, args ...interface{}) {
	l.log(LogLevelError, msg, args...)
}

// Fatal 记录fatal级别日志并退出程序
func (l *defaultLogger) Fatal(msg string, args ...interface{}) {
	l.log(LogLevelFatal, msg, args...)
}

// WithField 添加一个字段到上下文
func (l *defaultLogger) WithField(key string, value interface{}) Logger {
	newLogger := &defaultLogger{
		config:    l.config,
		writer:    l.writer,
		fileLog:   l.fileLog,
		stdoutLog: l.stdoutLog,
		context:   make(LogContext),
	}

	// 复制现有上下文
	for k, v := range l.context {
		newLogger.context[k] = v
	}

	// 添加新字段
	newLogger.context[key] = value

	return newLogger
}

// WithFields 添加多个字段到上下文
func (l *defaultLogger) WithFields(fields map[string]interface{}) Logger {
	newLogger := &defaultLogger{
		config:    l.config,
		writer:    l.writer,
		fileLog:   l.fileLog,
		stdoutLog: l.stdoutLog,
		context:   make(LogContext),
	}

	// 复制现有上下文
	for k, v := range l.context {
		newLogger.context[k] = v
	}

	// 添加新字段
	for k, v := range fields {
		newLogger.context[k] = v
	}

	return newLogger
}

// WithContext 设置完整的上下文
func (l *defaultLogger) WithContext(ctx LogContext) Logger {
	newLogger := &defaultLogger{
		config:    l.config,
		writer:    l.writer,
		fileLog:   l.fileLog,
		stdoutLog: l.stdoutLog,
		context:   make(LogContext),
	}

	// 复制现有上下文
	for k, v := range l.context {
		newLogger.context[k] = v
	}

	// 添加新上下文
	for k, v := range ctx {
		newLogger.context[k] = v
	}

	return newLogger
}

// SetLevel 设置日志级别
func (l *defaultLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Level = level
}

// GetLevel 获取当前日志级别
func (l *defaultLogger) GetLevel() LogLevel {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.config.Level
}

// Close 关闭日志记录器
func (l *defaultLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.fileLog != nil {
		return l.fileLog.Close()
	}
	return nil
}

// 全局默认日志记录器
var (
	defaultLoggerInstance Logger
	once                  sync.Once
)

// GetDefaultLogger 获取全局默认日志记录器
func GetDefaultLogger() Logger {
	once.Do(func() {
		config := LogConfig{
			Level:      LogLevelInfo,
			Format:     LogFormatText,
			Output:     LogOutputConsole,
			FilePath:   "logs/app.log",
			MaxSizeMB:  100,
			MaxBackups: 5,
			MaxAgeDays: 30,
			Compress:   true,
		}

		logger, err := NewLogger(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "初始化默认日志记录器失败: %v\n", err)
			os.Exit(1)
		}
		defaultLoggerInstance = logger
	})

	return defaultLoggerInstance
}

// InitDefaultLogger 初始化全局默认日志记录器
func InitDefaultLogger(config LogConfig) {
	logger, err := NewLogger(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化默认日志记录器失败: %v\n", err)
		os.Exit(1)
	}
	defaultLoggerInstance = logger
}

// Debug 记录debug级别日志（使用默认日志记录器）
func Debug(msg string, args ...interface{}) {
	GetDefaultLogger().Debug(msg, args...)
}

// Info 记录info级别日志（使用默认日志记录器）
func Info(msg string, args ...interface{}) {
	GetDefaultLogger().Info(msg, args...)
}

// Warn 记录warn级别日志（使用默认日志记录器）
func Warn(msg string, args ...interface{}) {
	GetDefaultLogger().Warn(msg, args...)
}

// Error 记录error级别日志（使用默认日志记录器）
func Error(msg string, args ...interface{}) {
	GetDefaultLogger().Error(msg, args...)
}

// Fatal 记录fatal级别日志并退出程序（使用默认日志记录器）
func Fatal(msg string, args ...interface{}) {
	GetDefaultLogger().Fatal(msg, args...)
} 