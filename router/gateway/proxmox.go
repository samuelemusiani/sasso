// Implementation of the Gateway interface when the gateway is a Proxmox VM.
package gateway

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"samuelemusiani/sasso/router/config"

	"github.com/luthermonson/go-proxmox"
)

type ProxmoxGateway struct {
	client *proxmox.Client
	vmid   uint
}

func NewProxmoxGateway() *ProxmoxGateway {
	return &ProxmoxGateway{}
}

func (pg *ProxmoxGateway) Init(c config.Gateway) error {
	url := c.Proxmox.Url
	if !strings.Contains(c.Proxmox.Url, "api2/json") {
		if !strings.HasSuffix(c.Proxmox.Url, "/") {
			url += "/"
		}
		url += "api2/json"
	}

	http_client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: c.Proxmox.InsecureSkipVerify,
			},
		},
	}

	pg.client = proxmox.NewClient(url,
		proxmox.WithHTTPClient(&http_client),
		proxmox.WithAPIToken(c.Proxmox.TokenID, c.Proxmox.Secret))

	pg.vmid = c.Proxmox.VMID
	return nil
}

func (pg *ProxmoxGateway) NewInterface(vnet string, vnetID uint, routerIP string) (*Interface, error) {
	vm, err := pg.getVM()

	// TODO: Check if in the future the APIs will acctually support Nets maps
	// https://github.com/luthermonson/go-proxmox/issues/211
	// This is a temporary workaround
	mnets := vm.VirtualMachineConfig.MergeNets()
	var snet = make([]string, len(mnets))
	var i int = 0
	for k := range mnets {
		snet[i] = mnets[k]
		i++
	}
	slices.Sort(snet)
	var firstEmptyIndex int = -1
	for i := range snet {
		if !strings.HasSuffix(snet[i], strconv.Itoa(i)) {
			firstEmptyIndex = i
			break
		}
	}
	if firstEmptyIndex == -1 {
		firstEmptyIndex = len(snet)
	}

	o := proxmox.VirtualMachineOption{
		Name:  "net" + strconv.Itoa(firstEmptyIndex),
		Value: fmt.Sprintf("virtio,bridge=%s,firewall=1", vnet),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t, err := vm.Config(ctx, o)
	cancel()
	if err != nil {
		logger.With("error", err).Error("Failed to add network interface to Proxmox VM")
		return nil, err
	}
	_, _, err = waitForTaskCompletion(t)
	if err != nil {
		logger.With("error", err).Error("Failed to wait for Proxmox task completion")
		return nil, err
	}

	return &Interface{
		LocalID: uint(firstEmptyIndex),
		VNet:    vnet,
		VNetID:  vnetID,
	}, nil
}

func (pg *ProxmoxGateway) RemoveInterface(id uint) error {
	vm, err := pg.getVM()
	if err != nil {
		logger.With("error", err).Error("Failed to get Proxmox VM")
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t, err := vm.Config(ctx, proxmox.VirtualMachineOption{
		Name:  "delete",
		Value: fmt.Sprintf("net%d", id),
	})
	cancel()
	if err != nil {
		logger.With("error", err).Error("Failed to remove network interface from Proxmox VM")
		return err
	}

	_, _, err = waitForTaskCompletion(t)
	if err != nil {
		logger.With("error", err).Error("Failed to wait for Proxmox task completion")
		return err
	}

	return nil
}

// getVM retrieves the Proxmox VM object corresponding to the configured VMID.
func (pg *ProxmoxGateway) getVM() (*proxmox.VirtualMachine, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	cluster, err := pg.client.Cluster(ctx)
	cancel()
	if err != nil {
		logger.With("error", err).Error("Failed to get Proxmox cluster")
		return nil, err
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	resources, err := cluster.Resources(ctx, "vm")
	cancel()

	var vmNode string
	for i := range resources {
		if resources[i].VMID == uint64(pg.vmid) {
			vmNode = resources[i].Node
			break
		}
	}
	if vmNode == "" {
		logger.Error("VM not found in Proxmox cluster")
		return nil, errors.New("vm_not_found")
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	node, err := pg.client.Node(ctx, vmNode)
	cancel()
	if err != nil {
		logger.With("error", err).Error("Failed to get Proxmox node")
		return nil, err
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	vm, err := node.VirtualMachine(ctx, int(pg.vmid))
	cancel()
	if err != nil {
		logger.With("error", err).Error("Failed to get Proxmox VM")
		return nil, err
	}

	return vm, nil
}

func waitForTaskCompletion(t *proxmox.Task) (bool, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	isSuccessful, completed, err := t.WaitForCompleteStatus(ctx, 120, 1)
	cancel()
	if err != nil {
		logger.With("error", err).Error("Failed to wait for Proxmox task completion")
		return false, false, err
	}

	if !completed {
		logger.Error("Proxmox task did not complete in time")
		return false, false, errors.New("task_timeout")
	}

	if !isSuccessful {
		logger.Error("Proxmox task failed")
		return false, true, errors.New("task_failed")
	}

	return true, true, nil
}
