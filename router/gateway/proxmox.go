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
	"samuelemusiani/sasso/router/utils"

	"github.com/luthermonson/go-proxmox"
	"github.com/vishvananda/netlink"
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	version, err := pg.client.Version(ctx)
	cancel()
	if err != nil {
		logger.With("err", err).Error("Reading Proxmox API version endpoint")
		return err
	}
	logger.Info("Proxmox version", "version", version.Version)

	return nil
}

func (pg *ProxmoxGateway) NewInterface(vnet string, vnetID uint32, subnet, routerIP, broadcast string) (*Interface, error) {
	vm, err := pg.getVM()

	// TODO: Check if in the future the APIs will acctually support Nets maps
	// https://github.com/luthermonson/go-proxmox/issues/211
	// This is a temporary workaround
	// At the moment we are using the samuelemusiani/go-proxmox fork
	mnets := vm.VirtualMachineConfig.Nets
	// mnets := map[net0:virtio=BC:24:11:D2:FA:F0,bridge=vmbr0,firewall=1 net1:virtio=BC:24:11:B6:1C:2A,bridge=sassoint,firewall=1]

	// snet := [1, 2, 3, ..]
	var snet = make([]int, len(mnets))
	var i int = 0
	for k := range mnets {
		tmp := strings.TrimPrefix(k, "net")
		tmpN, err := strconv.Atoi(tmp)
		if err != nil {
			continue
		}
		snet[i] = tmpN
		i++
	}
	slices.Sort(snet)
	logger.With("mnets", mnets, "snet", snet).Debug("Current network interfaces on Proxmox VM")

	var firstEmptyIndex int = -1
	for i := range snet {
		if snet[i] != i {
			firstEmptyIndex = i
			break
		}
	}
	if firstEmptyIndex == -1 {
		firstEmptyIndex = len(snet)
	}

	interfaceIndex := firstEmptyIndex
	needToAddInterfaceOnProxmox := true

	for i := range mnets {
		if strings.Contains(mnets[i], fmt.Sprintf("bridge=%s", vnet)) {
			logger.With("vnet", vnet, "bridge", mnets[i]).Warn("Network interface already exists on Proxmox VM")
			needToAddInterfaceOnProxmox = false

			// Extract the index from the interface name
			tmps, found := strings.CutPrefix(i, "net")
			if found {
				idx, err := strconv.Atoi(tmps)
				if err == nil {
					interfaceIndex = idx
				}
			}
			break
		}
	}

	if needToAddInterfaceOnProxmox {
		o := proxmox.VirtualMachineOption{
			Name:  "net" + strconv.Itoa(interfaceIndex),
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
	}

	ipConfigs := vm.VirtualMachineConfig.IPConfigs
	needToConfigureInterfaceOnProxmox := true

	for i := range ipConfigs {
		if strings.Contains(ipConfigs[i], fmt.Sprintf("ip=%s", routerIP)) {
			logger.With("routerIP", routerIP, "ipconfig", ipConfigs[i]).Warn("IP configuration already exists on Proxmox VM")
			needToConfigureInterfaceOnProxmox = false
		}
	}

	if needToConfigureInterfaceOnProxmox {
		o2 := proxmox.VirtualMachineOption{
			Name:  "ipconfig" + strconv.Itoa(interfaceIndex),
			Value: fmt.Sprintf("ip=%s", routerIP),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		t, err := vm.Config(ctx, o2)
		cancel()
		if err != nil {
			logger.With("error", err).Error("Failed to configure network interface on Proxmox VM")
			return nil, err
		}
		_, _, err = waitForTaskCompletion(t)
		if err != nil {
			logger.With("error", err).Error("Failed to wait for Proxmox task completion")
			return nil, err
		}
	}

	newVM, err := pg.getVM()
	if err != nil {
		logger.With("error", err).Error("Failed to get Proxmox VM")
		return nil, err
	}

	// Just adding the interface on Proxmox and configuring the IP on cloud-init is not enough
	// If the router is running the interface will not be configured until the next reboot
	// So we need to get the MAC address of the new interface and configure it manually
	// This implies that the router service is running on the Proxmox VM itself
	newInterface := newVM.VirtualMachineConfig.Nets["net"+strconv.Itoa(interfaceIndex)]
	mac, err := extractMacFromInterfaceString(newInterface)
	if err != nil {
		logger.With("error", err, "interface", newInterface).Error("Failed to extract MAC address from interface string")
		return nil, err
	}

	localIface, err := getLinkByMAC(mac)
	if err != nil {
		logger.With("err", err).Error("Failed to get link by mac address")
		return nil, err
	}

	ipAddress, err := netlink.ParseAddr(routerIP)
	if err != nil {
		logger.With("error", err, "routerIP", routerIP).Error("Failed to parse router IP address")
		return nil, err
	}

	addressedConfiguredOnSystem, err := netlink.AddrList(*localIface, netlink.FAMILY_V4)
	if err != nil {
		logger.With("error", err, "iface", (*localIface).Attrs().Name).Error("Failed to list addresses on network interface on router")
		return nil, err
	}

	var needToAddAddressOnSystem = true

	for i := range addressedConfiguredOnSystem {
		if addressedConfiguredOnSystem[i].IPNet.String() == ipAddress.IPNet.String() {
			logger.With("ipAddress", ipAddress, "iface", (*localIface).Attrs().Name).Info("IP address already configured on network interface on router")
			needToAddAddressOnSystem = false
		}
	}

	if needToAddAddressOnSystem {
		err = netlink.AddrAdd(*localIface, ipAddress)
		if err != nil {
			logger.With("error", err, "ipAddress", ipAddress, "iface", (*localIface).Attrs().Name).Error("Failed to add IP address to network interface on router")
			return nil, err
		}
	}

	err = netlink.LinkSetUp(*localIface)
	if err != nil {
		logger.With("error", err, "iface", (*localIface).Attrs().Name).Error("Failed to bring up network interface on router")
		return nil, err
	}

	return &Interface{
		LocalID: uint(interfaceIndex),
		VNet:    vnet,
		VNetID:  vnetID,

		Subnet:    subnet,
		RouterIP:  routerIP,
		Broadcast: broadcast,

		FirewallInterfaceName: (*localIface).Attrs().Name,
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

func extractMacFromInterfaceString(iface string) (string, error) {
	parts := strings.SplitSeq(iface, ",")
	for p := range parts {
		tmps, found := strings.CutPrefix(p, "virtio=")
		if found {
			return tmps, nil
		}
	}
	return "", errors.New("mac_not_found")
}

func getLinkByMAC(mac string) (*netlink.Link, error) {
	links, err := netlink.LinkList()
	if err != nil {
		logger.With("error", err).Error("Failed to list network interfaces on router")
		return nil, err
	}

	for i := range links {
		if utils.AreMACsEqual(links[i].Attrs().HardwareAddr.String(), mac) {
			return &links[i], nil
		}
	}
	return nil, errors.New("Interface not found on router")
}
