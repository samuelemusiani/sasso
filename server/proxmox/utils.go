package proxmox

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/luthermonson/go-proxmox"
)

const base62Alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// EncodeBase62 encodes a uint32 into a base62 string
func EncodeBase62(num uint32) string {
	if num == 0 {
		return string(base62Alphabet[0])
	}
	var sb strings.Builder
	for num > 0 {
		remainder := num % 62
		sb.WriteByte(base62Alphabet[remainder])
		num /= 62
	}
	// reverse since we construct in reverse order
	runes := []rune(sb.String())
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// DecodeBase62 decodes a base62 string into a uint32
func DecodeBase62(s string) (uint32, error) {
	var num uint32
	for _, c := range s {
		index := strings.IndexRune(base62Alphabet, c)
		if index == -1 {
			return 0, fmt.Errorf("invalid character: %c", c)
		}
		num = num*62 + uint32(index)
	}
	return num, nil
}

type Storage struct {
	Name    string
	VMID    uint32
	File    string
	Discard bool
	Size    uint
}

var (
	ErrInvalidStorageString = errors.New("invalid storage string")
)

// Parses a string like "storage0:1011/vm-1011-disk-1.qcow2,discard=on,size=4G"
func parseStorageFromString(s string) (*Storage, error) {
	var st Storage

	// Split name/path and options
	parts := strings.SplitN(s, ",", 2)
	if len(parts) < 1 {
		return nil, ErrInvalidStorageString
	}

	// "storage0:1011/vm-1011-disk-1.qcow2"
	np := parts[0]
	npParts := strings.SplitN(np, ":", 2)
	if len(npParts) != 2 {
		err := errors.Join(ErrInvalidStorageString, errors.New("Missing ':'"))
		return nil, err
	}
	st.Name = npParts[0]

	// "1011/vm-1011-disk-1.qcow2"
	vmFileParts := strings.SplitN(npParts[1], "/", 2)
	if len(vmFileParts) != 2 {
		err := errors.Join(ErrInvalidStorageString, errors.New("invalid VM/file format"))
		return nil, err
	}
	vmid, err := strconv.ParseUint(vmFileParts[0], 10, 32)
	if err != nil {
		err := errors.Join(ErrInvalidStorageString, errors.New("invalid VMID"))
		return nil, err
	}
	st.VMID = uint32(vmid)
	st.File = vmFileParts[1]

	if len(parts) < 2 {
		return &st, nil
	}

	options := strings.SplitSeq(parts[1], ",")
	for opt := range options {
		kv := strings.SplitN(opt, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "discard":
			st.Discard = (kv[1] == "on")
		case "size":
			sizeStr := strings.TrimSuffix(kv[1], "G")
			val, err := strconv.ParseUint(sizeStr, 10, 32)
			if err != nil {
				err := errors.Join(ErrInvalidStorageString, fmt.Errorf("invalid size: %v", err))
				return nil, err
			}
			st.Size = uint(val)
		}
	}

	return &st, nil
}

// If vlanTag is 0, remove any existing tag from the iface string
func substituteVlanTag(iface string, vlanTag uint16) string {
	// Iface has the following format: "virtio=BC:24:11:64:07:FE,bridge=saspS,tag=7,firewall=1"

	parts := strings.Split(iface, ",")
	var newParts []string
	for _, part := range parts {
		if strings.HasPrefix(part, "tag=") {
			if vlanTag == 0 {
				continue // skip existing tag
			} else {
				part = fmt.Sprintf("tag=%d", vlanTag)
			}
		}
		newParts = append(newParts, part)
	}

	if vlanTag != 0 && !strings.Contains(iface, "tag=") {
		newParts = append(newParts, fmt.Sprintf("tag=%d", vlanTag))
	}
	return strings.Join(newParts, ",")
}

func mapVMIDToProxmoxNodes(cluster *proxmox.Cluster) (map[uint64]string, error) {
	resources, err := getProxmoxResources(cluster, "vm")
	if err != nil {
		return nil, err
	}

	// Map VMID to Node
	vmNodes := make(map[uint64]string)
	for _, r := range resources {
		if r.Type != "qemu" {
			continue
		}
		vmNodes[r.VMID] = r.Node
	}

	return vmNodes, nil
}

func waitForProxmoxTaskCompletion(t *proxmox.Task) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	isSuccessful, completed, err := t.WaitForCompleteStatus(ctx, 240, 1)
	cancel()
	if err != nil {
		logger.Error("Failed to wait for Proxmox task completion", "error", err)
		return false, err
	}

	if !completed {
		return waitForProxmoxTaskCompletion(t)
	}

	if !isSuccessful {
		logger.Error("Proxmox task failed")
		return false, errors.New("task_failed")
	}

	return true, nil
}

func getProxmoxCluster(client *proxmox.Client) (*proxmox.Cluster, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	cluster, err := client.Cluster(ctx)
	cancel()
	if err != nil {
		logger.Error("Failed to get Proxmox cluster", "error", err)
		return nil, err
	}
	return cluster, nil
}

func getProxmoxNode(client *proxmox.Client, nodeName string) (*proxmox.Node, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	node, err := client.Node(ctx, nodeName)
	cancel()
	if err != nil {
		logger.Error("Failed to get Proxmox node", "error", err, "node", nodeName)
		return nil, err
	}
	return node, nil
}

func getProxmoxVM(node *proxmox.Node, vmid int) (*proxmox.VirtualMachine, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	vm, err := node.VirtualMachine(ctx, vmid)
	cancel()
	if err != nil {
		logger.Error("Failed to get Proxmox VM", "error", err, "node", node.Name, "vmid", vmid)
		return nil, err
	}
	return vm, nil
}

func getProxmoxResources(cluster *proxmox.Cluster, filters ...string) (proxmox.ClusterResources, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	resources, err := cluster.Resources(ctx, filters...)
	cancel()
	if err != nil {
		logger.Error("Failed to get Proxmox resources", "error", err)
		return nil, err
	}
	return resources, nil
}

func configureVM(vm *proxmox.VirtualMachine, config proxmox.VirtualMachineOption) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	task, err := vm.Config(ctx, config)
	cancel()
	if err != nil {
		logger.Error("Failed to set VM config", "error", err, "vmid", vm.VMID)
		return false, err
	}

	return waitForProxmoxTaskCompletion(task)
}

func getProxmoxStorage(node *proxmox.Node, storage string) (*proxmox.Storage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	s, err := node.Storage(ctx, storage)
	cancel()
	if err != nil {
		logger.Error("Failed to get Proxmox Storage", "error", err, "node", node.Name, "storage", storage)
		return nil, err
	}
	return s, nil
}

func getProxmoxStorageContent(s *proxmox.Storage) ([]*proxmox.StorageContent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	content, err := s.GetContent(ctx)
	cancel()
	if err != nil {
		logger.Error("Failed to get Proxmox Storage content", "error", err, "storage", s.Name)
		return nil, err
	}
	return content, nil
}
