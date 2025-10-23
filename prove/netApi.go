package main

import (
	"encoding/json"
	"fmt"
	"net"
)

// Sets the view associated to the given network
func SetUpNetwork(network Network) error {
	//check network matches net layout
	_, _, err := net.ParseCIDR(network.Network)
	if err != nil {
		return fmt.Errorf("Invalid network: %s", network.Network)
	}

	url := fmt.Sprintf("%s/networks/%s", BaseUrl, network.Network)

	body := map[string]interface{}{
		"view": network.View,
	}

	respBody, statusCode, err := HttpRequest("PUT", url, body)
	if err != nil {
		return fmt.Errorf("failed to set up network: %w", err)
	}

	fmt.Printf("%d Response: %s", statusCode, string(respBody))
	return nil
}

// from dns bash
// network set NET [VIEW]
//
//	Set the view for a network, or delete if no view argument.
func DeleteNetwork(network Network) error {
	_, _, err := net.ParseCIDR(network.Network)
	if err != nil {
		return fmt.Errorf("Invalid network: %s", network.Network)
	}

	body := map[string]interface{}{
		"view": "",
	}

	url := fmt.Sprintf("%s/networks/%s", BaseUrl, network.Network)

	respBody, statusCode, err := HttpRequest("PUT", url, body)
	if err != nil {
		return fmt.Errorf("failed to delete network: %w", err)
	}

	fmt.Printf("%d Response: %s", statusCode, string(respBody))
	return nil
}

func GetNetworks() ([]byte, error) {
	url := fmt.Sprintf("%s/networks", BaseUrl)

	respBody, _, err := HttpRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get networks: %w", err)
	}

	return respBody, nil
}

func PrintNetworks() error {
	body, err := GetNetworks()
	if err != nil {
		return fmt.Errorf("failed to get networks: %w", err)
	}

	var networksResp NetworksResponse
	if err := json.Unmarshal(body, &networksResp); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	fmt.Println("\nNetworks:")
	for _, network := range networksResp.Networks {
		fmt.Printf("Network: %s, View: %s\n", network.Network, network.View)
	}

	return nil
}
