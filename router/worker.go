package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"samuelemusiani/sasso/internal"
	"samuelemusiani/sasso/internal/auth"
	"samuelemusiani/sasso/router/config"
	"samuelemusiani/sasso/router/db"
	"samuelemusiani/sasso/router/fw"
	"samuelemusiani/sasso/router/gateway"
	"samuelemusiani/sasso/router/utils"
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

func rulesFromPortForwardsDB(portForwards []db.PortForward) []fw.Rule {
	var rules []fw.Rule
	for _, pf := range portForwards {
		rules = append(rules, fw.Rule{
			OutPort:  pf.OutPort,
			DestPort: pf.DestPort,
			DestIP:   pf.DestIP,
		})
	}

	return rules
}

func checkPortForwards(logger *slog.Logger, firewall fw.Firewall) error {
	portForwards, err := db.GetPortForwards()
	if err != nil {
		logger.With("error", err).Error("Failed to get all port forwards from DB")
		return err
	}

	rules := rulesFromPortForwardsDB(portForwards)

	// get all port forward rules present in db but not in firewall
	faultyRules, err := firewall.VerifyPortForwardRules(rules)
	if err != nil {
		logger.With("error", err).Error("Failed to verify port forward rules")
		return err
	}

	// add to firewall
	err = firewall.AddPortForwardRules(faultyRules)
	if err != nil {
		logger.With("error", err).Error("Failed to add faulty rules")
		return err
	}

	return nil
}

// IMPORTANT: pfs are the desired port forwards from the main server, NOT THE ONES THAT MUST BE DELETED.
// This function deletes the difference between the port forwards in database and the desired ones.
func deletePortForwards(logger *slog.Logger, firewall fw.Firewall, pfs []internal.PortForward) error {

	pdfDb, err := db.GetPortForwards()
	if err != nil {
		logger.Error("Failed to get all port forwards from database", "error", err)
		return err
	}

	var toBeDeleted []db.PortForward
	for _, pfDb := range pdfDb {
		// skip port forwards that are still desired
		if slices.IndexFunc(pfs, func(pf internal.PortForward) bool {
			return pf.ID == pfDb.ID
		}) != -1 {
			continue
		}

		// delete port forward
		toBeDeleted = append(toBeDeleted, pfDb)
	}

	// delete requested rules, even if are not in database
	var rules []fw.Rule
	for _, r := range toBeDeleted {
		rules = append(rules, fw.Rule{
			OutPort:  r.OutPort,
			DestPort: r.DestPort,
			DestIP:   r.DestIP,
		})
	}

	err = firewall.RemovePortForwardRules(rules)
	if err != nil {
		logger.Error("Failed to remove ports forward from firewall", "error", err)
		return err
	}

	// removes all rules present in database to be deleted
	for _, localPF := range toBeDeleted {
		err = db.RemovePortForward(localPF.ID)
		if err != nil {
			logger.Error("Failed to remove port forward from database", "error", err, "port_forward_id", localPF.ID)
			continue
		}
	}

	return nil
}

func createPortForwards(logger *slog.Logger, firewall fw.Firewall, pfs []internal.PortForward) error {
	// Get only the portForwards not in database
	var pfsNotDb []db.PortForward
	for _, pf := range pfs {
		_, err := db.GetPortForwardByID(pf.ID)
		if err == nil {
			continue
		} else if errors.Is(err, db.ErrNotFound) {
			pfsNotDb = append(pfsNotDb, db.PortForward{
				ID:       pf.ID,
				OutPort:  pf.OutPort,
				DestPort: pf.DestPort,
				DestIP:   pf.DestIP,
			})
		} else {
			logger.Error("Failed to get port forward from database", "error", err, "port_forward_id", pf.ID)
			continue
		}
	}

	// create rules only if not present in database
	rules := rulesFromPortForwardsDB(pfsNotDb)
	err := firewall.AddPortForwardRules(rules)
	if err != nil {
		logger.Error("Failed to add ports forward from firewall", "error", err)
		return err
	}

	for _, pfDb := range pfsNotDb {
		err = db.AddPortForward(pfDb)
		if err != nil {
			logger.Error("Failed to save port forward to database", "rule", pfDb, "error", err)
			continue
		}
	}
	return nil
}
