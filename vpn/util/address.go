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
		logger.Error("Invalid usable subnet in config", "subnet", subnet)
		return err
	}
	return nil
}

func NextAvailableAddress() (string, error) {
	usedAddresses, err := db.GetAllAddresses()
	slog.Debug("Used addresses from database", "used_addresses", usedAddresses)
	if err != nil {
		slog.Error("Failed to get all used addresses from database", "error", err)
		return "", err
	}

	dbTrie := ipaddr.NewTrie[*ipaddr.IPAddress]()
	for _, s := range usedAddresses {
		addr := ipaddr.NewIPAddressString(s).GetAddress()
		dbTrie.Add(addr)
	}

	iSubnet := ipaddr.NewIPAddressString(subnet).GetAddress()
	iterator := iSubnet.SetPrefixLen(32).PrefixIterator()
	for iterator.HasNext() {
		addr := iterator.Next()

		tmpAddr := addr.SetPrefixLen(24)
		if tmpAddr.IsMaxHost() || tmpAddr.IsZeroHost() ||
			iSubnet.GetUpper().Increment(-1).Equal(tmpAddr) {
			continue
		}

		if !dbTrie.ElementContains(addr) {
			logger.Debug("Found available address", "address", addr.String())
			return addr.String(), nil
		}

		println(addr.String())
	}

	return "", err
}
