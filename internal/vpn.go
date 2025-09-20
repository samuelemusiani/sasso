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

func UpdateVPNConfig(endpoint, secret string, vpn VPNUpdate) error {

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
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.Join(err, fmt.Errorf("failed to fetch nets status: non-200 status code. %s", res.Status))
	}

	return nil
}
