package wg

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"samuelemusiani/sasso/vpn/config"
	"samuelemusiani/sasso/vpn/db"
	"strings"
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
	interfaceName string
)

func Init(l *slog.Logger, config *config.Wireguard, iface string) {
	logger = l
	c = config
	interfaceName = iface
}

type WGPeer struct {
	Address    string
	PrivateKey string
	PublicKey  string
}

func NewWGConfig(address string) (*WGPeer, error) {
	privateKey, publicKey, err := genKeys()
	if err != nil {
		logger.With("err", err).Error("Error generating keys")
		return nil, err
	}
	logger.Info("Generated keys", "privateKey", privateKey, "publicKey", publicKey)
	return &WGPeer{address, privateKey, publicKey}, nil
}

func (WG *WGPeer) String() string {
	return fmt.Sprintf(fileTemplate, WG.Address, WG.PrivateKey, c.PublicKey, c.Endpoint, c.VPNSubnet, c.VMsSubnet)
}

func executeCommand(command string, args ...string) (string, string, error) {
	logger.Debug("Executing command", "command", command, "args", args)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(command, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func executeCommandWithStdin(stdin io.Reader, command string, args ...string) (string, string, error) {
	logger.Debug("Executing command with stdin", "command", command, "args", args)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(command, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = stdin
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func CreatePeer(i *WGPeer) error {
	stdout, stderr, err := executeCommand("wg", "set", interfaceName, "peer", i.PublicKey, "allowed-ips", i.Address)
	if err != nil {
		logger.With("err", err, "stdout", stdout, "stderr", stderr).Error("Error creating WireGuard peer")
		return err
	}
	logger.Info("WireGuard peer created", "stdout", stdout, "stderr", stderr)
	return nil
}

func DeletePeer(i *WGPeer) error {
	stdout, stderr, err := executeCommand("wg", "set", interfaceName, "peer", i.PublicKey, "remove")
	if err != nil {
		logger.With("err", err, "stdout", stdout, "stderr", stderr).Error("Error deleting WireGuard peer")
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
	privateKey, stderr, err := executeCommand("wg", "genkey")
	if err != nil {
		logger.With("err", err, "stderr", stderr).Error("Error generating private key")
		return "", "", err
	}
	publicKey, stderr, err := executeCommandWithStdin(strings.NewReader(privateKey), "wg", "pubkey")
	if err != nil {
		logger.With("err", err, "stderr", stderr).Error("Error generating public key")
		return "", "", err
	}
	return strings.TrimSuffix(privateKey, "\n"), strings.TrimSuffix(publicKey, "\n"), nil
}

func ParsePeers() (map[string]WGPeer, error) {
	stdout, stderr, err := executeCommand("wg", "show", interfaceName, "dump")
	if err != nil {
		logger.With("err", err, "stderr", stderr).Error("Error dumping peers")
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
			return nil, fmt.Errorf("not enough fields in wg show dump output")
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
