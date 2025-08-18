package proxmox

import (
	"context"
	"time"
)

func TestEndpointNetZone() {
	time.Sleep(5 * time.Second)
	wasError := false
	first := true

	for {
		if !isProxmoxReachable {
			time.Sleep(20 * time.Second)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cluster, err := client.Cluster(ctx)
		cancel()
		if err != nil {
			logger.Error("Failed to get Proxmox cluster", "error", err)
			time.Sleep(10 * time.Second)
			continue
		}

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		zone, err := cluster.SDNZone(ctx, "sasso")
		cancel()
		if err != nil {
			logger.Error("Failed to get Proxmox SDN cluster zone", "error", err)
			wasError = true
			time.Sleep(10 * time.Second)
			continue
		}

		if zone.Name != "sasso" {
			logger.Error("Proxmox SDN cluster zone name mismatch", "expected", "sasso", "got", zone.Name)
			wasError = true
			time.Sleep(10 * time.Second)
			continue
		}

		if zone.Type != "vxlan" {
			logger.Error("Proxmox SDN cluster zone type mismatch", "expected", "vxlan", "got", zone.Type)
			wasError = true
			time.Sleep(10 * time.Second)
			continue
		}

		if first {
			logger.Info("Proxmox SDN cluster zone is valid", "name", zone.Name, "type", zone.Type)
			first = false
		} else if wasError {
			logger.Info("Proxmox SDN cluster zone is valid again after error", "name", zone.Name, "type", zone.Type)
			wasError = false
		}

		// Should check the state or if it's pending?

		time.Sleep(10 * time.Second)
	}
}
