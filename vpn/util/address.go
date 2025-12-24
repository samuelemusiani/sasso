package util

import (
	"log/slog"

	"github.com/seancfoley/ipaddress-go/ipaddr"
	"samuelemusiani/sasso/vpn/db"
)

func NextAvailableAddress(subnet string) (string, error) {
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
	}

	return "", err
}
