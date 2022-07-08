package logging

import "go.uber.org/zap"

func GetLogger(dev bool) *zap.SugaredLogger {
	var log *zap.Logger
	if dev {
		log, _ = zap.NewDevelopment()
	} else {
		log, _ = zap.NewProduction()
	}
	defer log.Sync()
	return log.Sugar()
}
