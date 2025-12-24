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
	c      *config.Wireguard
	logger *slog.Logger

	fileTemplate = `[Interface]
Address = %s
PrivateKey = %s

[Peer]
PublicKey = %s
Endpoint = %s
AllowedIps = %s, %s`
)

func Init(l *slog.Logger, conf *config.Wireguard) error {
	// As executeCommand requires logger, set it first
	logger = l

	// Check permission for wg command and its existence
	_, stderr, err := executeWGCommand("--version")
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
	_, stderr, err := executeWGCommand("show", c.Interface)
	if err != nil {
		return fmt.Errorf("wireguard interface %s does not exist: %w, stderr: %s", c.Interface, err, stderr)
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

func executeWGCommand(args ...string) (stdout string, stderr string, err error) {
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

func genKeys() (private string, public string, err error) {
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
