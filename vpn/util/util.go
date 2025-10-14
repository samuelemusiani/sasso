package util

import (
	"log/slog"
	"net"
)

var (
	subnet string
	logger *slog.Logger
)

func Init(l *slog.Logger, s string) error {
	logger = l
	subnet = s

	_, _, err := net.ParseCIDR(subnet)
	if err != nil {
		logger.Error("Invalid usable subnet in config", "subnet", subnet)
		return err
	}
	return nil
}
