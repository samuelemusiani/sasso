package dns

import (
	"fmt"
	"log/slog"
	"samuelemusiani/sasso/server/config"

	"github.com/seancfoley/ipaddress-go/ipaddr"
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
	// DTODO: Add DNS configuration checks here. Like if the address of the dns is empty or an invalid format.
	ip := ipaddr.NewIPAddressString(config.DnsServer).GetAddress()
	if ip == nil {
		return fmt.Errorf("DNS server address is not a valid IP address")
	}

	return nil
}
