package logging

import (
	"github.com/mattn/go-colorable"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func GetLogger(dev bool) *zap.SugaredLogger {
	var log *zap.Logger
	if dev {
		cfg := zap.NewDevelopmentEncoderConfig()
		cfg.EncodeTime = zapcore.RFC3339TimeEncoder
		cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		log = zap.New(zapcore.NewCore(
			zapcore.NewConsoleEncoder(cfg),
			zapcore.AddSync(colorable.NewColorableStdout()),
			zapcore.DebugLevel,
		))

	} else {
		cfg := zap.NewProductionEncoderConfig()
		cfg.EncodeTime = zapcore.RFC3339TimeEncoder
		cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		log = zap.New(zapcore.NewCore(
			zapcore.NewConsoleEncoder(cfg),
			zapcore.AddSync(colorable.NewColorableStdout()),
			zapcore.InfoLevel,
		))
	}
	defer log.Sync()
	return log.Sugar()
}
