package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"samuelemusiani/sasso/internal/auth"
	"time"
)

type User struct {
	ID                 uint `json:"id"`
	NumberOFVPNConfigs uint `json:"number_of_vpn_configs"`
}

func FetchUsers(endpoint, secret string) ([]User, error) {
	client := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", endpoint+"/internal/user", nil)
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

	var users []User
	err = json.NewDecoder(res.Body).Decode(&users)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to decode nets status"))
	}

	return users, nil
}
