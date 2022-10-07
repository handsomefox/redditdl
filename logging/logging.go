// logging is a wrapper around uber/zap logger
// to quickly get a configured logger.
package logging

import (
	"github.com/mattn/go-colorable"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

// Get returns a new logger configured depending on the development bool.
func Get(development bool) *zap.SugaredLogger {
	if log != nil {
		return log.Sugar()
	}
	var (
		cfg   zapcore.EncoderConfig
		level zapcore.Level
	)
	switch development {
	case true:
		cfg = zap.NewDevelopmentEncoderConfig()
		level = zapcore.DebugLevel
	case false:
		cfg = zap.NewProductionEncoderConfig()
		level = zapcore.InfoLevel
	}

	cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.EncodeTime = zapcore.RFC3339TimeEncoder

	log = zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(cfg),
		zapcore.AddSync(colorable.NewColorableStdout()),
		level,
	))
	return log.Sugar()
}
