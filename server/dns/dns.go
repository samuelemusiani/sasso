package dns

import (
	"log/slog"
	"samuelemusiani/sasso/server/config"
)

var (
	logger *slog.Logger = nil
	cDNS   *config.DNS  = nil
)

func Init(dnsLogger *slog.Logger, config config.DNS) error {
	logger = dnsLogger

	err := configChecks(config)
	if err != nil {
		return err
	}

	cDNS = &config

	return nil
}

func configChecks(config config.DNS) error {
	// DTODO: Add DNS configuration checks here. Like if the address of the
	// dns is empty or an invalid format.

	return nil
}
