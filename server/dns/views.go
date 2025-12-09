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

	err = newZonesWithRRSets(view.Zones)
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

// Adds a zone to a given view, creating it if needed
func newZoneInView(zone Zone, view View) error {
	url := fmt.Sprintf("%s/views/%s", BaseUrl, view.Name)

	reqBody := map[string]interface{}{
		"name": zone.Name,
		"kind": "Native",
	}

	_, _, err := HttpRequest("POST", url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create zone in view", "err", err)
	}
	for _, rrset := range zone.RRSets {
		err := newRRSetInZone(rrset, zone)
		if err != nil {
			logger.Error("Failed to create RRSet in zone", "zone", zone.Name, "view", view.Name, "RRSet", rrset.Name)
		}
	}

	if err != nil {
		return err
	}

	return nil
}

// Deletes zone from a view
func deleteZoneFromView(zone Zone, view View) error {
	url := fmt.Sprintf("%s/views/%s/%s", BaseUrl, view.Name, zone.Name)

	respBody, statusCode, err := HttpRequest("DELETE", url, nil)
	if err != nil {
		logger.Error("failed to delete zone from view", "zone", zone.Name, "view", view.Name, "err", err)
		return err
	}

	fmt.Printf("%d Response: %s", statusCode, string(respBody))
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
		return View{}, fmt.Errorf("failed to get view", "view", viewName, "err", err)
	}

	var view View
	view.Name = viewName
	var tmp struct {
		Zones []string `json:"zones"`
	}

	if err := json.Unmarshal(respBody, &tmp); err != nil {
		logger.With("error", err).Error("Failed to parse view JSON")
		return View{}, fmt.Errorf("failed to parse JSON for view", "view", viewName, "err", err)
	}

	for _, zoneName := range tmp.Zones {
		zone, err := GetStructZoneWithRecordsByName(zoneName)
		if err != nil {
			logger.Error("Failed to get zone", "zone", zoneName)
			//return View{}, fmt.Errorf("failed to get zone %s: %w", zoneName, err)

			// if we fail to retrive zone we send an empty zone with same name
			zone = Zone{}
			zone.Name = zoneName
			view.Zones = append(view.Zones, zone)
		} else {
			view.Zones = append(view.Zones, zone)
		}
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
