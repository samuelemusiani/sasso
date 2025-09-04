package wg

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
)

var (
	pubKey    string = "qjYzM9d9hoaw1L5sDk7DjjdNcfX0UVrkoF7Vuztzajw="
	endpoint  string = "130.136.201.124:51820"
	vpnSubnet string = "10.253.0.0/24"

	fileTemplate string = `[Interface]
Address = %s
PrivateKey = %s

[Peer]
PublicKey = %s
Endpoint = %s
AllowedIps = %s, %s`
)

func NewWGConfig(address, subnet string) (string, error) {
	privateKey, publicKey, err := genKeys()
	if err != nil {
		slog.Error("Error generating keys:", err)
		return "", err
	}
	slog.Info("Generated keys", "privateKey", privateKey, "publicKey", publicKey)
	return fmt.Sprintf(fileTemplate, address, privateKey, pubKey, endpoint, subnet, vpnSubnet), nil
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

func genKeys() (string, string, error) {
	privateKey, stderr, err := executeCommand("wg", "genkey")
	if err != nil {
		slog.With("err", err, "stderr", stderr).Error("Error generating private key")
		return "", "", err
	}
	publicKey, stderr, err := executeCommandWithStdin(strings.NewReader(privateKey), "wg", "pubkey")
	if err != nil {
		slog.With("err", err, "stderr", stderr).Error("Error generating public key")
		return "", "", err
	}
	return strings.TrimSuffix(privateKey, "\n"), strings.TrimSuffix(publicKey, "\n"), nil
}
