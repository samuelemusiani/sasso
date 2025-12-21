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
	"strconv"
	"strings"

	"samuelemusiani/sasso/vpn/config"
	"samuelemusiani/sasso/vpn/db"
)

var (
	c      *config.Wireguard = nil
	logger *slog.Logger      = nil

	fileTemplate string = `[Interface]
Address = %s
PrivateKey = %s

[Peer]
PublicKey = %s
Endpoint = %s
AllowedIps = %s, %s`
)

func Init(l *slog.Logger, config *config.Wireguard) error {
	// As executeCommand requires logger, set it first
	logger = l

	// Check permission for wg command and its existence
	_, stderr, err := executeWGCommand("--version")
	if err != nil {
		return fmt.Errorf("wg command not found or not executable: %w, stderr: %s", err, stderr)
	}

	err = checkConfig(config)
	if err != nil {
		return err
	}

	c = config

	return nil
}

func checkConfig(config *config.Wireguard) error {
	if config.PublicKey == "" {
		return errors.New("wireguard public key is empty")
	}

	if config.Endpoint == "" {
		return errors.New("wireguard endpoint is empty")
	}

	if config.VPNSubnet == "" {
		return errors.New("wireguard vpn subnet is empty")
	}

	if config.VMsSubnet == "" {
		return errors.New("wireguard vms subnet is empty")
	}

	if config.Interface == "" {
		return errors.New("wireguard interface name is empty")
	}

	// Public key is base64 encoded, check it
	_, err := base64.StdEncoding.DecodeString(config.PublicKey)
	if err != nil {
		return fmt.Errorf("wireguard public key is not valid base64: %w", err)
	}

	// Endpoint could be an IP or a domain name
	rDomain := regexp.MustCompile(`^([a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,}$`) // regex for domain name

	// As endpoint could be in format domain:port, split by ':' and check only the domain part
	colonIndex := strings.LastIndex(config.Endpoint, ":")
	if colonIndex == -1 {
		return fmt.Errorf("wireguard endpoint (%s) must include port", config.Endpoint)
	}

	domainPart := config.Endpoint[:colonIndex]
	portPart := config.Endpoint[colonIndex+1:]
	// Check port is valid
	_, err = strconv.ParseUint(portPart, 10, 16)
	if err != nil {
		return fmt.Errorf("wireguard endpoint (%s) has invalid port: %w", config.Endpoint, err)
	}

	if domainPart == "" {
		return fmt.Errorf("wireguard endpoint (%s) has empty domain or IP part", config.Endpoint)
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
	for _, cidr := range []string{config.VPNSubnet, config.VMsSubnet} {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return fmt.Errorf("wireguard subnet %s is not a valid CIDR: %w", cidr, err)
		}
	}

	// Check interface exists
	_, stderr, err := executeWGCommand("show", config.Interface)
	if err != nil {
		return fmt.Errorf("wireguard interface %s does not exist: %w, stderr: %s", config.Interface, err, stderr)
	}

	return nil
}

type WGPeer struct {
	Address    string
	PrivateKey string
	PublicKey  string
}

func NewWGConfig(address string) (*WGPeer, error) {
	privateKey, publicKey, err := genKeys()
	if err != nil {
		logger.Error("Error generating keys", "err", err)

		return nil, err
	}

	logger.Info("Generated keys", "privateKey", privateKey, "publicKey", publicKey)

	return &WGPeer{address, privateKey, publicKey}, nil
}

func (wg *WGPeer) String() string {
	return fmt.Sprintf(fileTemplate, wg.Address, wg.PrivateKey, c.PublicKey, c.Endpoint, c.VPNSubnet, c.VMsSubnet)
}

func executeWGCommand(args ...string) (string, string, error) {
	logger.Debug("Executing wg command", "args", args)

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	cmd := exec.Command("wg", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	return stdout.String(), stderr.String(), err
}

func executeCommandWithStdin(stdin io.Reader, command string, args ...string) (string, string, error) {
	logger.Debug("Executing command with stdin", "command", command, "args", args)

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	cmd := exec.Command(command, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = stdin
	err := cmd.Run()

	return stdout.String(), stderr.String(), err
}

func CreatePeer(i *WGPeer) error {
	stdout, stderr, err := executeWGCommand("set", c.Interface, "peer", i.PublicKey, "allowed-ips", i.Address)
	if err != nil {
		logger.Error("Error creating WireGuard peer", "err", err, "stdout", stdout, "stderr", stderr)

		return err
	}

	logger.Info("WireGuard peer created", "stdout", stdout, "stderr", stderr)

	return nil
}

func DeletePeer(i *WGPeer) error {
	stdout, stderr, err := executeWGCommand("set", c.Interface, "peer", i.PublicKey, "remove")
	if err != nil {
		logger.Error("Error deleting WireGuard peer", "err", err, "stdout", stdout, "stderr", stderr)

		return err
	}

	logger.Info("WireGuard peer deleted", "stdout", stdout, "stderr", stderr)

	return nil
}

func UpdatePeer(i *WGPeer) error {
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

func genKeys() (string, string, error) {
	privateKey, stderr, err := executeWGCommand("genkey")
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

func ParsePeers() (map[string]WGPeer, error) {
	stdout, stderr, err := executeWGCommand("show", c.Interface, "dump")
	if err != nil {
		logger.Error("Error dumping peers", "err", err, "stderr", stderr)

		return nil, err
	}

	peers := make(map[string]WGPeer)

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

		peer := WGPeer{
			Address:    allowedIps,
			PrivateKey: privateKey,
			PublicKey:  publicKey,
		}
		peers[publicKey] = peer
	}

	return peers, nil
}

func PeerFromDB(iface *db.Peer) WGPeer {
	return WGPeer{
		Address:    iface.Address,
		PrivateKey: iface.PrivateKey,
		PublicKey:  iface.PublicKey,
	}
}
