package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

var (
	BaseIpAddress = "130.136.201.50"
	BasePort      = 8081
	BaseUrl       = fmt.Sprintf("http://%s:%d/api/v1/servers/localhost", BaseIpAddress, BasePort)
	ApiKey        = "omar"
)

func main() {
	// GetAll()
	//CreateNetwork()
	fmt.Println("Running\n")
	//err := AddZoneToView("client1", "example.org..trusted")
	//var net Network
	//net.Network = "18.18.18.18/32"
	//err := SetUpNetwork(net, "vermizio")

	var zone Zone
	zone.ID = "zalone"

	err := RemoveZoneFromView("checco", zone)
	if err != nil {
		fmt.Println("Error : ", err)
	}

}

func IncrementNetwork(network string) string {
	neworkParts := bytes.Split([]byte(network), []byte("/"))
	if len(neworkParts) != 2 {
		log.Fatalf("invalid network format: %s", network)
	}
	ipParts := bytes.Split(neworkParts[0], []byte("."))
	if len(ipParts) != 4 {
		log.Fatalf("invalid IP format: %s", neworkParts[0])
	}

	// Increment the last octet
	lastOctet := ipParts[3]
	lastOctetInt := int(lastOctet[0])
	lastOctetInt++
	if lastOctetInt > 254 {
		log.Fatalf("no more available IPs in the network: %s", network)
	}
	ipParts[3] = []byte(fmt.Sprintf("%d", lastOctetInt))
	newIP := bytes.Join(ipParts, []byte("."))
	return fmt.Sprintf("%s/%s", newIP, neworkParts[1])
}

func HttpPutRequest(url string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest("PUT", url, io.NopCloser(bytes.NewReader(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create PUT request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", ApiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform PUT request: %v", err)
	}
	return resp, nil
}

func CreateNetwork() {
	networksBody, err := GetNetworks()
	if err != nil {
		fmt.Printf("Error getting networks: %v\n", err)
		return
	}

	if i := bytes.IndexByte(networksBody, '{'); i >= 0 {
		networksBody = networksBody[i:]
	}

	var networksResp NetworksResponse
	if err := json.Unmarshal(networksBody, &networksResp); err != nil {
		log.Fatalf("failed to parse JSON: %v", err)
	}

	nets := make([]string, 0, len(networksResp.Networks))
	for _, n := range networksResp.Networks {
		nets = append(nets, n.Network)
	}

	//choose a free network with a more complex logic
	// newNet := FindFreeNetwork(nets)

	fmt.Printf("Existing networks: %v\n", nets)
	netNet := nets[len(nets)-1]
	newNet := IncrementNetwork(netNet)

	fmt.Printf("New network to be added: %s\n", newNet)

	url := BaseUrl + "/networks/" + newNet
	payload := map[string]string{
		"view": "client4",
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error marshalling payload: %v\n", err)
		return
	}
	fmt.Printf("Creating network with payload: %s\n", string(payloadBytes))
	resp, err := HttpPutRequest(url, payloadBytes)
	if err != nil {
		log.Fatalf("Error creating network: %v\n", err)
	}
	defer resp.Body.Close()
	fmt.Printf("Response: %s\n", resp.Status)
}
