package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"samuelemusiani/sasso/internal/auth"
)

func FetchNets(endpoint, secret string) ([]Net, error) {
	client := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", endpoint+"/internal/net", nil)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to create request to fetch nets status"))
	}
	auth.AddAuthToRequest(req, secret)

	res, err := client.Do(req)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to perform request to fetch nets status"))
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.Join(err, fmt.Errorf("failed to fetch nets status: non-200 status code. %s", res.Status))
	}

	var nets []Net
	err = json.NewDecoder(res.Body).Decode(&nets)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to decode nets status"))
	}

	return nets, nil
}

func FetchVPNConfigs(endpoint, secret string) ([]VPNUpdate, error) {
	client := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", endpoint+"/internal/vpn", nil)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to create request to fetch vpn status"))
	}
	auth.AddAuthToRequest(req, secret)

	res, err := client.Do(req)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to perform request to fetch vpn status"))
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.Join(err, fmt.Errorf("failed to fetch nets status: non-200 status code. %s", res.Status))
	}

	var nets []VPNUpdate
	err = json.NewDecoder(res.Body).Decode(&nets)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to decode nets status"))
	}

	return nets, nil
}

func UpdateVPNConfig(endpoint, secret string, vpn VPNUpdate) error {
	client := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("PUT", endpoint+"/internal/vpn", nil)
	if err != nil {
		return errors.Join(err, errors.New("failed to create request to fetch vpn status"))
	}
	auth.AddAuthToRequest(req, secret)

	res, err := client.Do(req)
	if err != nil {
		return errors.Join(err, errors.New("failed to perform request to fetch vpn status"))
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.Join(err, fmt.Errorf("failed to fetch nets status: non-200 status code. %s", res.Status))
	}

	return nil
}
