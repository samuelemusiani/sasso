package gateway

import (
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
}

func NewLinuxGateway() *LinuxGateway {
	return &LinuxGateway{}
}

func (lg *LinuxGateway) Init(c config.Gateway) error {
	lg.Port = c.Linux.Port

	for _, p := range c.Linux.Peers {
		ip := net.ParseIP(p)
		if ip == nil {
			return fmt.Errorf("Failed to parse peer IP: %s", p)
		}
		lg.Peers = append(lg.Peers, ip)
	}

	return nil
}

func (lg *LinuxGateway) NewInterface(vnet string, vnetID uint32, subnet, routerIP, broadcast string) (*Interface, error) {

	link := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name: vnet,
		},
		VxlanId: int(vnetID),
		Port:    int(lg.Port),
	}
	err := netlink.LinkAdd(link)
	if err != nil {
		logger.With("error", err).Error("Failed to create VxLAN interface")
		return nil, err
	}
	ipAddr, err := netlink.ParseAddr(routerIP)
	if err != nil {
		logger.With("error", err, "routerIP", routerIP).Error("Failed to parse router IP address")
		return nil, err
	}

	err = netlink.AddrAdd(link, ipAddr)
	if err != nil {
		logger.With("error", err, "ipAddress", ipAddr, "iface", link.Name).Error("Failed to add IP address to network interface on router")
		return nil, err
	}

	err = netlink.LinkSetUp(link)
	if err != nil {
		slog.With("error", err).Error("Failed to set interface up")
		panic(err)
	}

	peers := []net.IP{net.IP{130, 136, 201, 1}, net.IP{130, 136, 201, 2}, net.IP{130, 136, 201, 3}}

	for _, p := range peers {
		err = netlink.NeighAppend(&netlink.Neigh{
			LinkIndex:    link.Index,
			IP:           p,
			HardwareAddr: make(net.HardwareAddr, 6),
			Flags:        netlink.NTF_SELF,
			State:        netlink.NUD_PERMANENT,
			Family:       unix.AF_BRIDGE,
		})
		if err != nil {
			slog.With("error", err, "p", p.String(), "LinkIndex", link.Index).Error("Failed to add neighbor")
			return nil, err
		}
	}

	return &Interface{
		ID:     uint(link.Index),
		VNet:   vnet,
		VNetID: vnetID,

		Subnet:    subnet,
		RouterIP:  routerIP,
		Broadcast: broadcast,

		FirewallInterfaceName: link.Name,
	}, nil
}

func (lg *LinuxGateway) RemoveInterface(id uint) error {
	err := netlink.LinkDel(&netlink.Vxlan{LinkAttrs: netlink.LinkAttrs{Index: int(id)}})
	if err != nil {
		logger.With("error", err, "id", id).Error("Failed to remove VxLAN interface")
		return err
	}
	return nil
}
