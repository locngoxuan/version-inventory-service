package main

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.SugaredLogger

func initializeLogger() {
	encConfig := zap.NewProductionEncoderConfig()
	encConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewJSONEncoder(encConfig)

	var output zapcore.WriteSyncer = os.Stdout
	core := zapcore.NewCore(
		encoder,
		output,
		zapcore.InfoLevel,
	)
	log := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.DPanicLevel))
	defer func() {
		_ = log.Sync()
	}()
	logger = log.Sugar()
}
