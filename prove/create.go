package main

import (
	"fmt"
	"encoding/json"
	"bytes"
	"net/http"
	"io"
	"net"
)


func HttpRequest(method , url string, body interface{}) ([]byte, int, error) {
	//check body format
	var bodyReader io.Reader
	if method == "POST" || method == "PUT"{
		b, err := json.Marshal(body)
		if err != nil{
			return nil, -1, fmt.Errorf("failed reading body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method , url, bodyReader)
	if err != nil {
		return nil, -1, fmt.Errorf("failed to create request: %w", err)
	}

	//check if need more headers
	req.Header.Set("X-API-Key", ApiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, -1, fmt.Errorf("failed to perform request: %w", err)
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil{
		return nil, -1, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400{
		return nil, nil, fmt.Errorf("HTTP %d: %s \n", resp.StatusCode, respBytes)
	}
	
	return respBytes, resp.StatusCode, nil
}

//Adds a zone to a given view, creating it if needed
func AddZoneToView(view string, zone string) error {
	url := 	fmt.Sprintf("%s/views/%s", BaseUrl , view)

	newViewBody := map[string]interface{}{
        "name" : zone,
    }

	respBody, statusCode, err := HttpRequest("POST", url, newViewBody)
	if err != nil{
		return fmt.Errorf("failed to add view: %w", err)
	}

	fmt.Println("%d Response: %s", statusCode, string(respBody))
	return nil
}

//Sets the view associated to the given network
func SetUpNetwork(network string, view string) error {
	//check network matches net layout
	 _, _, err := net.ParseCIDR(network)
    if err != nil{
		return fmt.Errorf("Invalid network: %s", network) 
	}

	url := 	fmt.Sprintf("%s/networks/%s", BaseUrl , network)

	body := map[string]interface{}{
        "view" : view,
    }

	respBody, statusCode, err := HttpRequest("PUT", url, body)
	if err != nil{
		return fmt.Errorf("failed to set up network: %w", err)
	}

	fmt.Println("%d Response: %s", statusCode, string(respBody))
	return nil
}

