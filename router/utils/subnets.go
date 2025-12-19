package utils

import (
	"errors"
	"log/slog"
	"net"

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

func Init(l *slog.Logger, c config.Network) error {
	logger = l
	cNetwork = c

	_, n, err := net.ParseCIDR(c.UsableSubnet)
	if err != nil {
		logger.Error("invalid usable subnet in config", "subnet", c.UsableSubnet)
		return err
	}

	if c.NewSubnetPrefix > 30 {
		logger.Error("new subnet prefix too large, must be <= 30", "prefix", c.NewSubnetPrefix)
		return ErrPrefixTooLarge
	}
	ones, _ := n.Mask.Size()
	if c.NewSubnetPrefix < ones {
		logger.Error("new subnet prefix too small, must be >= usable subnet prefix", "prefix", c.NewSubnetPrefix, "usable_subnet", c.UsableSubnet)
		return ErrPrefixTooLarge
	}

	return nil
}

func NextAvailableSubnet() (string, error) {
	usedSubnets, err := db.GetAllUsedSubnets()
	logger.Debug("used subnets from database", "used_subnets", usedSubnets)
	if err != nil {
		logger.Error("failed to get all used subnets from database", "error", err)
		return "", err
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

func GatewayAddressFromSubnet(subnet string) (string, error) {
	s := ipaddr.NewIPAddressString(subnet).GetAddress()
	if s == nil {
		logger.Error("invalid subnet", "subnet", subnet)
		return "", errors.New("invalid subnet")
	}

	return s.GetUpper().Increment(-1).String(), nil
}

func GetBroadcastAddressFromSubnet(subnet string) (string, error) {
	s := ipaddr.NewIPAddressString(subnet).GetAddress()
	if s == nil {
		logger.Error("invalid subnet", "subnet", subnet)
		return "", errors.New("invalid subnet")
	}

	return s.GetUpper().String(), nil
}
