package util

import (
	"log/slog"
)

var (
	logger *slog.Logger
)

func Init(l *slog.Logger) {
	logger = l
}
