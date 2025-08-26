package utils

import (
	"errors"
	"log/slog"
	"net"
	"samuelemusiani/sasso/router/config"
	"samuelemusiani/sasso/router/db"
	"sync"
	"time"

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
		logger.With("subnet", c.UsableSubnet).Error("Invalid usable subnet in config")
		return err
	}

	if c.NewSubnetPrefix > 30 {
		logger.With("prefix", c.NewSubnetPrefix).Error("New subnet prefix too large, must be <= 30")
		return ErrPrefixTooLarge
	}
	ones, _ := n.Mask.Size()
	if c.NewSubnetPrefix < ones {
		logger.With("prefix", c.NewSubnetPrefix, "usable_subnet", c.UsableSubnet).Error("New subnet prefix too small, must be >= usable subnet prefix")
		return ErrPrefixTooLarge
	}

	return nil
}

var (
	// Temporary in memory trie to store newly allocated subnets that are not
	// yet commited in the database
	//
	// A trie is used to allow efficient search of subnets
	trieNewSubnets = ipaddr.NewTrie[*ipaddr.IPAddress]()

	// Timestamp of the last modification of the trie. If more than 1 minute
	// has passed since the last modification, the trie is cleared to avoid
	// stale data. 1 minute should be enough to commit the new subnet
	// to the database
	lastModified = time.Time{}

	// Mutex to protect access to the trie and lastModified since multiple
	// goroutines may call NextAvailableSubnet concurrently (eg. multiple API
	// requests)
	mutex = sync.Mutex{}
)

func NextAvailableSubnet() (string, error) {
	mutex.Lock()
	defer mutex.Unlock()

	if lastModified.Add(1 * time.Minute).Before(time.Now()) {
		trieNewSubnets = ipaddr.NewTrie[*ipaddr.IPAddress]()
		lastModified = time.Now()
	}

	usedSubnets, err := db.GetAllSubnets()
	if err != nil {
		logger.With("error", err).Error("Failed to get all used subnets from database")
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
		if !dbTrie.ElementContains(n) && !trieNewSubnets.ElementContains(n) {
			trieNewSubnets.Add(n)
			logger.Debug("Found available subnet", "subnet", n.String())
			return n.String(), nil
		}
	}

	return "", ErrNoAvailable
}

func GatewayAddressFromSubnet(subnet string) (string, error) {
	s := ipaddr.NewIPAddressString(subnet).GetAddress()
	if s == nil {
		logger.With("subnet", subnet).Error("Invalid subnet")
		return "", errors.New("Invalid subnet")
	}

	return s.GetUpper().Increment(-1).String(), nil
}

func GetBroadcastAddressFromSubnet(subnet string) (string, error) {
	s := ipaddr.NewIPAddressString(subnet).GetAddress()
	if s == nil {
		logger.With("subnet", subnet).Error("Invalid subnet")
		return "", errors.New("Invalid subnet")
	}

	return s.GetUpper().String(), nil
}
