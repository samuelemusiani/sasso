package gateway

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"samuelemusiani/sasso/router/config"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

type LinuxGateway struct {
	Port  uint16
	Peers []net.IP
	MTU   uint16
}

func NewLinuxGateway() *LinuxGateway {
	return &LinuxGateway{}
}

func (lg *LinuxGateway) Init(c config.Gateway) error {
	if c.Linux.Port == 0 {
		return fmt.Errorf("Linux gateway port cannot be 0")
	}

	lg.Port = c.Linux.Port

	if len(c.Linux.Peers) == 0 {
		return fmt.Errorf("Linux gateway must have at least one peer")
	}

	for _, p := range c.Linux.Peers {
		ip := net.ParseIP(p)
		if ip == nil {
			return fmt.Errorf("Failed to parse peer IP: %s", p)
		}
		lg.Peers = append(lg.Peers, ip)
	}

	if c.Linux.MTU == 0 {
		return fmt.Errorf("Linux gateway MTU cannot be 0")
	}

	lg.MTU = c.Linux.MTU

	return nil
}

func (lg *LinuxGateway) NewInterface(vnet string, vnetID uint32, subnet, routerIP, broadcast string) (*Interface, error) {

	link := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			MTU:  int(lg.MTU),
			Name: vnet,
		},
		VxlanId: int(vnetID),
		Port:    int(lg.Port),
	}
	err := netlink.LinkAdd(link)
	if err != nil {
		logger.Error("Failed to create VxLAN interface", "error", err)
		return nil, err
	}
	ipAddr, err := netlink.ParseAddr(routerIP)
	if err != nil {
		logger.Error("Failed to parse router IP address", "error", err, "routerIP", routerIP)
		return nil, err
	}

	err = netlink.AddrAdd(link, ipAddr)
	if err != nil {
		logger.Error("Failed to add IP address to network interface on router", "error", err, "ipAddress", ipAddr, "iface", link.Name)
		return nil, err
	}

	err = netlink.LinkSetUp(link)
	if err != nil {
		slog.Error("Failed to set interface up", "error", err)
		panic(err)
	}

	for _, p := range lg.Peers {
		err = netlink.NeighAppend(&netlink.Neigh{
			LinkIndex:    link.Index,
			IP:           p,
			HardwareAddr: make(net.HardwareAddr, 6),
			Flags:        netlink.NTF_SELF,
			State:        netlink.NUD_PERMANENT,
			Family:       unix.AF_BRIDGE,
		})
		if err != nil {
			slog.Error("Failed to add neighbor", "error", err, "p", p.String(), "LinkIndex", link.Index)
			return nil, err
		}
	}

	return &Interface{
		LocalID: uint(link.Index),
		VNet:    vnet,
		VNetID:  vnetID,

		Subnet:    subnet,
		RouterIP:  routerIP,
		Broadcast: broadcast,

		FirewallInterfaceName: link.Name,
	}, nil
}

func (lg *LinuxGateway) RemoveInterface(id uint) error {
	err := netlink.LinkDel(&netlink.Vxlan{LinkAttrs: netlink.LinkAttrs{Index: int(id)}})
	if err != nil && !errors.Is(err, unix.ENODEV) {
		logger.Error("Failed to remove VxLAN interface", "error", err, "id", id)
		return err
	}
	return nil
}

// True if interface is verified, false otherwise
func (lg *LinuxGateway) VerifyInterface(iface *Interface) (bool, error) {

	link, err := netlink.LinkByIndex(int(iface.LocalID))

	// not present, inconsistant
	var linkNotFoundErr netlink.LinkNotFoundError
	if errors.As(err, &linkNotFoundErr) {
		return false, nil
	}

	if err != nil {
		logger.Error("Failed to get Link", "error", err, "id", iface.LocalID)
		return false, err
	}

	// not a vxlan, inconsistant
	if link.Type() != "vxlan" {
		return false, nil
	}

	vxlanlink, ok := link.(*netlink.Vxlan)
	if !ok {
		return false, nil
	}

	// other  consistancy checks
	if (vxlanlink.Name != iface.VNet) ||
		(vxlanlink.VxlanId != int(iface.VNetID)) ||
		!vxlanlink.SrcAddr.Equal(net.ParseIP(iface.RouterIP)) ||
		!vxlanlink.Group.Equal(net.ParseIP(iface.Broadcast)) {
		return false, nil
	}

	// else is consistant
	return true, nil
}
