package wg

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"samuelemusiani/sasso/vpn/config"
)

var (
	c      *config.Wireguard
	logger *slog.Logger
)

func Init(l *slog.Logger, conf *config.Wireguard) error {
	// As executeCommand requires logger, set it first
	logger = l

	// Check permission for wg command and its existence
	_, stderr, err := executeCommand("--version")
	if err != nil {
		return fmt.Errorf("wg command not found or not executable: %w, stderr: %s", err, stderr)
	}

	err = checkConfig(conf)
	if err != nil {
		return err
	}

	c = conf

	return nil
}

func checkConfig(c *config.Wireguard) error {
	if c.PublicKey == "" {
		return errors.New("wireguard public key is empty")
	}

	if c.Endpoint == "" {
		return errors.New("wireguard endpoint is empty")
	}

	if c.VPNSubnet == "" {
		return errors.New("wireguard vpn subnet is empty")
	}

	if c.VMsSubnet == "" {
		return errors.New("wireguard vms subnet is empty")
	}

	if c.Interface == "" {
		return errors.New("wireguard interface name is empty")
	}

	// Public key is base64 encoded, check it
	_, err := base64.StdEncoding.DecodeString(c.PublicKey)
	if err != nil {
		return fmt.Errorf("wireguard public key is not valid base64: %w", err)
	}

	// Endpoint could be an IP or a domain name
	rDomain := regexp.MustCompile(`^([a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,}$`) // regex for domain name

	// As endpoint could be in format domain:port, split by ':' and check only the domain part
	colonIndex := strings.LastIndex(c.Endpoint, ":")
	if colonIndex == -1 {
		return fmt.Errorf("wireguard endpoint (%s) must include port", c.Endpoint)
	}

	domainPart := c.Endpoint[:colonIndex]
	portPart := c.Endpoint[colonIndex+1:]
	// Check port is valid
	_, err = strconv.ParseUint(portPart, 10, 16)
	if err != nil {
		return fmt.Errorf("wireguard endpoint (%s) has invalid port: %w", c.Endpoint, err)
	}

	if domainPart == "" {
		return fmt.Errorf("wireguard endpoint (%s) has empty domain or IP part", c.Endpoint)
	}

	if !rDomain.MatchString(domainPart) {
		// Could be an IP, check it
		shouldBeV6 := false

		if domainPart[0] == '[' && domainPart[len(domainPart)-1] == ']' {
			// IPv6 in brackets, remove them
			domainPart = domainPart[1 : len(domainPart)-1]
			shouldBeV6 = true
		}

		ip := net.ParseIP(domainPart)

		if ip == nil {
			return fmt.Errorf("wireguard endpoint (%s) is not a valid IP or domain name", domainPart)
		}

		if ip.To16() == nil && shouldBeV6 {
			return fmt.Errorf("wireguard endpoint (%s) is not a valid IPv6 address", domainPart)
		}
	}
	// Is a domain name, ok

	// Check VPNSubnet and VMsSubnet are valid CIDRs
	for _, cidr := range []string{c.VPNSubnet, c.VMsSubnet} {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return fmt.Errorf("wireguard subnet %s is not a valid CIDR: %w", cidr, err)
		}
	}

	// Check interface exists
	_, stderr, err := executeCommand("show", c.Interface)
	if err != nil {
		return fmt.Errorf("wireguard interface %s does not exist: %w, stderr: %s", c.Interface, err, stderr)
	}

	return nil
}

type Peer struct {
	IP              string
	PeerPrivateKey  string
	ServerPublicKey string
	Endpoint        string
	AllowedIPs      []string
}

func (p Peer) Equal(other Peer) bool {
	if p.IP != other.IP ||
		p.PeerPrivateKey != other.PeerPrivateKey ||
		p.ServerPublicKey != other.ServerPublicKey ||
		p.Endpoint != other.Endpoint {
		return false
	}

	if len(p.AllowedIPs) != len(other.AllowedIPs) {
		return false
	}

	for i := range p.AllowedIPs {
		if !slices.Contains(other.AllowedIPs, p.AllowedIPs[i]) {
			return false
		}
	}

	return true
}

// NewPeer creates a new Wireguard peer with a new key pair and the given IP
// address. This function does not add the peer to the Wireguard interface, it
// only creates the struct with the necessary information
func NewPeer(address string) (*Peer, error) {
	privateKey, publicKey, err := genKeys()
	if err != nil {
		logger.Error("Error generating keys", "err", err)

		return nil, err
	}

	logger.Info("Generated keys", "privateKey", privateKey, "publicKey", publicKey)

	return &Peer{
		IP:              address,
		PeerPrivateKey:  privateKey,
		ServerPublicKey: publicKey,
		Endpoint:        c.Endpoint,
		AllowedIPs:      []string{c.VMsSubnet, c.VPNSubnet},
	}, nil
}

func executeCommand(args ...string) (stdout string, stderr string, err error) {
	logger.Debug("Executing wg command", "args", args)

	var (
		stdoutBuff bytes.Buffer
		stderrBuff bytes.Buffer
	)

	cmd := exec.Command("wg", args...)
	cmd.Stdout = &stdoutBuff
	cmd.Stderr = &stderrBuff
	err = cmd.Run()

	return stdoutBuff.String(), stderrBuff.String(), err
}

func executeCommandWithStdin(stdin io.Reader, command string, args ...string) (stdout string, stderr string, err error) {
	logger.Debug("Executing command with stdin", "command", command, "args", args)

	var (
		stdoutBuff bytes.Buffer
		stderrBuff bytes.Buffer
	)

	cmd := exec.Command(command, args...)
	cmd.Stdout = &stdoutBuff
	cmd.Stderr = &stderrBuff
	cmd.Stdin = stdin
	err = cmd.Run()

	return stdoutBuff.String(), stderrBuff.String(), err
}

func CreatePeer(i *Peer) error {
	stdout, stderr, err := executeCommand("set", c.Interface, "peer", i.ServerPublicKey, "allowed-ips", i.IP)
	if err != nil {
		logger.Error("Error creating WireGuard peer", "err", err, "stdout", stdout, "stderr", stderr)

		return err
	}

	logger.Info("WireGuard peer created", "stdout", stdout, "stderr", stderr)

	return nil
}

func DeletePeer(i *Peer) error {
	stdout, stderr, err := executeCommand("set", c.Interface, "peer", i.ServerPublicKey, "remove")
	if err != nil {
		logger.Error("Error deleting WireGuard peer", "err", err, "stdout", stdout, "stderr", stderr)

		return err
	}

	logger.Info("WireGuard peer deleted", "stdout", stdout, "stderr", stderr)

	return nil
}

func UpdatePeer(i *Peer) error {
	err := DeletePeer(i)
	if err != nil {
		return err
	}

	err = CreatePeer(i)
	if err != nil {
		return err
	}

	return nil
}

func genKeys() (private string, public string, err error) {
	privateKey, stderr, err := executeCommand("genkey")
	if err != nil {
		logger.Error("Error generating private key", "err", err, "stderr", stderr)

		return "", "", err
	}

	publicKey, stderr, err := executeCommandWithStdin(strings.NewReader(privateKey), "wg", "pubkey")
	if err != nil {
		logger.Error("Error generating public key", "err", err, "stderr", stderr)

		return "", "", err
	}

	return strings.TrimSuffix(privateKey, "\n"), strings.TrimSuffix(publicKey, "\n"), nil
}

func ParsePeers() (map[string]Peer, error) {
	stdout, stderr, err := executeCommand("show", c.Interface, "dump")
	if err != nil {
		logger.Error("Error dumping peers", "err", err, "stderr", stderr)

		return nil, err
	}

	peers := make(map[string]Peer)

	lines := strings.Split(stdout, "\n")
	for i, l := range lines {
		if i == 0 {
			continue // fist is the interface
		}

		l = strings.TrimSpace(l)
		fields := strings.Split(l, "\t")

		if len(fields) == 1 {
			continue // skip empty lines
		}

		if len(fields) < 4 {
			// not enough fields, error
			return nil, errors.New("not enough fields in wg show dump output")
		}

		publicKey := fields[0]
		privateKey := fields[1]
		allowedIps := fields[3]

		allowedIPs := strings.Split(allowedIps, ",")
		slices.Sort(allowedIPs)
		peers[publicKey] = Peer{
			IP:              allowedIps,
			PeerPrivateKey:  privateKey,
			ServerPublicKey: publicKey,
			Endpoint:        c.Endpoint,
			AllowedIPs:      allowedIPs,
		}
	}

	return peers, nil
}
