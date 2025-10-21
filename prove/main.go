package main

import (
	// "bytes"
	// "encoding/json"
	"fmt"
	// "io"
	"log"
	// "net/http"
)

var (
	BaseIpAddress = "130.136.201.50"
	BasePort      = 8081
	BaseUrl       = fmt.Sprintf("http://%s:%d/api/v1/servers/localhost", BaseIpAddress, BasePort)
	ApiKey        = "omar"
	MainZone      = "sasso.."
)

func main() {
	// BASILAR ROUTIN TO SET UP/SHUT DOWN A CLIENT
	// view := "client9"
	//
	// var net Network
	// net.Network = "130.136.201.59/32"
	// net.View = view
	//
	// var zone Zone
	// zone.ID = MainZone + view
	// zone.Name = MainZone + view
	// zone.Kind = "Native"
	//
	// Record := Records{
	// 	Content:  "192.168.1.1",
	// 	Disabled: false,
	// }
	//
	// RRSet := RRSet{
	// 	Name:    "pippo.sasso.",
	// 	Records: []Records{Record},
	// 	Type:    "A",
	// 	TTL:     3600,
	// }
	//
	// err := SetUpNetwork(net)
	// if err != nil {
	// 	log.Fatalf("Error setting up network: %v", err)
	// }
	//
	// err = CreateZone(zone)
	// if err != nil {
	// 	// 409 conflict error means ZONE ALREADY EXIST. To be still decided if fatal or not
	// 	if !bytes.Contains([]byte(err.Error()), []byte("409")) {
	// 		log.Fatalf("Error creating zone: %v", err)
	// 	}
	// }
	//
	// err = AddZoneToView(view, zone)
	// if err != nil {
	// 	log.Fatalf("Error adding zone to view: %v", err)
	// }
	//
	// err = NewRRsetInZone(RRSet, zone)
	// if err != nil {
	// 	log.Fatalf("Error creating RRset in zone: %v", err)
	// }
	//
	// err := DeleteRRsetFromZone(RRSet, zone)
	// if err != nil {
	// 	log.Fatalf("Error deleting RRset from zone: %v", err)
	// }
	//
	// err = RemoveZoneFromView(view, zone)
	// if err != nil {
	// 	fmt.Println("Error : ", err)
	// }
	//
	// err = DeleteZone(zone)
	// if err != nil {
	// 	log.Fatalf("Error deleting zone: %v", err)
	// }
	// CAN'T WORK
	// err := DeleteNetwork(net)
	// if err != nil {
	// 	log.Fatalf("Error deleting network: %v", err)
	// }
}
