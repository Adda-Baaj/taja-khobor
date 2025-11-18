package logger

import (
	"os"

	"github.com/Adda-Baaj/taja-khobor/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var S *zap.SugaredLogger

func Init(cfg *config.Config) (*zap.SugaredLogger, error) {
	var level zapcore.Level
	switch cfg.LogLevel {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn", "warning":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "ts"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(zapcore.Lock(os.Stdout)),
		level,
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	sugar := logger.Sugar()
	S = sugar
	return sugar, nil
}

func Close() error {
	if S == nil {
		return nil
	}
	return S.Sync()
}

func InfoObj(msg, key string, obj interface{}) {
	if S == nil {
		return
	}
	S.Desugar().Info(msg, zap.Any(key, obj))
}

func DebugObj(msg, key string, obj interface{}) {
	if S == nil {
		return
	}
	S.Desugar().Debug(msg, zap.Any(key, obj))
}

func WarnObj(msg, key string, obj interface{}) {
	if S == nil {
		return
	}
	S.Desugar().Warn(msg, zap.Any(key, obj))
}

func ErrorObj(msg, key string, obj interface{}) {
	if S == nil {
		return
	}
	S.Desugar().Error(msg, zap.Any(key, obj))
}
