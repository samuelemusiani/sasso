package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"samuelemusiani/sasso/server/db"
	"samuelemusiani/sasso/server/proxmox"
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

	userID := mustGetUserIDFromContext(r)

	n, err := db.GetNetByID(req.VNetID)
	if err != nil {
		http.Error(w, "vnet not found", http.StatusBadRequest)
		return
	}

	if n.UserID != userID {
		http.Error(w, "vnet does not belong to the user", http.StatusForbidden)
		return
	}

	tmpFace := proxmox.Interface{
		VNetID:  req.VNetID,
		VlanTag: req.VlanTag,
		IPAdd:   req.IPAdd,
		Gateway: req.Gateway,
	}

	if err := proxmox.InterfacesChecks(n, &tmpFace); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	iface, err := proxmox.NewInterface(uint(vm.ID), req.VNetID, req.VlanTag, req.IPAdd, req.Gateway)
	if err != nil {
		if errors.Is(err, proxmox.ErrInvalidVMState) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
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

	userID := mustGetUserIDFromContext(r)

	var piface = proxmox.Interface{
		ID: iface.ID,
	}

	if req.VNetID != nil {
		piface.VNetID = *req.VNetID
	}
	if req.VlanTag != nil {
		piface.VlanTag = *req.VlanTag
	}
	if req.IPAdd != nil {
		piface.IPAdd = *req.IPAdd
	}
	if req.Gateway != nil {
		piface.Gateway = *req.Gateway
	}

	n, err := db.GetNetByID(piface.VNetID)
	if err != nil {
		http.Error(w, "vnet not found", http.StatusBadRequest)
		return
	}

	if n.UserID != userID {
		http.Error(w, "vnet does not belong to the user", http.StatusForbidden)
		return
	}

	if err := proxmox.InterfacesChecks(n, &piface); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := proxmox.UpdateInterface(&piface); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func deleteInterface(w http.ResponseWriter, r *http.Request) {
	iface := getInterfaceFromContext(r)

	if err := proxmox.DeleteInterface(iface.ID); err != nil {
		if errors.Is(err, proxmox.ErrInterfaceNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		} else if errors.Is(err, proxmox.ErrInvalidVMState) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
