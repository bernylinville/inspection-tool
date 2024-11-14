package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitLogger 初始化日志配置
func InitLogger(debug bool) {
	// 设置日志格式
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}

	// 设置日志级别
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// 设置日志输出
	log.Logger = log.Output(output)
}

// Debug 输出调试日志
func Debug() *zerolog.Event {
	return log.Debug()
}

// Info 输出信息日志
func Info() *zerolog.Event {
	return log.Info()
}

// Warn 输出警告日志
func Warn() *zerolog.Event {
	return log.Warn()
}

// Error 输出错误日志
func Error() *zerolog.Event {
	return log.Error()
}
