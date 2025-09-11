package main

import (
	"fmt"
	"os"

	"samuelemusiani/sasso/vpn/api"
	"samuelemusiani/sasso/vpn/config"
	"samuelemusiani/sasso/vpn/db"
	"samuelemusiani/sasso/vpn/wg"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <config> \n", os.Args[0])
		os.Exit(1)
	}
	err := config.Parse(os.Args[1])
	if err != nil {
		fmt.Printf("Error parsing config: %v\n", err)
		os.Exit(1)
	}
	c := config.Get()
	wg.Init(&c.Wireguard, &c.WBInterfaceName)
	if err = db.Init(&c.Database); err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		os.Exit(1)
	}

	api.Init(&c.Firewall)
	if err = api.ListenAndServe(c.Server.Bind); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		os.Exit(1)
	}
}
