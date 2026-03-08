package main

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/seancfoley/ipaddress-go/ipaddr"
	"samuelemusiani/sasso/router/config"
	"samuelemusiani/sasso/router/db"
)

var (
	cNetwork config.Network
	logger   *slog.Logger

	ErrPrefixTooLarge = errors.New("new subnet prefix too large, must be <= 30")
	ErrNoAvailable    = errors.New("no available subnet found")
)

// nextAvailableSubnet finds the next available subnets that are
// not in the database and in the subnets slice passed as argument.
func nextAvailableSubnet(subnets []string) (string, error) {
	usedSubnets, err := db.GetAllUsedSubnets()
	logger.Debug("used subnets from database", "used_subnets", usedSubnets)

	usedSubnets = append(usedSubnets, subnets...)

	if err != nil {
		return "", fmt.Errorf("failed to get used subnets from database: %w", err)
	}

	dbTrie := ipaddr.NewTrie[*ipaddr.IPAddress]()

	for _, s := range usedSubnets {
		addr := ipaddr.NewIPAddressString(s).GetAddress()
		dbTrie.Add(addr)
	}

	subnet := ipaddr.NewIPAddressString(cNetwork.UsableSubnet).GetAddress()

	iterator := subnet.SetPrefixLen(cNetwork.NewSubnetPrefix).PrefixIterator()
	for iterator.HasNext() {
		n := iterator.Next()
		if !dbTrie.ElementContains(n) {
			logger.Debug("Found available subnet", "subnet", n.String())

			return n.String(), nil
		}
	}

	return "", ErrNoAvailable
}

func gatewayAddressFromSubnet(subnet string) (string, error) {
	s := ipaddr.NewIPAddressString(subnet).GetAddress()
	if s == nil {
		return "", fmt.Errorf("invalid subnet: %s", subnet)
	}

	return s.GetUpper().Increment(-1).String(), nil
}

func getBroadcastAddressFromSubnet(subnet string) (string, error) {
	s := ipaddr.NewIPAddressString(subnet).GetAddress()
	if s == nil {
		return "", fmt.Errorf("invalid subnet: %s", subnet)
	}

	return s.GetUpper().String(), nil
}
