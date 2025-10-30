package dns

import (
	// "encoding/json"
	"fmt"
	// "samuelemusiani/sasso/server/db"
)

func SetupNewViewOnDNS(view *View) error {
	err := SetUpNetworks(view)
	if err != nil {
		logger.With("error", err).Error("Failed to set up network for view")
		return fmt.Errorf("failed to set up network: %w", err)
	}

	err = CreateZones(view.Zones)
	if err != nil {
		logger.With("error", err).Error("Failed to create zones for view")
		return fmt.Errorf("failed to create zones for view: %w", err)
	}

	err = AddZonesToView(view)
	if err != nil {
		logger.With("error", err).Error("Failed to add zones to view")
		return fmt.Errorf("failed to add zones to view: %w", err)
	}
	return nil
}

// Adds a zone to a given view, creating it if needed
func AddZonesToView(view *View) error {
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
// func RemoveZoneFromView(view string, zone Zone) error {
// 	url := fmt.Sprintf("%s/views/%s/%s", BaseUrl, view, zone.ID)
//
// 	respBody, statusCode, err := HttpRequest("DELETE", url, nil)
// 	if err != nil {
// 		logger.With("error", err).Error("Failed to remove zone from view")
// 		return fmt.Errorf("failed to remove view: %w", err)
// 	}
//
// 	fmt.Printf("%d Response: %s", statusCode, string(respBody))
// 	return nil
// }
//
// func GetAllViews() (*Views, error) {
// 	url := fmt.Sprintf("%s/views", BaseUrl)
//
// 	respBody, _, err := HttpRequest("GET", url, nil)
// 	if err != nil {
// 		logger.With("error", err).Error("Failed to list views")
// 		return nil, fmt.Errorf("failed to list views: %w", err)
// 	}
//
// 	var viewsResp Views
//
// 	if err := json.Unmarshal(respBody, &viewsResp); err != nil {
// 		logger.With("error", err).Error("Failed to parse views JSON")
// 		return nil, fmt.Errorf("failed to parse JSON: %w", err)
// 	}
//
// 	return &viewsResp, nil
// }
//
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
