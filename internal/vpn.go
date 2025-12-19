package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"samuelemusiani/sasso/internal/auth"
)

func FetchVPNConfigs(endpoint, secret string) (vpns []VPNProfile, err error) {
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

	defer func() {
		if e := res.Body.Close(); e != nil {
			err = fmt.Errorf("error while closing request body: %w", e)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return nil, errors.Join(err, fmt.Errorf("failed to fetch nets status: non-200 status code. %s", res.Status))
	}

	err = json.NewDecoder(res.Body).Decode(&vpns)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to decode nets status"))
	}

	return
}

func UpdateVPNConfig(endpoint, secret string, vpn VPNProfile) (err error) {
	body, err := json.Marshal(vpn)
	if err != nil {
		return errors.Join(err, errors.New("failed to marshal vpn update"))
	}

	client := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("PUT", endpoint+"/internal/vpn", bytes.NewBuffer(body))
	if err != nil {
		return errors.Join(err, errors.New("failed to create request to fetch vpn status"))
	}
	auth.AddAuthToRequest(req, secret)

	res, err := client.Do(req)
	if err != nil {
		return errors.Join(err, errors.New("failed to perform request to fetch vpn status"))
	}

	defer func() {
		if e := res.Body.Close(); e != nil {
			err = fmt.Errorf("error while closing request body: %w", e)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return errors.Join(err, fmt.Errorf("failed to fetch nets status: non-200 status code. %s", res.Status))
	}

	return
}
