package api

import (
	"encoding/json"
	"net/http"
	"samuelemusiani/sasso/server/db"
	"samuelemusiani/sasso/server/proxmox"

	"github.com/seancfoley/ipaddress-go/ipaddr"
)

func getInterfaces(w http.ResponseWriter, r *http.Request) {
	vm := getVMFromContext(r)

	dbIfaces, err := db.GetInterfacesByVMID(vm.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ifaces := make([]proxmox.Interface, len(dbIfaces))
	for i, dbIface := range dbIfaces {
		ifaces[i] = proxmox.Interface{
			ID:      dbIface.ID,
			VNetID:  dbIface.VNetID,
			VlanTag: dbIface.VlanTag,
			IPAdd:   dbIface.IPAdd,
			Gateway: dbIface.Gateway,
			Status:  proxmox.InterfaceStatus(dbIface.Status),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ifaces)
}

func addInterface(w http.ResponseWriter, r *http.Request) {
	vm := getVMFromContext(r)

	var req struct {
		VNetID  uint   `json:"vnet_id"`
		VlanTag uint16 `json:"vlan_tag"`
		IPAdd   string `json:"ip_add"`
		Gateway string `json:"gateway"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	n, err := db.GetNetByID(req.VNetID)
	if err != nil {
		http.Error(w, "vnet not found", http.StatusBadRequest)
		return
	}

	userID := mustGetUserIDFromContext(r)

	if n.UserID != userID {
		http.Error(w, "vnet does not belong to the user", http.StatusForbidden)
		return
	}

	subnet := ipaddr.NewIPAddressString(n.Subnet)
	reqIPAdd := ipaddr.NewIPAddressString(req.IPAdd)
	if !subnet.Contains(reqIPAdd) {
		http.Error(w, "ip_add not in the subnet of the vnet", http.StatusBadRequest)
		return
	}

	if reqIPAdd.GetNetworkPrefixLen() == nil {
		http.Error(w, "ip_add must have a subnet mask", http.StatusBadRequest)
		return
	}

	reqGateway := ipaddr.NewIPAddressString(req.Gateway)
	if !subnet.Contains(reqGateway) {
		http.Error(w, "gateway not in the subnet of the vnet", http.StatusBadRequest)
		return
	}

	if reqGateway.GetNetworkPrefixLen() != nil {
		http.Error(w, "gateway must not have a subnet mask", http.StatusBadRequest)
		return
	}

	iface, err := proxmox.NewInterface(uint(vm.ID), req.VNetID, req.VlanTag, req.IPAdd, req.Gateway)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(iface)
}

func updateInterface(w http.ResponseWriter, r *http.Request) {
	iface := getInterfaceFromContext(r)

	var req struct {
		VNetID  *uint   `json:"vnet_id"`
		VlanTag *uint16 `json:"vlan_tag"`
		IPAdd   *string `json:"ip_add"`
		Gateway *string `json:"gateway"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.VNetID != nil {
		iface.VNetID = *req.VNetID
	}
	if req.VlanTag != nil {
		iface.VlanTag = *req.VlanTag
	}
	if req.IPAdd != nil {
		iface.IPAdd = *req.IPAdd
	}
	if req.Gateway != nil {
		iface.Gateway = *req.Gateway
	}

	if err := db.UpdateInterface(iface); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func deleteInterface(w http.ResponseWriter, r *http.Request) {
	iface := getInterfaceFromContext(r)

	if err := proxmox.DeleteInterface(iface.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
