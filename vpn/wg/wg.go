package wg

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"samuelemusiani/sasso/vpn/config"
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

func Init(l *slog.Logger, config *config.Wireguard, configIN *config.WBInterfaceName) {
	logger = l
	c = config
	interfaceName = configIN.InterfaceName
}

type WGInterface struct {
	Address    string
	PrivateKey string
	PublicKey  string
	Subnet     string
}

func NewWGConfig(address, subnet string) (*WGInterface, error) {
	privateKey, publicKey, err := genKeys()
	if err != nil {
		logger.With("err", err).Error("Error generating keys")
		return nil, err
	}
	logger.Info("Generated keys", "privateKey", privateKey, "publicKey", publicKey)
	return &WGInterface{address, privateKey, publicKey, subnet}, nil
}

func (WG *WGInterface) String() string {
	return fmt.Sprintf(fileTemplate, WG.Address, WG.PrivateKey, c.PublicKey, c.Endpoint, WG.Subnet, c.Subnet)
}

func executeCommand(command string, args ...string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(command, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func executeCommandWithStdin(stdin io.Reader, command string, args ...string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(command, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = stdin
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func CreateInterface(i *WGInterface) error {
	stdout, stderr, err := executeCommand("wg", "set", "sasso", "peer", c.PublicKey, "allowed-ips", i.Address)
	if err != nil {
		logger.With("err", err, "stdout", stdout, "stderr", stderr).Error("Error creating WireGuard interface")
		return err
	}
	logger.Info("WireGuard interface created", "stdout", stdout, "stderr", stderr)
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
