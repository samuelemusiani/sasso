package dns

import (
	// "encoding/json"
	"encoding/json"
	"fmt"
	// "samuelemusiani/sasso/server/db"
)

func setupNewStructViewOnDNS(view *View) error {
	err := setUpNetworksFromView(view)
	if err != nil {
		logger.With("error", err).Error("Failed to set up network for view")
		return fmt.Errorf("failed to set up network: %w", err)
	}

	err = createZonesWithRRSets(view.Zones)
	if err != nil {
		logger.With("error", err).Error("Failed to create zones for view")
		return fmt.Errorf("failed to create zones for view: %w", err)
	}

	err = addZonesToView(view)
	if err != nil {
		logger.With("error", err).Error("Failed to add zones to view")
		return fmt.Errorf("failed to add zones to view: %w", err)
	}
	return nil
}

// Adds a zone to a given view, creating it if needed
func addZonesToView(view *View) error {
	for _, zone := range view.Zones {
		url := fmt.Sprintf("%s/views/%s", BaseUrl, view.Name)

		newViewBody := map[string]interface{}{
			"name": zone.Name,
		}

		_, _, err := HttpRequest("POST", url, newViewBody)
		if err != nil {
			logger.With("error", err).Error("Failed to add zone to view")
			return fmt.Errorf("failed to add view: %w", err)
		}
	}
	return nil
}

// // Removes the given zone from the given view
//
//	func RemoveZoneFromView(view string, zone Zone) error {
//		url := fmt.Sprintf("%s/views/%s/%s", BaseUrl, view, zone.ID)
//
//		respBody, statusCode, err := HttpRequest("DELETE", url, nil)
//		if err != nil {
//			logger.With("error", err).Error("Failed to remove zone from view")
//			return fmt.Errorf("failed to remove view: %w", err)
//		}
//
//		fmt.Printf("%d Response: %s", statusCode, string(respBody))
//		return nil
//	}
func GetStructAllViews() (Views, error) {
	url := fmt.Sprintf("%s/views", BaseUrl)

	respBody, _, err := HttpRequest("GET", url, nil)
	if err != nil {
		logger.With("error", err).Error("Failed to list views")
		return Views{}, fmt.Errorf("failed to list views: %w", err)
	}

	var tmp = struct {
		Views []string `json:"views"`
	}{}
	var views Views

	if err := json.Unmarshal(respBody, &tmp); err != nil {
		logger.With("error", err).Error("Failed to parse views JSON")
		return Views{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

	networks, err := GetNetworks()

	for _, viewName := range tmp.Views {
		// view zones
		view, err := GetStructViewWithZonesByName(viewName)
		if err != nil {
			logger.With("error", err).Error("Failed to get view by name")
			return Views{}, fmt.Errorf("failed to get view %s: %w", viewName, err)
		}

		// view networks
		viewNets, err := populateViewNetworks(viewName, networks)

		if err != nil {
			logger.With("error", err).Error("Failed to populate view networks")
			return Views{}, fmt.Errorf("failed to populate networks for view %s: %w", viewName, err)
		}

		view.Networks = viewNets

		views.Views = append(views.Views, view)
	}

	return views, nil
}

func GetStructViewWithZonesByName(viewName string) (View, error) {
	url := fmt.Sprintf("%s/views/%s", BaseUrl, viewName)

	respBody, _, err := HttpRequest("GET", url, nil)
	if err != nil {
		logger.With("error", err).Error("Failed to get view by name")
		return View{}, fmt.Errorf("failed to get view %s: %w", viewName, err)
	}

	var view View
	view.Name = viewName
	var tmp struct {
		Zones []string `json:"zones"`
	}

	if err := json.Unmarshal(respBody, &tmp); err != nil {
		logger.With("error", err).Error("Failed to parse view JSON")
		return View{}, fmt.Errorf("failed to parse JSON for view %s: %w", viewName, err)
	}

	for _, zoneName := range tmp.Zones {
		zone, err := GetStructZoneWithRecordsByName(zoneName)
		if err != nil {
			logger.With("error", err).Error("Failed to get zone by name")
			return View{}, fmt.Errorf("failed to get zone %s: %w", zoneName, err)
		}
		view.Zones = append(view.Zones, zone)
	}

	return view, nil
}

// func GetViewByVPNIp(vpnIp string) (View, bool, error) {
// 	views, err := GetAllViews()
// 	if err != nil {
// 		logger.With("err", err).Error("failed to get views")
// 		return View{}, false, fmt.Errorf("failed to get views: %w", err)
// 	}
//
// 	for _, view := range views.Views {
// 		if view.Network == vpnIp {
// 			return view, true, nil
// 		}
// 	}
// 	return View{}, false, nil
// }
//
// func AddVMRecordsToView(view *View, vms []db.VM) error {
// 	for _, vm := range vms {
// 		// scegliere la zona in cui aggiungere la vm
//
// 	}
// 	return nil
// }
//
// func ViewMustContainVMsRecords(view *View, vms []db.VM) error {
// 	for _, vm := range vms {
// 		found := false
//
// 		var vmPrimaryInterface *db.Interface
// 		// si puo` fare una vm primaria senza gateway?
// 		for _, vmInterface := range vm.Interfaces {
// 			if vmInterface.IPAdd != "" && vmInterface.Gateway != "" {
// 				vmPrimaryInterface = &vmInterface
// 			}
// 		}
// 		if vmPrimaryInterface == nil {
// 			// gestire problema se non c'e` vm primaria
// 		}
//
// 		// possono esserci pi`u zone associate alla stessa view?
// 		for _, zone := range view.Zones {
// 			records, err := GetZoneRecords(&zone)
// 			if err != nil {
// 				logger.With("error", err).Error("Failed to get zone records")
// 				return fmt.Errorf("failed to get zone records for zone %s: %w", zone.ID, err)
// 			}
//
// 			for _, record := range records.Records {
// 				if record.Ip == vmPrimaryInterface.IPAdd { //&& !record.Disabled
// 					found = true
// 				}
// 			}
// 		}
// 		// se non c'è la vm in nessuna zona e` da aggiungere
// 		if found == false {
// 			// se possono esserci piu zone associate alla stessa view quale zona e` da scegliere?
// 			// la zona da scegliere e` quella in cui cade l'ip della vm se la net esiste?
// 		}
// 	}
// 	// se ci sono zone in eccesso da rimuovere?
// 	return nil
// }
