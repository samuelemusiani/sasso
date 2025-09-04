package main

import (
	"fmt"
	"os"
	// "samuelemusiani/sasso/vpn/api"
	"samuelemusiani/sasso/vpn/wg"
)

func main() {
	// fmt.Println("Hello, World!")
	// api.Init()
	// err := api.ListenAndServe("0.0.0.0:8080")
	// if err != nil {
	//   fmt.Printf("Error starting server: %v\n", err)
	//   os.Exit(1)
	// }
	config, err := wg.NewWGConfig("10.253.0.4/24", "10.254.0.0/29")
	if err != nil {
		fmt.Printf("Error generating WireGuard config: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Generated WireGuard config:")
	fmt.Println(config)
}
