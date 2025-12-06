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
		logger.Info("DNS State fetched succesfully")

		updateDNS(dnsState.Views)

		logger.Info("DNS updated")

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

func updateDNS(dnsViews []View) {

	// what we need to do:
	//    for each net retrived with db.GetAllNets() we:
	//        build the view associated with the net including right RRSets from the vms having the net as gateway
	//
	//        check if there is a view in the dns with that name : if not add the view from scratch
	//				[!!!! --- WE SHOULD ALSO DELETE VIEWS NOT PRESENT IN DATABASE --- !!!!]
	//
	//        if the view is in the dns we need to confont dns_view with created_view (database_view)

	nets, err := db.GetAllNets()
	if err == nil {
		err = ValidateUniqueNetNames(nets)
	}
	if err != nil {
		logger.Error("Error retrieving all nets from DB", "error", err)
	}

	for _, net := range nets {

		logger.Debug("Control on net", "network", net.ID)

		//databaseView is what the dnsView must be like
		databaseView, err := buildViewFromNet(net)
		if err != nil {
			logger.Error("Error creating view from net", "error", err, "net", net.Subnet, "view", databaseView.Name)
			continue
		}

		//see if view exists in DNS, if not create a new one based on databaseView
		if !viewsHasViewWithName(databaseView.Name, dnsViews) {
			logger.Debug("Creating new view on dns server", "newView", databaseView.Name)
			err := setupNewStructViewOnDNS(&databaseView)
			if err != nil {
				logger.Error("Error setting up view on DNS for net", "netID", net.ID, "view", databaseView.Name, "error", err)
			}
			continue
		}

		//get view from the dns server and first sync the zones (database and dns) and then rrsets (still database and dns)
		dnsView, err := getViewFromViewsWithName(databaseView.Name, dnsViews)
		if err != nil {
			logger.Error("Error getting DNS view", "error", err)
			continue
		}
		err = syncZones(databaseView.Zones, dnsView.Zones)
		if err != nil {
			logger.Error("Error syncing zones", "error", err, "view", databaseView.Name)
			continue
		}

		//for each zone we sync the rrsets between database and dns
		for _, databaseZone := range databaseView.Zones {
			dnsZone, err := getZoneFromZonesWithName(databaseZone.Name, dnsView.Zones)
			if err != nil {
				logger.Error("Error getting zone", "error", err, "zone", databaseZone.Name)
			}

			err = syncRRSets(databaseZone.RRSets, dnsZone.RRSets, dnsZone)
			if err != nil {
				logger.Error("Error syncing RRSets", "error", err, "view", databaseView.Name, "zone", databaseZone.Name)
			}
		}

	}
}

func syncZones(updatedZones []Zone, behindZones []Zone) error {
	// to sync zones we need first to check the lenght of each arguments
	// if they are different there may be three possibilities :
	//      1.  len(updatedZones) = 0 ->  remove all zones from dns
	//      2.  len(behindZones) = 0 -> we add all zones from updatedView.Zones to dns
	//      3.  eliminiamo dal dns con la API ogni zona non presente in updatedZones ma presente in behindZones ;
	//          e dopo aggiungiamo al dns con la API le zone presenti in updatedZones ma non preesenti in behindZones (mi è venuta da scriverlo in italiano)

	err := ValidateUniqueZoneNames(updatedZones)
	if err != nil {
		return err
	}

	if len(updatedZones) == 0 && len(behindZones) == 0 {
		return nil
	}

	if len(updatedZones) == 0 {
		deleteZonesFromDNS(behindZones)
		return nil
	} else if len(behindZones) == 0 {
		createZonesWithRRSets(updatedZones)
		return nil
	}

	//----NOTICE----- there should be only one zone but i'm not sure about this, so i'm leaving like this; if it's just one zone is easier

	for _, z := range behindZones {
		if !zonesHasZoneWithName(z.Name, updatedZones) {
			logger.Debug("Deleting zone", "zone", z.Name)
			if err := deleteZoneFromDNS(z); err != nil {
				return err
			}
		}
	}

	for _, z := range updatedZones {
		if !zonesHasZoneWithName(z.Name, behindZones) {
			logger.Debug("Creating zone %s", "zone", z.Name)
			if err := createZoneWithRRSets(z); err != nil {
				return err
			}
		}
	}

	return nil
}

func syncRRSets(updatedRRSets []RRSet, behindRRSets []RRSet, dnsZone Zone) error {
	// to sync rrsets we need first to check the lenght of each arguments
	// if they are different there may be three possibilities :
	//      1.  len(updatedRRSets) = 0 ->  remove all rrsets from dns
	//      2.  len(behindRRSets) = 0 -> we add all rrsets from updatedRRSets to dns
	//      3.  eliminiamo dal dns con la API ogni rrset non presente in updatedRRSets ma presente in behindRRSets ;
	//          e dopo aggiungiamo al dns con la API gli rrsets presenti in updatedRRSets ma non preesenti in behindRRSets (mi è venuta da scriverlo in italiano --- ora ho fatto copia e incolla da prima :))

	err := ValidateUniqueRRSetNames(updatedRRSets)
	if err != nil {
		return err
	}

	if len(updatedRRSets) == 0 && len(behindRRSets) == 0 {
		return nil
	}

	if len(updatedRRSets) == 0 {
		logger.Debug("Deleting all RRSets from zone", "zone", dnsZone.Name)
		deleteAllRRSetsFromZone(dnsZone)
		return nil
	} else if len(behindRRSets) == 0 {
		logger.Debug("Copying RRSets into zone %s", "zone", dnsZone.Name)
		addRRSetsToZone(updatedRRSets, dnsZone)
		return nil
	}

	for _, r := range behindRRSets {
		if !rrsetsContainsRRSet(r, updatedRRSets) {
			logger.Debug("Deleting RRSet", "zone", dnsZone.Name, "RRSet", r.Name)
			if err := deleteRRSetFromZone(r, dnsZone); err != nil {
				return err
			}
		}
	}

	for _, r := range updatedRRSets {
		if !rrsetsContainsRRSet(r, behindRRSets) {
			logger.Debug("Adding RRSet", "zone", dnsZone.Name, "RRSet", r.Name)
			logger.Debug("Comparison", "shouldFind", r, "in", behindRRSets)
			if err := newRRSetInZone(r, dnsZone); err != nil {
				return err
			}
		}
	}

	return nil
}

func buildViewFromNet(net db.Net) (View, error) {

	// to build the view we first setup the view and the zone structs
	// then for each vm of the net we create an RRSet in (only) zone of the view

	view := View{
		Name:     fmt.Sprintf("net%d", net.ID),
		Networks: []string{net.Subnet},
	}

	zone := Zone{
		Name: fmt.Sprintf("sasso..%s", view.Name),
	}

	// Attach VMs as RRSets
	vms, err := db.GetVMsWithPrimaryInterfaceInVNet(net.ID)
	if err != nil {
		return View{}, fmt.Errorf("Failed retrieving VMs", "netID", net.ID, "error", err)
	}

	// for each vm we create an RRSet
	for _, vm := range vms {
		ip := strings.Split(vm.InterfaceIP, "/")[0]
		rr := RRSet{
			Name:    fmt.Sprintf("%s.sasso.", vm.VMName),
			Type:    "A",
			TTL:     300,
			Records: []Record{{Ip: ip, Disabled: false}},
		}
		zone.RRSets = append(zone.RRSets, rr)
	}

	view.Zones = []Zone{zone}
	return view, nil
}

func zonesHasZoneWithName(zoneName string, zones []Zone) bool {
	return slices.ContainsFunc(zones, func(z Zone) bool {
		return z.Name == zoneName
	})
}

func viewsHasViewWithName(viewName string, views []View) bool {
	return slices.ContainsFunc(views, func(v View) bool {
		return v.Name == viewName
	})
}

func getViewFromViewsWithName(viewName string, views []View) (View, error) {
	viewIdx := slices.IndexFunc(views, func(v View) bool {
		return v.Name == viewName
	})

	// check if view exists
	if viewIdx == -1 {
		return View{}, fmt.Errorf("View not in list of Views", "viewName", viewName)
	}

	return views[viewIdx], nil
}

func getZoneFromZonesWithName(zoneName string, zones []Zone) (Zone, error) {
	zoneIdx := slices.IndexFunc(zones, func(z Zone) bool {
		return z.Name == zoneName
	})

	// check if zone exists
	if zoneIdx == -1 {
		return Zone{}, fmt.Errorf("Zone not in list of Zone", "zoneName", zoneName)
	}

	return zones[zoneIdx], nil
}

func rrsetsContainsRRSet(rrset RRSet, rrsets []RRSet) bool {
	return slices.ContainsFunc(rrsets, func(r RRSet) bool {
		return rrset.Name == r.Name &&
			rrset.Type == r.Type &&
			rrset.TTL == r.TTL &&
			CompareRecords(rrset, r)
	})
}
