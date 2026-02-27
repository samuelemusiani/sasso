package main

import (
	"context"
	"errors"
	"fmt"
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

func worker(parentCtx context.Context, logger *slog.Logger, conf config.Server, gtw gateway.Gateway, firewall fw.Firewall) {
	logger.Info("Worker started")

	var (
		timeToSleep time.Duration
		err         error
	)
	for {
		if err != nil {
			timeToSleep = 10 * time.Second
		} else {
			timeToSleep = 5 * time.Second
		}

		select {
		case <-time.After(timeToSleep):
		case <-parentCtx.Done():
			logger.Info("Worker stopped")

			return
		}

		// This worker takes care of two things:
		// 1. Nets (interfaces)
		// 2. Port forwards
		//
		// We have 3 states for this resources:
		// 1. Server main state (what we want)
		// 2. Router state (what we have)
		// 3. Router DB (what we rember we had)
		//
		// - The 3rd state is actually needed only if we lost the connection to the
		// 	 main server and we want to have the last rembered state. This is the
		//   case where the service restart and is not able to pull the status from
		// 	 the main server.
		// - Every time we pull an update from the server we must update the DB and
		//   update the Router state
		// - The DB dependecy will be removed in the future
		// - The router state will always be based on the Server state or the last
		//   router DB state.
		//
		// Flow:
		// 1. Pull from Main server (if fail pass over)
		// 2. Update DB (if no update from main server use last stored state)
		// 3. Update router State (with Server state or last DB state)
		// 4. Repeat

		nets, err := fetchNetsFromMainServer(parentCtx, conf)
		if err != nil {
			logger.Error("failed to get VNets with status", "error", err)

			continue
		}

		oldNets := make([]internal.Net, len(nets))
		copy(oldNets, nets)

		nets, err = fillNetsEmptyFields(logger, nets)
		if err != nil {
			logger.Error("failed to fill nets empty fields", "error", err)

			continue
		}

		err = updateDBWithServerNets(logger, nets)
		if err != nil {
			logger.Error("failed to update DB with server nets", "error", err)
		}

		err = applyNetsToGateway(logger, gtw, nets)
		if err != nil {
			logger.Error("failed to apply nets to gateway", "error", err)
		}

		err = pushNetsToMainServer(parentCtx, logger, conf, oldNets, nets)
		if err != nil {
			logger.Error("failed to update VNets", "error", err)
		}

		// ----- port forwards -----

		portForwards, err := fetchPortForwardsFromMainServer(parentCtx, conf)
		if err != nil {
			logger.Error("failed to get port forwards status", "error", err)

			continue
		}

		err = updateDBWithServerPortForwards(logger, portForwards)
		if err != nil {
			logger.Error("failed to update DB with server port forwards", "error", err)
		}

		err = applyPortForwardsToFirewall(logger, firewall, portForwards)
		if err != nil {
			logger.Error("failed to apply port forwards to firewall", "error", err)
		}
	}
}

// Fetch the main sasso server for the status of the nets
func fetchNetsFromMainServer(parentCtx context.Context, conf config.Server) ([]internal.Net, error) {
	nets, err := internal.FetchNets(parentCtx, conf.Endpoint, conf.Secret)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch nets status from main server: %w", err)
	}

	return nets, nil
}

// pushNetsToMainServer takes care of updating the nets on the main server with
// the correct subnet, gateway and broadcast fields that we have assigned locally.
func pushNetsToMainServer(parentCtx context.Context, logger *slog.Logger, conf config.Server, oldNets, currentNets []internal.Net) error {
	for _, n := range currentNets {
		oldNetIndex := slices.IndexFunc(oldNets, func(on internal.Net) bool {
			return on.Tag == n.Tag
		})

		if oldNetIndex == -1 {
			return fmt.Errorf("failed to find old net with tag %d", n.Tag)
		}

		if oldNets[oldNetIndex].Subnet == n.Subnet && oldNets[oldNetIndex].Gateway == n.Gateway && oldNets[oldNetIndex].Broadcast == n.Broadcast {
			continue
		}

		err := internal.UpdateNet(parentCtx, conf.Endpoint, conf.Secret, n)
		if err != nil {
			return fmt.Errorf("failed to update net on main server: %w", err)
		}

		logger.Info("Updated net on main server", "vnet", n.Name)
	}

	return nil
}

func fetchPortForwardsFromMainServer(parentCtx context.Context, conf config.Server) ([]internal.PortForward, error) {
	pfs, err := internal.FetchPortForwards(parentCtx, conf.Endpoint, conf.Secret)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch port forwards status from main server: %w", err)
	}

	return pfs, nil
}

func updateDBWithServerPortForwards(logger *slog.Logger, pfs []internal.PortForward) error {
	dbPfs := make([]db.PortForward, 0, len(pfs))

	for _, pf := range pfs {
		dbPfs = append(dbPfs, db.PortForward{
			ID:       pf.ID,
			OutPort:  pf.OutPort,
			DestPort: pf.DestPort,
			DestIP:   pf.DestIP,
		})
	}

	err := db.UpdateAllPortForwards(dbPfs)
	if err != nil {
		logger.Error("failed to update all port forwards in database", "error", err)

		return err
	}

	return nil
}

// This function updates the DB with the nets status from the main server.
func updateDBWithServerNets(logger *slog.Logger, nets []internal.Net) error {
	dbInterfaces := make([]db.Interface, 0, len(nets))

	for _, n := range nets {
		dbInterfaces = append(dbInterfaces, db.Interface{
			VNet:      n.Name,
			VNetID:    n.Tag,
			Subnet:    n.Subnet,
			RouterIP:  n.Gateway,
			Broadcast: n.Broadcast,
		})
	}

	err := db.UpdateAllInterfaces(dbInterfaces)
	if err != nil {
		logger.Error("failed to update all interfaces in database", "error", err)

		return err
	}

	return nil
}

// fillNetsEmptyFields fills the subnet, gateway and broadcast fields of the
// nets if they are empty.
func fillNetsEmptyFields(logger *slog.Logger, nets []internal.Net) ([]internal.Net, error) {
	var err error

	subnets := make([]string, 0)

	for i := range nets {
		if nets[i].Subnet == "" {
			nets[i].Subnet, err = utils.NextAvailableSubnetWithNewSubnets(subnets)
			if err != nil {
				logger.Error("failed to get next available subnet", "error", err)

				return nil, err
			}

			subnets = append(subnets, nets[i].Subnet)
		}

		if nets[i].Gateway == "" {
			nets[i].Gateway, err = utils.GatewayAddressFromSubnet(nets[i].Subnet)
			if err != nil {
				logger.Error("failed to get gateway address from subnet", "error", err)

				return nil, err
			}
		}

		if nets[i].Broadcast == "" {
			nets[i].Broadcast, err = utils.GetBroadcastAddressFromSubnet(nets[i].Subnet)
			if err != nil {
				logger.Error("failed to get broadcast address from subnet", "error", err)

				return nil, err
			}
		}
	}

	return nets, nil
}

// applyNetsToGateway applies the nets passed to the gateway. It takes the
// current status of the gateway and deletes the interfaces that are not
// present in the nets slice and creates new interfaces.
func applyNetsToGateway(logger *slog.Logger, gtw gateway.Gateway, nets []internal.Net) error {
	gtwInterfaces, err := gtw.GetAllInterfaces()
	if err != nil {
		logger.Error("failed to get all interfaces from gateway", "error", err)

		return err
	}

	netsMap := make(map[uint32]internal.Net)
	for _, n := range nets {
		netsMap[n.Tag] = n
	}

	gtwInterfacesMap := make(map[uint32]*gateway.Interface)
	for _, i := range gtwInterfaces {
		gtwInterfacesMap[i.VNetID] = i
	}

	// delete interfaces not present in nets slice
	for _, iface := range gtwInterfaces {
		if _, ok := netsMap[iface.VNetID]; ok {
			continue
		}

		err = gtw.RemoveInterface(iface.LocalID)
		if err != nil {
			return fmt.Errorf("failed to remove interface from gateway: %w", err)
		}

		logger.Info("Removed interface from gateway", "name", iface.FirewallInterfaceName)
	}

	// create interfaces present in nets slice but not in gateway
	for _, n := range nets {
		if _, ok := gtwInterfacesMap[n.Tag]; ok {
			continue
		}

		// This should not happen because this function is called after fillNetsEmptyFields,
		// but we check it just in case
		if n.Subnet == "" || n.Gateway == "" || n.Broadcast == "" {
			return fmt.Errorf("net %s has empty fields, cannot create interface on gateway", n.Name)
		}

		_, err := gtw.NewInterface(n.Name, n.Tag, n.Subnet, n.Gateway, n.Broadcast)
		if err != nil {
			return fmt.Errorf("failed to create new interface on gateway: %w", err)
		}

		logger.Info("Created new interface on gateway", "name", n.Name)
	}

	// verify interfaces that are present in both slices
	for _, n := range nets {
		iface, ok := gtwInterfacesMap[n.Tag]
		if !ok {
			continue
		}

		ok, err := gtw.VerifyInterface(iface)
		if err != nil {
			return fmt.Errorf("failed to verify interface on gateway: %w", err)
		}

		if ok {
			continue
		}

		logger.Error("interface on gateway is not consistent with net from main server, recreating it", "vnet", n.Name)

		err = gtw.RemoveInterface(iface.LocalID)
		if err != nil {
			return fmt.Errorf("failed to remove interface from gateway: %w", err)
		}

		logger.Info("Removed inconsistent interface from gateway", "name", iface.FirewallInterfaceName)

		_, err = gtw.NewInterface(n.Name, n.Tag, n.Subnet, n.Gateway, n.Broadcast)
		if err != nil {
			return fmt.Errorf("failed to create new interface on gateway: %w", err)
		}
	}

	return nil
}

// applyPortForwardsToFirewall applies the port forwards passed to the firewall.
// It takes the current status of the firewall and deletes the port forwards
// that are not present in the port forwards slice and creates new port forwards.
func applyPortForwardsToFirewall(logger *slog.Logger, firewall fw.Firewall, wantedRules []internal.PortForward) error {
	currentRules, err := firewall.PortForwardRules()
	if err != nil {
		logger.Error("failed to get all port forward rules from firewall", "error", err)

		return err
	}

	formatRule := func(r fw.Rule) string {
		return fmt.Sprintf("%d-%s-%d", r.OutPort, r.DestIP, r.DestPort)
	}

	currentRulesMap := make(map[string]fw.Rule)
	for _, r := range currentRules {
		currentRulesMap[formatRule(r)] = r
	}

	wantedRulesMap := make(map[string]internal.PortForward)
	for _, pf := range wantedRules {
		wantedRulesMap[formatRule(fw.Rule{
			OutPort:  pf.OutPort,
			DestIP:   pf.DestIP,
			DestPort: pf.DestPort,
		})] = pf
	}

	// Delete port forwards not present in pfs slice
	for _, r := range currentRules {
		if _, ok := wantedRulesMap[formatRule(r)]; ok {
			continue
		}

		err = firewall.RemovePortForwardRule(r)
		if err != nil {
			return fmt.Errorf("failed to remove port forward rule from firewall: %w", err)
		}
	}

	// Create port forwards present in pfs slice but not in firewall
	for _, pf := range wantedRules {
		r := fw.Rule{
			OutPort:  pf.OutPort,
			DestIP:   pf.DestIP,
			DestPort: pf.DestPort,
		}

		if _, ok := currentRulesMap[formatRule(r)]; ok {
			continue
		}

		err = firewall.AddPortForwardRule(r)
		if err != nil {
			return fmt.Errorf("failed to add port forward rule to firewall: %w", err)
		}
	}

	// Verify port forwards that are present in both slices
	for _, pf := range wantedRules {
		r := fw.Rule{
			OutPort:  pf.OutPort,
			DestIP:   pf.DestIP,
			DestPort: pf.DestPort,
		}

		currentRule, ok := currentRulesMap[formatRule(r)]
		if !ok {
			continue
		}

		ok, err := firewall.VerifyPortForwardRule(currentRule)
		if err != nil {
			return fmt.Errorf("failed to verify port forward rule on firewall: %w", err)
		}

		if ok {
			continue
		}

		logger.Error("port forward rule on firewall is not consistent with port forward from main server, recreating it", "rule", r)

		err = firewall.RemovePortForwardRule(currentRule)
		if err != nil {
			return fmt.Errorf("failed to remove port forward rule from firewall: %w", err)
		}

		err = firewall.AddPortForwardRule(r)
		if err != nil {
			return fmt.Errorf("failed to add port forward rule to firewall: %w", err)
		}
	}

	return nil
}
