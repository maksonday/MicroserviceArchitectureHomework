package logging

import (
	"auth/config"
	"log"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var onceLog sync.Once

func Init(appName string, config *config.Config) {
	onceLog.Do(func() {
		zapConfig := zap.NewProductionConfig()
		lvl, err := zapcore.ParseLevel(config.LogLevel)
		if err != nil {
			log.Fatalf("invalid log level: %v", err)
		}

		zapConfig.Level = zap.NewAtomicLevelAt(lvl)
		zapConfig.OutputPaths = []string{config.LogFile}
		zapConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)

		logger, err := zapConfig.Build()
		if err != nil {
			log.Fatalf("failed to create logger: %v", err)
		}

		logger = logger.Named(appName)
		zap.ReplaceGlobals(logger)
		zap.RedirectStdLog(logger)
	})
}
