package logging

import (
	"github.com/mattn/go-colorable"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

func GetLogger(isDev bool) *zap.SugaredLogger {
	if log != nil {
		return log.Sugar()
	}

	var (
		cfg         zapcore.EncoderConfig
		level       zapcore.Level
		encodeTime  = zapcore.RFC3339TimeEncoder
		encodeLevel = zapcore.CapitalColorLevelEncoder
	)

	if isDev {
		cfg = zap.NewDevelopmentEncoderConfig()
		level = zapcore.DebugLevel
	} else {
		cfg = zap.NewProductionEncoderConfig()
		level = zapcore.InfoLevel
	}

	cfg.EncodeLevel = encodeLevel
	cfg.EncodeTime = encodeTime

	log = zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(cfg),
		zapcore.AddSync(colorable.NewColorableStdout()),
		level,
	))

	return log.Sugar()
}
