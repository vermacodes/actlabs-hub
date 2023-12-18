package logger

import (
	"os"
	"strconv"

	"golang.org/x/exp/slog"
)

func SetupLogger() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		slog.Info("LOG_LEVEL not set will default to 0")
		logLevel = "0"
	}

	logLevelInt, err := strconv.Atoi(logLevel)
	if err != nil {
		slog.Error("Error converting LOG_LEVEL to int will default to 0", err)
		logLevelInt = 0
	}

	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.Level(logLevelInt),
	}

	slogHandler := slog.NewTextHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(slogHandler))
}
