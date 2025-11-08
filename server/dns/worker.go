package dns

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"samuelemusiani/sasso/server/db"
)

var (
	workerContext    context.Context    = nil
	workerCancelFunc context.CancelFunc = nil
	workerReturnChan chan error         = make(chan error, 1)
)

func StartWorker() {
	workerContext, workerCancelFunc = context.WithCancel(context.Background())
	go func() {
		workerReturnChan <- worker(workerContext)
		close(workerReturnChan)
	}()
}

func ShutdownWorker() error {
	if workerCancelFunc != nil {
		workerCancelFunc()
	}
	var err error = nil
	if workerReturnChan != nil {
		err = <-workerReturnChan
	}
	if err != nil && err != context.Canceled {
		return err
	} else {
		return nil
	}
}

func worker(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(0 * time.Second):
		// Just a small delay to let other components start
	}

	logger.Info("Proxmox worker started")

	timeToWait := 10 * time.Second

	for {
		// Handle graceful shutdown at the start of each cycle
		select {
		case <-ctx.Done():
			logger.Info("Proxmox worker shutting down")
			return ctx.Err()
		case <-time.After(timeToWait):
		}

		now := time.Now()

		// DTODO: This is the main loop that it's executed periodically
		//
		// This loop should check that records and views are present in the DNS.
		// Then delete stale records and views and add missing ones.
		//
		// Records are always of type A (we don't support IPv6 for now).
		//
		// The IPs are on the interface, but the name of the record is the VM
		// name. To retrieve the list of VMs and their IPs, we need a special DB
		// query with a JOIN.
		//
		// A VM could have multiple interfaces, so we need to take the primary one
		// only (the one with the gateway).
		//
		// A view is per VNet. So for the ACLs we must check the VNet subnet and
		// to add a record in the correct view we must check the VNet of the interface.
		//
		// A view is also per User. In the user view all the records of all his VMs
		// must be present. (It's like a sum of all the other views for that user).
		// The network is based on the VPN IP of the user.
		//
		// GROUPS:
		// A view per VNet is still created, and all the Group VMs are added there.
		//
		// For all the members of the group, their user view must also contain the Group VMs.
		// To distinguish Group VMs in the user view, we can add them to a subdomain.
		// For example if a normal VM is "vm1.sasso", a Group VM in the group "devs"
		// should be "vm2.devs.sasso". This is sufficient for the users only, but
		// because it could create some confusion we can also add these records in the
		// views of the Group. So in the Groups view we have "vm2.sasso" and
		// "vm2.devs.sasso". In the user views we have only "vm2.devs.sasso".

		// // The logic here is:
		// // for each user:
		// //  get VPN IPs
		// //  for each VPN IP:
		// //    check if view exists
		// //      if yes: check if all VMs are present, add missing ones
		// //      if no: create view and add all VMs
		// //  get nets
		// //  for each net:
		// //     get VMs
		// //     if net is group:
		// //       check if group view exists
		// //         if yes: check if all VMs are present, add missing ones
		// //         if no: create view and add all VMs
		// //     else:
		// //       check if view exists
		// //         if yes: check if all VMs are present, add missing ones
		// //         if no: create view and add all VMs
		// //
		// //
		// // We need to implement the following functions:
		// //  general;
		// // - GetViewByName(viewName string) (View, error)
		// // - ViewMustContainVMsRecords(view View, vms []VM) error - maybe should be specific for user/group/net?
		// // - GetVMsByNetID(netID int) ([]VM, error)
		// //
		// // 	user related:
		// // - DoesExistViewWithUserName(userName string) bool
		// // - CreateViewForUser(userName string, vpnIP string) (View, error)
		// //
		// // 	groups related:
		// // - DoesExistGroupViewWithNetName(netName string) bool
		// // - CreateViewForGroupNet(netName string) (View, error)
		// // - AddVMRecordsToGroupView(view View, vm VM) error
		// //
		// //  net related:
		// // - DoesExistViewWithNetName(netName string) bool
		// // - CreateViewForNet(netName string) (View, error)
		// // - AddVMRecordsToView(view View, vm VM) error
		// //
		// // Note: maybe DoesExist... functions are not all needed, jus one may be fine
		// //
		// //
		// // and we need to put everything in functions to make the code cleaner.
		//
		// users, err := db.GetAllUsers()
		// if err != nil {
		// 	logger.Error("Error retrieving users from DB", "error", err)
		// 	return err
		// 	//continue ?
		// }
		//
		// for _, user := range users {
		// 	VPNConfigs, err := db.GetVPNConfigsByUserID(user.ID)
		// 	if err != nil {
		// 		logger.Error("Error retrieving VPN config for user from DB", "userID", user.ID, "error", err)
		// 		continue
		// 	}
		//
		// 	// tutte le vm delle networks sono legate all'user id?
		// 	userVMs, err := db.GetVMsByUserID(user.ID)
		// 	if err != nil {
		// 		logger.Error("Error retrieving VMs for user from DB", "userID", user.ID, "error", err)
		// 		continue
		// 	}
		//
		// 	for _, vpnConfig := range VPNConfigs {
		// 		vpnIp := vpnConfig.VPNIP
		// 		view, existsView, err := GetViewByVPNIp(vpnIp)
		// 		if err != nil {
		// 			logger.Error("Error retrieving view for user from DNS", "userID", user.ID, "error", err)
		// 			continue
		// 		}
		//
		// 		if existsView {
		// 			//ensure all VMs are present in the View or in a Zone?
		// 			err := ViewMustContainVMsRecords(&view, userVMs) //to implement
		// 			if err != nil {
		// 				logger.Error("Error ensuring VM records in user view in DNS", "userID", user.ID, "error", err)
		// 				continue
		// 			}
		// 		} else {
		// 			//create view
		//
		// 			newView := View{}
		// 			newView.Name = user.Username
		// 			newView.Network = vpnIp
		// 			newView.Zones = []Zone{} // which zones?
		//
		// 			err := SetupNewViewOnDNS(&newView)
		// 			if err != nil {
		// 				logger.Error("Error creating view for user in DNS", "userID", user.ID, "error", err)
		// 				continue
		// 			}
		//
		// 			// fill the view with user VMs
		// 			for _, vm := range userVMs {
		// 				err := AddVMRecordsToView(&newView, userVMs) //to implement
		// 				if err != nil {
		// 					logger.Error("Error adding VM records to user view in DNS", "userID", user.ID, "vmID", vm.ID, "error", err)
		// 					continue
		// 				}
		// 			}
		// 		}
		// 	}
		//
		// 	nets, err := db.GetNetsByUserID(user.ID)
		// 	if err != nil {
		// 		logger.Error("Error retrieving nets for user from DB", "userID", user.ID, "error", err)
		// 		continue
		// 	}
		//
		// 	for _, net := range nets {
		// 		VMs, err := db.GetVMsByNetID(net.ID) //to implement
		// 		if err != nil {
		// 			logger.Error("Error retrieving VMs for net from DB", "netID", net.ID, "error", err)
		// 			continue
		// 		}
		//
		// 		if 1 /*is a group*/ {
		// 			if DoesExistGroupViewWithNetName(net.Name) { //to implement
		//
		// 				view, err := GetViewByName(net.Name) //to implement
		// 				if err != nil {
		// 					logger.Error("Error retrieving view for group net from DNS", "netID", net.ID, "error", err)
		// 					continue
		// 				}
		//
		// 				err := GroupViewMustContainVMsRecords(view, VMs) //to implement
		// 				if err != nil {
		// 					logger.Error("Error ensuring VM records in group view in DNS", "netID", net.ID, "error", err)
		// 					continue }
		// 			} else {
		// 				//create view
		// 				view, err := CreateViewForGroupNet(net.Name) //to implement}
		// 				if err != nil {
		// 					logger.Error("Error creating view for group net in DNS", "netID", net.ID, "error", err)
		// 					continue
		// 				}
		// 				for _, vm := range VMs {
		// 					err := AddVMRecordsToGroupView(view, vm) //to implement
		// 					if err != nil {
		// 						logger.Error("Error adding VM records to group view in DNS", "netID", net.ID, "vmID", vm.ID, "error", err)
		// 						continue
		// 					}
		// 				}
		// 			}
		// 		} else {
		// 			//normal net view handling
		// 			if DoesExistViewWithNetName(net.Name) { //to implement
		// 				view, err := GetViewByName(net.Name) //to implement
		// 				if err != nil {
		// 					logger.Error("Error retrieving view for net from DNS", "netID", net.ID, "error", err)
		// 					continue
		// 				}
		// 				err := ViewMustContainVMsRecords(view, VMs) //to implement
		// 				if err != nil {
		// 					logger.Error("Error ensuring VM records in net view in DNS", "netID", net.ID, "error", err)
		// 					continue
		// 				}
		// 			} else {
		// 				//create view
		// 				view, err := CreateViewForNet(net.Name) //to implement}
		// 				if err != nil {
		// 					logger.Error("Error creating view for net in DNS", "netID", net.ID, "error", err)
		// 					continue
		// 				}
		// 				for _, vm := range VMs {
		// 					err := AddVMRecordsToView(view, vm) //to implement
		// 					if err != nil {
		// 						logger.Error("Error adding VM records to net view in DNS", "netID", net.ID, "vmID", vm.ID, "error", err)
		// 						continue
		// 					}
		// 				}
		// 			}
		// 		}
		// 	}
		// }

		dnsState, err := getDNSState()
		if err != nil {
			logger.Error("Error retrieving DNS state", "error", err)
			continue
		}

		//
		elapsed := time.Since(now)
		if elapsed < 10*time.Second {
			timeToWait = 10*time.Second - elapsed
		} else {
			timeToWait = 0
		}
	}
}

func getDNSState() (Views, error) {
	views, err := GetStructAllViews()
	if err != nil {
		return Views{}, fmt.Errorf("Error retrieving all views from DNS: %w", err)
	}

	return views, nil
}

func updateDNS(dnsState Views) {

	allNets, err := db.GetAllNets()
	if err != nil {
		logger.Error("Error retrieving all nets from DB", "error", err)
	}

	for _, net := range allNets {
		var view View
		// a view cant have 2 nets in this way
		view.Name = fmt.Sprintf("net%d", net.ID)

		view.Networks = []string{net.Subnet}

		//zone := Zone{
		// 	Name: fmt.Sprintf("sasso..%s", view.Name),
		// 	Zones: []Zone{zone},
		// }
		var zone Zone
		zone.Name = fmt.Sprintf("sasso..%s", view.Name)
		// zones are always one per view
		view.Zones = []Zone{zone}

		vms, err := db.GetVMsWithPrimaryInterfaceInVNet(net.ID)
		if err != nil {
			logger.Error("Error retrieving VMs for net from DB", "netID", net.ID, "error", err)
			continue
		}

		for _, vm := range vms {
			rrset := RRSet{
				Name: fmt.Sprintf("%s.sasso.", vm.VMName),
				Type: "A",
				TTL:  300,
				Records: []Record{
					{Ip: strings.Split(vm.InterfaceIP, "/")[0], Disabled: false},
				},
			}
			view.Zones[0].RRSets = append(view.Zones[0].RRSets, rrset)
		}

		dnsViewIdx := slices.IndexFunc(dnsState.Views, func(v View) bool {
			return v.Name == view.Name
		})

		// check if view exists
		if dnsViewIdx == -1 {
			// if view doesn't exist set all
			err := setupNewStructViewOnDNS(&view)
			if err != nil {
				logger.Error("Error setting up view on DNS for net", "netID", net.ID, "view", view.Name, "error", err)

			}
			continue
		}

		// check if networks match
		// NON USARE slices.Equal perche l'ordine puo' essere diverso
		if !slices.Contains(dnsState.Views[dnsViewIdx].Networks, net.Subnet) {
			// be careful, still need to know if there should be more than one net per structural view
			if len(dnsState.Views[dnsViewIdx].Networks) >= 1 {
				err = deleteNetworksFromDNS(dnsState.Views[dnsViewIdx].Networks)
				if err != nil {
					logger.Error("Error deleting old network on DNS for net", "netID", net.ID, "network", dnsState.Views[dnsViewIdx].Networks, "error", err)
				}
			}

			err = setUpNetworksFromView(&view)
			if err != nil {
				logger.Error("Error setting up networks on DNS for net", "netID", net.ID, "network", view.Networks, "error", err)
			}
			continue
		}

		// check zones
		ConfrontZones(dnsState, view)
	}
	// check for extra views
	// check for extra nets
}

// check rrset and records differences
func ConfrontZones(dnsState Views, dbState Views) bool {
	if len(view.Zones) != len(dnsState.Views[dnsViewIdx].Zones) {
		if len(view.Zones) < 1 { //view has no zones in db; remove zones from dns
			err := deleteZonesFromDNS(dnsState.Views[dnsViewIdx].Zones)
			if err != nil {
				logger.Error("Error deleting zones on DNS for net", "netID", net.ID, "error", err)
			}
		} else {
			if len(dnsState.Views[dnsViewIdx].Zones) < 1 { //view in dns has no zones; add zones from db
				err := createZonesWithRRSets(view.Zones)
				if err != nil {
					logger.Error("Error creating zones on DNS for net", "netID", net.ID, "error", err)
				}
			} else {
				for _, zone := range dnsState.Views[dnsViewIdx].Zones {
					if view.Zones[0].Name != zone.Name { // check if dns zone and db zone correspond
						err := deleteZoneFromDNS(zone)
						if err != nil {
							logger.Error("Error deleting zone on DNS for net", "netID", net.ID, "zone", zone.Name, "error", err)
						}
					}
				}
			}
		}
	} else if view.Zones[0].Name != dnsState.Views[dnsViewIdx].Zones[0].Name { // check if dns zone and db zone correspond
		err := deleteZonesFromDNS(dnsState.Views[dnsViewIdx].Zones)
		if err != nil {
			logger.Error("Error deleting zones on DNS for net", "netID", net.ID, "error", err)
		}
		err = createZonesWithRRSets(view.Zones)
		if err != nil {
			logger.Error("Error creating zones on DNS for net", "netID", net.ID, "error", err)
		}
	} else {
		// check rrset and records differences
		ConfrontRRSets(dnsState.Views[dnsViewIdx].Zones[0].RRSets, view.Zones[0].RRSets)
	}
}

func ConfrontRRSets(dnsRRSets []RRSet, dbRRSets []RRSet) bool {
	if len(dbRRSets) != len(dnsRRSets) {
		if len(dbRRSets) < 1 { //zone has no rrsets in db; remove rrsets from dns
			// I think a db zone of the net always has RRSets, but not sure
		} else {
			if len(dnsRRSets) < 1 { //zone in dns has no rrsets; add rrsets from db
				err := addRRSetsToZone(dbRRSets, view.Zones[0])
				if err != nil {
					logger.Error("Error adding rrsets to zone on DNS for net", "netID", net.ID, "error", err)
				}
			} else {
				for _, dbRRSet := range dbRRSets { // for each rrset in db zone, check if it's in dns zone, add if missing
					if !slices.ContainsFunc(dnsRRSets, func(dnsRRSet RRSet) bool {
						return dnsRRSet.Name == dbRRSet.Name &&
							dnsRRSet.Type == dbRRSet.Type &&
							dnsRRSet.TTL == dbRRSet.TTL &&
							ConfrontRecords(dnsRRSet.Records, dbRRSet.Records)
					}) {
						err := newRRSetInZone(dbRRSet, view.Zones[0])
						if err != nil {
							logger.Error("Error adding rrset to zone on DNS for net", "netID", net.ID, "dbRRSet", dbRRSet, "error", err)
						}
					}
				}
				for _, dnsRRSet := range dnsRRSets { // for each rrset in dns zone, check if it's in db zone, delete if extra
					if !slices.ContainsFunc(dbRRSets, func(dbRRSet RRSet) bool {
						return dnsRRSet.Name == dbRRSet.Name &&
							dnsRRSet.Type == dbRRSet.Type &&
							dnsRRSet.TTL == dbRRSet.TTL &&
							ConfrontRecords(dnsRRSet.Records, dbRRSet.Records)
					}) {
						err := deleteRRSetFromZone(dnsRRSet, view.Zones[0])
						if err != nil {
							logger.Error("Error deleting rrset from zone on DNS for net", "netID", net.ID, "dnsRRSet", dnsRRSet, "error", err)
						}
					}
				}
			}
		}
	} else {
		for _, dbRRSet := range dbRRSets { // for each rrset in db zone, check if it's in dns zone, add if missing
			if !slices.ContainsFunc(dnsRRSets, func(dnsRRSet RRSet) bool {
				return dnsRRSet.Name == dbRRSet.Name &&
					dnsRRSet.Type == dbRRSet.Type &&
					dnsRRSet.TTL == dbRRSet.TTL &&
					ConfrontRecords(dnsRRSet.Records, dbRRSet.Records) //check records too
			}) {
				err := newRRSetInZone(dbRRSet, view.Zones[0])
				if err != nil {
					logger.Error("Error adding rrset to zone on DNS for net", "netID", net.ID, "dbRRSet", dbRRSet, "error", err)
				}
			}
		}
		for _, dnsRRSet := range dnsRRSets { // for each rrset in dns zone, check if it's in db zone, delete if extra
			if !slices.ContainsFunc(dbRRSets, func(dbRRSet RRSet) bool {
				return dnsRRSet.Name == dbRRSet.Name &&
					dnsRRSet.Type == dbRRSet.Type &&
					dnsRRSet.TTL == dbRRSet.TTL &&
					ConfrontRecords(dnsRRSet.Records, dbRRSet.Records) //check records too
			}) {
				err := deleteRRSetFromZone(dnsRRSet, view.Zones[0])
				if err != nil {
					logger.Error("Error deleting rrset from zone on DNS for net", "netID", net.ID, "dnsRRSet", dnsRRSet, "error", err)
				}
			}
		}
	}
}
