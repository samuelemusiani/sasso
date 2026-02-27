package gateway

import (
	"errors"
	"fmt"
	"log/slog"
	"net"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"samuelemusiani/sasso/router/config"
)

type LinuxGateway struct {
	Port          uint16
	Peers         []net.IP
	MTU           uint16
	LinkAliasCode string
}

func NewLinuxGateway() *LinuxGateway {
	return &LinuxGateway{}
}

func (lg *LinuxGateway) Init(c config.Gateway) error {
	if c.Linux.Port == 0 {
		return errors.New("linux gateway port cannot be 0")
	}

	lg.Port = c.Linux.Port

	if len(c.Linux.Peers) == 0 {
		return errors.New("linux gateway must have at least one peer")
	}

	for _, p := range c.Linux.Peers {
		ip := net.ParseIP(p)
		if ip == nil {
			return fmt.Errorf("failed to parse peer IP: %s", p)
		}

		lg.Peers = append(lg.Peers, ip)
	}

	if c.Linux.MTU == 0 {
		return errors.New("linux gateway MTU cannot be 0")
	}

	lg.MTU = c.Linux.MTU

	// This is used to identify the links created by sasso in the system.
	// The 'random' string is created with
	//  echo -n "managed iface" | base64
	lg.LinkAliasCode = "sasso-bWFuYWdlZCBpZmFjZQ"

	return nil
}

func (lg *LinuxGateway) NewInterface(vnet string, vnetID uint32, subnet, routerIP, broadcast string) (*Interface, error) {
	ipAddr, err := netlink.ParseAddr(routerIP)
	if err != nil {
		logger.Error("Failed to parse router IP address", "error", err, "routerIP", routerIP)

		return nil, err
	}

	link := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			MTU:   int(lg.MTU),
			Name:  vnet,
			Alias: lg.LinkAliasCode,
		},
		VxlanId: int(vnetID),
		Port:    int(lg.Port),
	}

	err = netlink.LinkAdd(link)
	if err != nil {
		logger.Error("Failed to create VxLAN interface", "error", err)

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

func (*LinuxGateway) RemoveInterface(localID uint) error {
	err := netlink.LinkDel(&netlink.Vxlan{LinkAttrs: netlink.LinkAttrs{Index: int(localID)}})
	if err != nil && !errors.Is(err, unix.ENODEV) {
		logger.Error("Failed to remove VxLAN interface", "error", err, "id", localID)

		return err
	}

	return nil
}

// VerifyInterface returns True if interface is verified, false otherwise.
// "Verified" means that the interfaces exists and has all the correct attributes.
func (*LinuxGateway) VerifyInterface(iface *Interface) (bool, error) {
	link, err := netlink.LinkByIndex(int(iface.LocalID))

	// not present, inconsistent
	var linkNotFoundErr netlink.LinkNotFoundError
	if errors.As(err, &linkNotFoundErr) {
		return false, nil
	}

	if err != nil {
		logger.Error("Failed to get Link", "error", err, "id", iface.LocalID)

		return false, err
	}

	// not a vxlan, inconsistent
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

	// else is consistent
	return true, nil
}

func (lg *LinuxGateway) GetAllInterfaces() ([]*Interface, error) {
	links, err := netlink.LinkList()
	if err != nil {
		logger.Error("Failed to list links", "error", err)

		return nil, err
	}

	var ifaces []*Interface

	for _, link := range links {
		if link.Attrs().Alias == lg.LinkAliasCode {
			vxlanLink, ok := link.(*netlink.Vxlan)
			if !ok {
				logger.Error("Failed to cast link to vxlan", "linkName", link.Attrs().Name)

				continue
			}

			addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
			if err != nil {
				logger.Error("Failed to get addresses for link", "error", err, "linkName", link.Attrs().Name)

				continue
			}

			if len(addrs) == 0 {
				logger.Error("No addresses found for link", "linkName", link.Attrs().Name)

				continue
			}

			var subnet string

			for _, addr := range addrs {
				if addr.IP.Equal(vxlanLink.SrcAddr) {
					subnet = addr.IPNet.String()

					break
				}
			}

			iface := &Interface{
				LocalID:               uint(link.Attrs().Index),
				VNet:                  link.Attrs().Name,
				VNetID:                uint32(vxlanLink.VxlanId),
				Subnet:                subnet,
				RouterIP:              vxlanLink.SrcAddr.String(),
				Broadcast:             vxlanLink.Group.String(),
				FirewallInterfaceName: link.Attrs().Name,
			}
			ifaces = append(ifaces, iface)
		}
	}

	return ifaces, nil
}
