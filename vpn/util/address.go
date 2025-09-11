package util

import (
	"log/slog"
	"net"
	"samuelemusiani/sasso/vpn/db"

	"github.com/seancfoley/ipaddress-go/ipaddr"
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
		logger.With("subnet", subnet).Error("Invalid usable subnet in config")
		return err
	}
	return nil
}
func NextAvailableAddress() (string, error) {

	usedAddresses, err := db.GetAllAddresses()
	slog.With("used_addresses", usedAddresses).Debug("Used addresses from database")
	if err != nil {
		slog.With("error", err).Error("Failed to get all used addresses from database")
		return "", err
	}

	dbTrie := ipaddr.NewTrie[*ipaddr.IPAddress]()
	for _, s := range usedAddresses {
		addr := ipaddr.NewIPAddressString(s).GetAddress()
		dbTrie.Add(addr)
	}

	iterator := ipaddr.NewIPAddressString(subnet).GetAddress().PrefixIterator()
	for iterator.HasNext() {
		n := iterator.Next()
		if !dbTrie.ElementContains(n) /* && !trieNewSubnets.ElementContains(n) */ {
			// trieNewSubnets.Add(n)
			logger.Debug("Found available address", "address", n.String())
			return n.String(), nil
		}
	}

	return "", err
}
