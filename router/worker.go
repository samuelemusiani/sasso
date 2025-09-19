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

	for {
		nets, err := getNetsStatus(logger, conf)
		if err != nil {
			logger.With("error", err).Error("Failed to get VNets with status")
			time.Sleep(10 * time.Second)
			continue
		}

		err = deleteNets(logger, gtw, nets)
		if err != nil {
			logger.With("error", err).Error("Failed to delete VNets")
		}

		err = createNets(logger, gtw, nets)
		if err != nil {
			logger.With("error", err).Error("Failed to create VNets")
		}

		err = updateNets(logger, conf, nets)
		if err != nil {
			logger.With("error", err).Error("Failed to update VNets")
		}

		time.Sleep(5 * time.Second)
	}
}

// Fetch the main sasso server for the status of the nets
func getNetsStatus(logger *slog.Logger, conf config.Server) ([]internal.Net, error) {
	nets, err := internal.FetchNets(conf.Endpoint, conf.Secret)
	if err != nil {
		logger.With("error", err).Error("Failed to fetch nets status from main server")
		return nil, err
	}
	logger.Info("Fetched nets status from main server", "nets", nets)
	return nets, nil
}

// This function takes care of deleting the nets that are present on the DB
// but not on the nets slice anymore
func deleteNets(logger *slog.Logger, gtw gateway.Gateway, nets []internal.Net) error {

	dbInterfaces, err := db.GetAllInterfaces()
	if err != nil {
		logger.With("error", err).Error("Failed to get all interfaces from database")
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
			logger.With("error", err, "vnet", n.VNet).Error("Failed to get interface from database")
			return err
		}

		iface := gateway.InterfaceFromDB(dbIface)
		err = gtw.RemoveInterface(iface.LocalID)
		if err != nil {
			logger.With("error", err, "local_id", iface.LocalID).Error("Failed to remove interface from gateway")
		}

		err = fw.DeleteInterface(iface)
		if err != nil {
			logger.With("error", err, "interface_id", iface.ID).Error("Failed to delete interface from firewall")
		}

		err = db.DeleteInterface(iface.ID)
		if err != nil {
			logger.With("error", err, "interface_id", iface.ID).Error("Failed to delete interface from database")
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
			logger.With("error", err, "vnet", n.Name).Error("Failed to get interface from database")
			continue
		}

		s, err := utils.NextAvailableSubnet()
		if err != nil {
			logger.With("error", err).Error("Failed to get next available subnet")
			return err
		}
		n.Subnet = s

		gt, err := utils.GatewayAddressFromSubnet(s)
		if err != nil {
			logger.With("error", err).Error("Failed to get gateway address from subnet")
			return err
		}
		n.Gateway = gt

		br, err := utils.GetBroadcastAddressFromSubnet(s)
		if err != nil {
			logger.With("error", err).Error("Failed to get broadcast address from subnet")
			return err
		}
		n.Broadcast = br

		inter, err := gtw.NewInterface(n.Name, n.Tag, n.Subnet, n.Gateway, n.Broadcast)
		if err != nil {
			logger.With("error", err).Error("Failed to create new interface on gateway")
			return err
		}

		err = inter.SaveToDB()
		if err != nil {
			logger.With("error", err).Error("Failed to save interface to database")
			return err
		}

		err = fw.AddInterface(inter)
		if err != nil {
			logger.With("error", err).Error("Failed to create new interface on firewall")
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
			logger.With("error", err, "vnet", n.Name).Error("Failed to get interface from database")
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
			logger.With("error", err, "vnet", n.Name).Error("Failed to marshal net")
			continue
		}

		client := http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("PUT", fmt.Sprintf("%s/internal/net/%d", conf.Endpoint, n.ID), bytes.NewBuffer(data))
		if err != nil {
			logger.With("error", err, "vnet", n.Name).Error("Failed to create request to update net")
			continue
		}

		auth.AddAuthToRequest(req, conf.Secret)
		req.Header.Set("Content-Type", "application/json")
		res, err := client.Do(req)
		if err != nil {
			logger.With("error", err, "vnet", n.Name).Error("Failed to update net")
			continue
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			logger.With("status", res.StatusCode, "vnet", n.Name).Error("Failed to update net")
			continue
		}
		logger.With("vnet", n.Name).Info("Updated net on main server")
	}

	return nil
}
