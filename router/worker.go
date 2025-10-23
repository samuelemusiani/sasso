package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"sort"
	"time"

	"samuelemusiani/sasso/internal"
	"samuelemusiani/sasso/internal/auth"
	"samuelemusiani/sasso/router/config"
	"samuelemusiani/sasso/router/db"
	"samuelemusiani/sasso/router/fw"
	"samuelemusiani/sasso/router/gateway"
	"samuelemusiani/sasso/router/utils"

	shorewall "github.com/samuelemusiani/go-shorewall"
)

func worker(logger *slog.Logger, conf config.Server) {
	time.Sleep(5 * time.Second)

	logger.Info("Worker started")

	gtw := gateway.Get()
	if gtw == nil {
		panic("Gateway not initialized")
	}

	fw := fw.Get()
	if fw == nil {
		panic("Firewall not initialized")
	}

	for {
		err := verifyNets(logger, gtw)
		if err != nil {
			logger.Error("Failed to verify VNets", "error", err)
		}

		err = checkPortForwards(logger, fw)
		if err != nil {
			logger.Error("Failed to verify port forwards", "error", err)
		}

		nets, err := getNetsStatus(logger, conf)
		if err != nil {
			logger.Error("Failed to get VNets with status", "error", err)
			time.Sleep(10 * time.Second)
			continue
		}

		err = deleteNets(logger, gtw, nets)
		if err != nil {
			logger.Error("Failed to delete VNets", "error", err)
		}

		err = createNets(logger, gtw, nets)
		if err != nil {
			logger.Error("Failed to create VNets", "error", err)
		}

		err = updateNets(logger, conf, nets)
		if err != nil {
			logger.Error("Failed to update VNets", "error", err)
		}

		portForwards, err := getPortForwardsStatus(logger, conf)
		if err != nil {
			logger.Error("Failed to get port forwards status", "error", err)
			time.Sleep(10 * time.Second)
			continue
		}

		err = deletePortForwards(logger, fw, portForwards)
		if err != nil {
			logger.Error("Failed to delete port forwards", "error", err)
		}

		err = createPortForwards(logger, fw, portForwards)
		if err != nil {
			logger.Error("Failed to create port forwards", "error", err)
		}

		time.Sleep(5 * time.Second)
	}
}

// Fetch the main sasso server for the status of the nets
func getNetsStatus(logger *slog.Logger, conf config.Server) ([]internal.Net, error) {
	nets, err := internal.FetchNets(conf.Endpoint, conf.Secret)
	if err != nil {
		logger.Error("Failed to fetch nets status from main server", "error", err)
		return nil, err
	}
	return nets, nil
}

// This function takes care of deleting the interfaces that are present on the DB
// but not on the machine
func verifyNets(logger *slog.Logger, gtw gateway.Gateway) error {

	dbInterfaces, err := db.GetAllInterfaces()
	if err != nil {
		logger.Error("Failed to get all interfaces from database", "error", err)
		return err
	}

	for _, dbIface := range dbInterfaces {
		ok, err := gtw.VerifyInterface(gateway.InterfaceFromDB(&dbIface))

		if err != nil {
			return err
		}

		if !ok {
			// if is not consistant, remove it
			err = gtw.RemoveInterface(dbIface.LocalID)
			if err != nil {
				logger.Error("Failed to remove interface from gateway", "error", err, "local_id", dbIface.LocalID)
			}

			err = db.DeleteInterface(dbIface.ID)
			if err != nil {
				logger.Error("Failed to delete interface from database", "error", err, "interface_id", dbIface.ID)
			}
		}
	}

	return nil
}

// This function takes care of deleting the nets that are present on the DB
// but not on the nets slice anymore
func deleteNets(logger *slog.Logger, gtw gateway.Gateway, nets []internal.Net) error {

	dbInterfaces, err := db.GetAllInterfaces()
	if err != nil {
		logger.Error("Failed to get all interfaces from database", "error", err)
		return err
	}

	var toDelete []db.Interface

	for _, dbIface := range dbInterfaces {
		if slices.IndexFunc(nets, func(n internal.Net) bool {
			return n.Name == dbIface.VNet
		}) == -1 {
			toDelete = append(toDelete, dbIface)
		}
	}

	for _, n := range toDelete {
		dbIface, err := db.GetInterfaceByVNet(n.VNet)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				continue
			}
			logger.Error("Failed to get interface from database", "error", err, "vnet", n.VNet)
			return err
		}

		iface := gateway.InterfaceFromDB(dbIface)
		err = gtw.RemoveInterface(iface.LocalID)
		if err != nil {
			logger.Error("Failed to remove interface from gateway", "error", err, "local_id", iface.LocalID)
		}

		err = db.DeleteInterface(iface.ID)
		if err != nil {
			logger.Error("Failed to delete interface from database", "error", err, "interface_id", iface.ID)
		}

	}
	return nil
}

// This function takes care of creating the nets that are not present on the DB
// but are present on the nets slice
func createNets(logger *slog.Logger, gtw gateway.Gateway, nets []internal.Net) error {
	for _, n := range nets {
		_, err := db.GetInterfaceByVNet(n.Name)
		if err == nil {
			continue
		} else if !errors.Is(err, db.ErrNotFound) {
			logger.Error("Failed to get interface from database", "error", err, "vnet", n.Name)
			continue
		}

		if n.Subnet == "" {
			n.Subnet, err = utils.NextAvailableSubnet()
			if err != nil {
				logger.Error("Failed to get next available subnet", "error", err)
				return err
			}
		}

		if n.Gateway == "" {
			n.Gateway, err = utils.GatewayAddressFromSubnet(n.Subnet)
			if err != nil {
				logger.Error("Failed to get gateway address from subnet", "error", err)
				return err
			}
		}

		if n.Broadcast == "" {
			n.Broadcast, err = utils.GetBroadcastAddressFromSubnet(n.Subnet)
			if err != nil {
				logger.Error("Failed to get broadcast address from subnet", "error", err)
				return err
			}
		}

		inter, err := gtw.NewInterface(n.Name, n.Tag, n.Subnet, n.Gateway, n.Broadcast)
		if err != nil {
			logger.Error("Failed to create new interface on gateway", "error", err)
			return err
		}

		err = inter.SaveToDB()
		if err != nil {
			logger.Error("Failed to save interface to database", "error", err)
			return err
		}
	}

	return nil
}

// This function takes care of updating the nets on the main server that are
// present on the local DB with the correct subnet, gateway and broadcast
func updateNets(logger *slog.Logger, conf config.Server, nets []internal.Net) error {

	for _, n := range nets {
		dbNet, err := db.GetInterfaceByVNet(n.Name)
		if err != nil {
			logger.Error("Failed to get interface from database", "error", err, "vnet", n.Name)
			return err
		}

		if dbNet.Subnet == n.Subnet && dbNet.RouterIP == n.Gateway && dbNet.Broadcast == n.Broadcast {
			continue
		}
		n.Subnet = dbNet.Subnet
		n.Gateway = dbNet.RouterIP
		n.Broadcast = dbNet.Broadcast

		data, err := json.Marshal(n)
		if err != nil {
			logger.Error("Failed to marshal net", "error", err, "vnet", n.Name)
			continue
		}

		client := http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("PUT", fmt.Sprintf("%s/internal/net/%d", conf.Endpoint, n.ID), bytes.NewBuffer(data))
		if err != nil {
			logger.Error("Failed to create request to update net", "error", err, "vnet", n.Name)
			continue
		}

		auth.AddAuthToRequest(req, conf.Secret)
		req.Header.Set("Content-Type", "application/json")
		res, err := client.Do(req)
		if err != nil {
			logger.Error("Failed to update net", "error", err, "vnet", n.Name)
			continue
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			logger.Error("Failed to update net", "status", res.StatusCode, "vnet", n.Name)
			continue
		}
		logger.Info("Updated net on main server", "vnet", n.Name)
	}

	return nil
}

func getPortForwardsStatus(logger *slog.Logger, conf config.Server) ([]internal.PortForward, error) {
	pfs, err := internal.FetchPortForwards(conf.Endpoint, conf.Secret)
	if err != nil {
		logger.Error("Failed to fetch port forwards status from main server", "error", err)
		return nil, err
	}
	return pfs, nil
}

func checkPortForwards(logger *slog.Logger, fw fw.Firewall) error {
	portForwards, err := db.GetPortForwards()
	if err != nil {
		logger.With("error", err).Error("Failed to get all port forwards from DB")
		return err
	}

	fwRules, err := shorewall.GetRules()
	if err != nil {
		logger.With("error", err).Error("Failed to get firewall rules")
		return err
	}

	// sort rules
	sort.Slice(fwRules, func(i, j int) bool {
		if fwRules[i].Action != fwRules[j].Action {
			return fwRules[i].Action < fwRules[j].Action
		}
		if fwRules[i].Destination != fwRules[j].Destination {
			return fwRules[i].Destination < fwRules[j].Destination
		}
		if fwRules[i].Dport != fwRules[j].Dport {
			return fwRules[i].Dport < fwRules[j].Dport
		}
		if fwRules[i].Protocol != fwRules[j].Protocol {
			return fwRules[i].Protocol < fwRules[j].Protocol
		}
		if fwRules[i].Source != fwRules[j].Source {
			return fwRules[i].Source < fwRules[j].Source
		}
		return fwRules[i].Sport < fwRules[j].Sport
	})

	reloadFirewall := false
	for _, pf := range portForwards {
		pfRule := fw.CreatePortForwardsRule(pf.OutPort, pf.DestPort, pf.DestIP)

		index := sort.Search(len(fwRules), func(i int) bool {
			if fwRules[i].Action != pfRule.Action {
				return fwRules[i].Action > pfRule.Action
			}
			if fwRules[i].Destination != pfRule.Destination {
				return fwRules[i].Destination > pfRule.Destination
			}
			if fwRules[i].Dport != pfRule.Dport {
				return fwRules[i].Dport > pfRule.Dport
			}
			if fwRules[i].Protocol != pfRule.Protocol {
				return fwRules[i].Protocol > pfRule.Protocol
			}
			if fwRules[i].Source != pfRule.Source {
				return fwRules[i].Source > pfRule.Source
			}
			return fwRules[i].Sport >= pfRule.Sport
		})

		if index >= len(fwRules) || fwRules[index] != pfRule {
			logger.Info("Port forward rule missing in firewall, adding it", "port_forward_id", pf.ID)
			err = fw.AddPortForward(pf.OutPort, pf.DestPort, pf.DestIP)
			if err != nil {
				logger.With("error", err, "port_forward_id", pf.ID).Error("Failed to add missing port forward rule to firewall")
				continue
			}
			reloadFirewall = true
		}
	}

	// reload shorewall to apply changes
	if reloadFirewall {
		err = shorewall.Reload()
		if err != nil {
			logger.With("error", err).Error("Failed to reload firewall")
			return err
		}
	}

	return nil
}

func deletePortForwards(logger *slog.Logger, fw fw.Firewall, pfs []internal.PortForward) error {
	localPortForwards, err := db.GetPortForwards()
	if err != nil {
		logger.Error("Failed to get all port forwards from database", "error", err)
		return err
	}

	for _, localPF := range localPortForwards {
		if slices.IndexFunc(pfs, func(pf internal.PortForward) bool {
			return pf.ID == localPF.ID
		}) != -1 {
			continue
		}

		err = fw.RemovePortForward(localPF.OutPort, localPF.DestPort, localPF.DestIP)
		if err != nil {
			logger.Error("Failed to remove port forward from firewall", "error", err, "port_forward_id", localPF.ID)
			continue
		}

		err = db.RemovePortForward(localPF.ID)
		if err != nil {
			logger.Error("Failed to remove port forward from database", "error", err, "port_forward_id", localPF.ID)
			continue
		}
	}

	return nil
}

func createPortForwards(logger *slog.Logger, fw fw.Firewall, pfs []internal.PortForward) error {
	for _, pf := range pfs {
		_, err := db.GetPortForwardByID(pf.ID)
		if err == nil {
			continue
		} else if !errors.Is(err, db.ErrNotFound) {
			logger.Error("Failed to get port forward from database", "error", err, "port_forward_id", pf.ID)
			continue
		}

		err = fw.AddPortForward(pf.OutPort, pf.DestPort, pf.DestIP)
		if err != nil {
			logger.Error("Failed to add port forward to firewall", "error", err, "port_forward_id", pf.ID)
			continue
		}

		err = db.AddPortForward(db.PortForward{
			ID:       pf.ID,
			OutPort:  pf.OutPort,
			DestPort: pf.DestPort,
			DestIP:   pf.DestIP,
		})
		if err != nil {
			logger.Error("Failed to save port forward to database", "error", err, "port_forward_id", pf.ID)
			continue
		}
	}
	return nil
}
