package utils

import (
	"errors"
	"log/slog"
	"net"
	"samuelemusiani/sasso/router/config"
	"samuelemusiani/sasso/router/db"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

var (
	cNetwork config.Network
	logger   *slog.Logger

	ErrPrefixTooLarge = errors.New("New subnet prefix too large, must be <= 30")
	ErrNoAvailable    = errors.New("No available subnet found")
)

func Init(l *slog.Logger, c config.Network) error {
	logger = l
	cNetwork = c

	_, n, err := net.ParseCIDR(c.UsableSubnet)
	if err != nil {
		logger.Error("Invalid usable subnet in config", "subnet", c.UsableSubnet)
		return err
	}

	if c.NewSubnetPrefix > 30 {
		logger.Error("New subnet prefix too large, must be <= 30", "prefix", c.NewSubnetPrefix)
		return ErrPrefixTooLarge
	}
	ones, _ := n.Mask.Size()
	if c.NewSubnetPrefix < ones {
		logger.Error("New subnet prefix too small, must be >= usable subnet prefix", "prefix", c.NewSubnetPrefix, "usable_subnet", c.UsableSubnet)
		return ErrPrefixTooLarge
	}

	return nil
}

func NextAvailableSubnet() (string, error) {
	usedSubnets, err := db.GetAllUsedSubnets()
	logger.Debug("Used subnets from database", "used_subnets", usedSubnets)
	if err != nil {
		logger.Error("Failed to get all used subnets from database", "error", err)
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
		logger.Error("Invalid subnet", "subnet", subnet)
		return "", errors.New("Invalid subnet")
	}

	return s.GetUpper().Increment(-1).String(), nil
}

func GetBroadcastAddressFromSubnet(subnet string) (string, error) {
	s := ipaddr.NewIPAddressString(subnet).GetAddress()
	if s == nil {
		logger.Error("Invalid subnet", "subnet", subnet)
		return "", errors.New("Invalid subnet")
	}

	return s.GetUpper().String(), nil
}
