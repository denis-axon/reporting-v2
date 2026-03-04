package logger

import (
	"sync"

	"go.uber.org/zap"
)

var log *zap.SugaredLogger
var o sync.Once

func Get() *zap.SugaredLogger {
	o.Do(func() {
		logger, err := zap.NewProduction()
		if err != nil {
			panic(err)
		}
		log = logger.Sugar()
	})
	return log
}
