package main

import (
	"errors"
	"log/slog"
	"net/url"
	"slices"
	"time"

	"samuelemusiani/sasso/internal"
	"samuelemusiani/sasso/router/config"
	"samuelemusiani/sasso/router/db"
	"samuelemusiani/sasso/router/fw"
	"samuelemusiani/sasso/router/gateway"
	"samuelemusiani/sasso/router/utils"
)

func checkConfig(c config.Server) error {
	if c.Endpoint == "" {
		return errors.New("server endpoint cannot be empty")
	}

	if c.Secret == "" {
		return errors.New("server secret cannot be empty")
	}

	_, err := url.Parse(c.Endpoint)
	if err != nil {
		return errors.New("server endpoint is not a valid URL")
	}

	return nil
}

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
			logger.Error("failed to verify VNets", "error", err)
		}

		err = checkPortForwards(logger, fw)
		if err != nil {
			logger.Error("failed to verify port forwards", "error", err)
		}

		nets, err := getNetsStatus(logger, conf)
		if err != nil {
			logger.Error("failed to get VNets with status", "error", err)
			time.Sleep(10 * time.Second)

			continue
		}

		err = deleteNets(logger, gtw, nets)
		if err != nil {
			logger.Error("failed to delete VNets", "error", err)
		}

		err = createNets(logger, gtw, nets)
		if err != nil {
			logger.Error("failed to create VNets", "error", err)
		}

		err = updateNets(logger, conf, nets)
		if err != nil {
			logger.Error("failed to update VNets", "error", err)
		}

		portForwards, err := getPortForwardsStatus(logger, conf)
		if err != nil {
			logger.Error("failed to get port forwards status", "error", err)
			time.Sleep(10 * time.Second)

			continue
		}

		err = deletePortForwards(logger, fw, portForwards)
		if err != nil {
			logger.Error("failed to delete port forwards", "error", err)
		}

		err = createPortForwards(logger, fw, portForwards)
		if err != nil {
			logger.Error("failed to create port forwards", "error", err)
		}

		time.Sleep(5 * time.Second)
	}
}

// Fetch the main sasso server for the status of the nets
func getNetsStatus(logger *slog.Logger, conf config.Server) ([]internal.Net, error) {
	nets, err := internal.FetchNets(conf.Endpoint, conf.Secret)
	if err != nil {
		logger.Error("failed to fetch nets status from main server", "error", err)

		return nil, err
	}

	return nets, nil
}

// This function takes care of deleting the interfaces that are present on the DB
// but not on the machine
func verifyNets(logger *slog.Logger, gtw gateway.Gateway) error {
	dbInterfaces, err := db.GetAllInterfaces()
	if err != nil {
		logger.Error("failed to get all interfaces from database", "error", err)

		return err
	}

	for _, dbIface := range dbInterfaces {
		ok, err := gtw.VerifyInterface(gateway.InterfaceFromDB(&dbIface))
		if err != nil {
			return err
		}

		if !ok {
			// if is not consistent, remove it
			err = gtw.RemoveInterface(dbIface.LocalID)
			if err != nil {
				logger.Error("failed to remove interface from gateway", "error", err, "local_id", dbIface.LocalID)
			}

			err = db.DeleteInterface(dbIface.ID)
			if err != nil {
				logger.Error("failed to delete interface from database", "error", err, "interface_id", dbIface.ID)
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
		logger.Error("failed to get all interfaces from database", "error", err)

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

			logger.Error("failed to get interface from database", "error", err, "vnet", n.VNet)

			return err
		}

		iface := gateway.InterfaceFromDB(dbIface)

		err = gtw.RemoveInterface(iface.LocalID)
		if err != nil {
			logger.Error("failed to remove interface from gateway", "error", err, "local_id", iface.LocalID)
		}

		err = db.DeleteInterface(iface.ID)
		if err != nil {
			logger.Error("failed to delete interface from database", "error", err, "interface_id", iface.ID)
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
			logger.Error("failed to get interface from database", "error", err, "vnet", n.Name)

			continue
		}

		if n.Subnet == "" {
			n.Subnet, err = utils.NextAvailableSubnet()
			if err != nil {
				logger.Error("failed to get next available subnet", "error", err)

				return err
			}
		}

		if n.Gateway == "" {
			n.Gateway, err = utils.GatewayAddressFromSubnet(n.Subnet)
			if err != nil {
				logger.Error("failed to get gateway address from subnet", "error", err)

				return err
			}
		}

		if n.Broadcast == "" {
			n.Broadcast, err = utils.GetBroadcastAddressFromSubnet(n.Subnet)
			if err != nil {
				logger.Error("failed to get broadcast address from subnet", "error", err)

				return err
			}
		}

		inter, err := gtw.NewInterface(n.Name, n.Tag, n.Subnet, n.Gateway, n.Broadcast)
		if err != nil {
			logger.Error("failed to create new interface on gateway", "error", err)

			return err
		}

		err = inter.SaveToDB()
		if err != nil {
			logger.Error("failed to save interface to database", "error", err)

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
			logger.Error("failed to get interface from database", "error", err, "vnet", n.Name)

			return err
		}

		if dbNet.Subnet == n.Subnet && dbNet.RouterIP == n.Gateway && dbNet.Broadcast == n.Broadcast {
			continue
		}

		n.Subnet = dbNet.Subnet
		n.Gateway = dbNet.RouterIP
		n.Broadcast = dbNet.Broadcast

		err = internal.UpdateNet(conf.Endpoint, conf.Secret, n)
		if err != nil {
			logger.Error("failed to update net on main server", "error", err, "vnet", n.Name)

			continue
		}

		logger.Info("Updated net on main server", "vnet", n.Name)
	}

	return nil
}

func getPortForwardsStatus(logger *slog.Logger, conf config.Server) ([]internal.PortForward, error) {
	pfs, err := internal.FetchPortForwards(conf.Endpoint, conf.Secret)
	if err != nil {
		logger.Error("failed to fetch port forwards status from main server", "error", err)

		return nil, err
	}

	return pfs, nil
}

func rulesFromPortForwardsDB(portForwards []db.PortForward) []fw.Rule {
	rules := make([]fw.Rule, 0, len(portForwards))
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
		logger.With("error", err).Error("failed to get all port forwards from DB")

		return err
	}

	rules := rulesFromPortForwardsDB(portForwards)

	// get all port forward rules present in db but not in firewall
	faultyRules, err := firewall.VerifyPortForwardRules(rules)
	if err != nil {
		logger.With("error", err).Error("failed to verify port forward rules")

		return err
	}

	// add to firewall
	err = firewall.AddPortForwardRules(faultyRules)
	if err != nil {
		logger.With("error", err).Error("failed to add faulty rules")

		return err
	}

	return nil
}

// IMPORTANT: pfs are the desired port forwards from the main server, NOT THE ONES THAT MUST BE DELETED.
// This function deletes the difference between the port forwards in database and the desired ones.
func deletePortForwards(logger *slog.Logger, firewall fw.Firewall, pfs []internal.PortForward) error {
	pdfDB, err := db.GetPortForwards()
	if err != nil {
		logger.Error("failed to get all port forwards from database", "error", err)

		return err
	}

	//nolint:prealloc // Deletions are rare; preallocation would waste memory
	var toBeDeleted []db.PortForward

	for _, pfDB := range pdfDB {
		// skip port forwards that are still desired
		if slices.IndexFunc(pfs, func(pf internal.PortForward) bool {
			return pf.ID == pfDB.ID
		}) != -1 {
			continue
		}

		// delete port forward
		toBeDeleted = append(toBeDeleted, pfDB)
	}

	// delete requested rules, even if are not in database
	rules := make([]fw.Rule, 0, len(toBeDeleted))
	for _, r := range toBeDeleted {
		rules = append(rules, fw.Rule{
			OutPort:  r.OutPort,
			DestPort: r.DestPort,
			DestIP:   r.DestIP,
		})
	}

	err = firewall.RemovePortForwardRules(rules)
	if err != nil {
		logger.Error("failed to remove ports forward from firewall", "error", err)

		return err
	}

	// removes all rules present in database to be deleted
	for _, localPF := range toBeDeleted {
		err = db.RemovePortForward(localPF.ID)
		if err != nil {
			logger.Error("failed to remove port forward from database", "error", err, "port_forward_id", localPF.ID)

			continue
		}
	}

	return nil
}

func createPortForwards(logger *slog.Logger, firewall fw.Firewall, pfs []internal.PortForward) error {
	// Get only the portForwards not in database
	var pfsNotDB []db.PortForward

	for _, pf := range pfs {
		_, err := db.GetPortForwardByID(pf.ID)
		switch {
		case err == nil:
			continue
		case errors.Is(err, db.ErrNotFound):
			pfsNotDB = append(pfsNotDB, db.PortForward{
				ID:       pf.ID,
				OutPort:  pf.OutPort,
				DestPort: pf.DestPort,
				DestIP:   pf.DestIP,
			})
		default:
			logger.Error("failed to get port forward from database", "error", err, "port_forward_id", pf.ID)

			continue
		}
	}

	// create rules only if not present in database
	rules := rulesFromPortForwardsDB(pfsNotDB)

	err := firewall.AddPortForwardRules(rules)
	if err != nil {
		logger.Error("failed to add ports forward from firewall", "error", err)

		return err
	}

	for _, pfDB := range pfsNotDB {
		err = db.AddPortForward(pfDB)
		if err != nil {
			logger.Error("failed to save port forward to database", "rule", pfDB, "error", err)

			continue
		}
	}

	return nil
}
