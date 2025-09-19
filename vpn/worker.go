package main

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"samuelemusiani/sasso/internal"
	"samuelemusiani/sasso/vpn/config"
	"samuelemusiani/sasso/vpn/db"
	"samuelemusiani/sasso/vpn/util"
	"samuelemusiani/sasso/vpn/wg"

	shorewall "github.com/samuelemusiani/go-shorewall"
)

func worker(logger *slog.Logger, serverConfig config.Server, fwConfig config.Firewall) {
	logger.Info("Worker started")

	for {
		nets, err := internal.FetchNets(serverConfig.Endpoint, serverConfig.Secret)
		if err != nil {
			logger.With("error", err).Error("Failed to fetch nets status from main server")
			time.Sleep(10 * time.Second)
			continue
		}

		err = deleteNets(logger, nets, fwConfig)
		if err != nil {
			logger.With("error", err).Error("Failed to delete VNets")
		}

		err = createNets(logger, nets, fwConfig)
		if err != nil {
			logger.With("error", err).Error("Failed to create VNets")
		}

		err = updateNetsOnServer(logger, serverConfig.Endpoint, serverConfig.Secret)
		if err != nil {
			logger.With("error", err).Error("Failed to update nets on main server")
		}

		time.Sleep(5 * time.Second)
	}
}

// This function takes care of deleting the nets that are present on the DB
// but not present on the main server
func deleteNets(logger *slog.Logger, nets []internal.Net, fwConfig config.Firewall) error {
	logger.Info("Deleting nets", "nets", nets)
	// TODO: Implement this function
	return nil
}

// This function takes care of creating the nets that are present on the main
// server but not present on the DB
func createNets(logger *slog.Logger, nets []internal.Net, fwConfig config.Firewall) error {
	logger.Info("Creating nets", "nets", nets)

	for _, n := range nets {
		if n.Subnet == "" {
			logger.Info("Skipping net with empty subnet", "net", n)
			continue
		}

		// Check if the subnet associated to the net already exists in the DB
		exist, err := db.CheckSubnetExists(n.Subnet)
		if err != nil {
			logger.With("error", err).Error("Failed to check if subnet exists")
			continue
		}

		// If it exists, skip it
		if exist {
			logger.Info("Subnet already exists, skipping", "subnet", n.Subnet)
			continue
		}

		newAddr, err := util.NextAvailableAddress()
		if err != nil {
			logger.With("error", err).Error("Failed to generate new address")
			continue
		}

		wgInterface, err := wg.NewWGConfig(newAddr, n.Subnet)
		if err != nil {
			logger.With("error", err).Error("Failed to generate WireGuard config")
			continue
		}

		err = db.NewInterface(wgInterface.PrivateKey, wgInterface.PublicKey, n.Subnet, newAddr, n.UserID)
		if err != nil {
			logger.With("error", err).Error("Failed to save interface to database")
			continue
		}

		err = shorewall.AddRule(shorewall.Rule{
			Action:      "ACCEPT",
			Source:      fmt.Sprintf("%s:%s", fwConfig.VPNZone, newAddr),
			Destination: fmt.Sprintf("%s:%s", fwConfig.SassoZone, n.Subnet),
		})

		if err != nil {
			logger.With("error", err).Error("Failed to add firewall rule")
			continue
		}

		if err = shorewall.Reload(); err != nil {
			logger.With("error", err).Error("Failed to reload firewall")
			continue
		}

		err = wg.CreateInterface(wgInterface)
		if err != nil {
			logger.With("error", err).Error("Failed to create WireGuard interface")
			continue
		}
	}

	return nil
}

func updateNetsOnServer(logger *slog.Logger, endpoint, secret string) error {
	vpns, err := internal.FetchVPNConfigs(endpoint, secret)
	if err != nil {
		logger.With("error", err).Error("Failed to fetch VPN configs from main server")
		return err
	}

	logger.With("vpns", vpns).Info("Fetched VPN configs from main server")

	localInterfaces, err := db.GetAllInterfaces()
	if err != nil {
		logger.With("error", err).Error("Failed to fetch local interfaces from DB")
		return err
	}

	for _, i := range localInterfaces {
		if slices.IndexFunc(vpns, func(v internal.VPNUpdate) bool { return v.UserID == i.UserID }) == -1 {
			vpns = append(vpns, internal.VPNUpdate{
				UserID:    i.UserID,
				VPNConfig: "",
			})
		}
	}

	for _, v := range vpns {
		iface, err := db.GetInterfaceByUserID(v.UserID)
		if err != nil {
			logger.With("error", err).Error("Failed to get interface from DB")
			continue
		}

		wgIface := wg.InterfaceFromDB(iface)
		base64Conf := base64.StdEncoding.EncodeToString([]byte(wgIface.String()))
		if base64Conf == v.VPNConfig {
			continue
		}

		logger.With("user_id", v.UserID).Info("Updating VPN config on main server")
		err = internal.UpdateVPNConfig(endpoint, secret, internal.VPNUpdate{
			UserID:    v.UserID,
			VPNConfig: base64Conf,
		})
	}

	return nil
}
