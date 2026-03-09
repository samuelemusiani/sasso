package main

import (
	"fmt"

	"github.com/seancfoley/ipaddress-go/ipaddr"
	"samuelemusiani/sasso/vpn/db"
)

func nextAvailableAddress(subnet string, addresses []string) (string, error) {
	usedAddresses, err := db.GetAllAddresses()
	if err != nil {
		return "", fmt.Errorf("failed to get used addresses from database: %w", err)
	}

	usedAddresses = append(usedAddresses, addresses...)

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
			return addr.String(), nil
		}
	}

	return "", err
}
