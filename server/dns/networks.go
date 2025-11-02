package dns

import (
	"encoding/json"
	"fmt"
	"net"
)

// Sets the view associated to the given network
func setUpNetworksFromView(view *View) error {
	for _, network := range view.Networks {
		_, _, err := net.ParseCIDR(network)
		if err != nil {
			return fmt.Errorf("Invalid network: %s", network)
		}
		url := fmt.Sprintf("%s/networks/%s", BaseUrl, network)

		body := map[string]interface{}{
			"view": view.Name,
		}

		_, _, err = HttpRequest("PUT", url, body)
		if err != nil {
			return fmt.Errorf("failed to set up network: %w", err)
		}
	}
	return nil
}

func deleteNetwoksFromDNS(networks []string) error {
	for _, network := range networks {
		err := deleteNetworkFromDNS(network)
		if err != nil {
			return fmt.Errorf("failed to delete network from view: %w", err)
		}
	}
	return nil
}

func deleteNetworkFromDNS(network string) error {
	_, _, err := net.ParseCIDR(network)
	if err != nil {
		return fmt.Errorf("Invalid network: %s", network)
	}

	body := map[string]interface{}{
		"view": "",
	}

	url := fmt.Sprintf("%s/networks/%s", BaseUrl, network)

	respBody, statusCode, err := HttpRequest("PUT", url, body)
	if err != nil {
		return fmt.Errorf("failed to delete network: %w", err)
	}

	fmt.Printf("%d Response: %s", statusCode, string(respBody))
	return nil
}

func GetNetworks() ([]network, error) {
	url := fmt.Sprintf("%s/networks", BaseUrl)

	respBody, _, err := HttpRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get networks: %w", err)
	}

	var tmp struct {
		Networks []network `json:"networks"`
	}

	if err := json.Unmarshal(respBody, &tmp); err != nil {
		return nil, fmt.Errorf("failed to parse networks JSON: %w", err)
	}

	return tmp.Networks, nil
}

func populateViewNetworks(viewName string, networks []network) ([]string, error) {
	var viewNets []string
	for _, net := range networks {
		if net.View == viewName {
			viewNets = append(viewNets, net.Network)
		}
	}
	return viewNets, nil
}
