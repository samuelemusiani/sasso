package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"samuelemusiani/sasso/server/db"
	"samuelemusiani/sasso/server/proxmox"
	"strings"
	"time"
)

func getInterfacesForVM(w http.ResponseWriter, r *http.Request) {
	vm := mustGetVMFromContext(r)

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
	vm := mustGetVMFromContext(r)

	if vm.OwnerType == "Group" {
		role := mustGetUserRoleInGroupFromContext(r)
		if role == "member" {
			http.Error(w, "user does not have permission to add interface to this VM", http.StatusForbidden)
			return
		}
	}

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

	req.IPAdd = strings.TrimSpace(req.IPAdd)
	req.Gateway = strings.TrimSpace(req.Gateway)

	if vm.LifeTime.Before(time.Now()) {
		http.Error(w, "Cannot add interface to expired VM", http.StatusConflict)
		return
	}

	userID := mustGetUserIDFromContext(r)

	n, err := db.GetNetByID(req.VNetID)
	if err != nil {
		http.Error(w, "vnet not found", http.StatusBadRequest)
		return
	}

	if n.OwnerType == "User" {
		if n.OwnerID != userID {
			http.Error(w, "vnet does not belong to the user", http.StatusForbidden)
			return
		} else if vm.OwnerType == "Group" || vm.OwnerID != n.OwnerID {
			http.Error(w, "VM does not belong to the same user as the vnet", http.StatusForbidden)
			return
		}
	} else if n.OwnerType == "Group" {
		role, err := db.GetUserRoleInGroup(userID, n.OwnerID)
		if err != nil {
			if err == db.ErrNotFound {
				http.Error(w, "group not found or user not in group", http.StatusBadRequest)
				return
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if role == "member" {
			http.Error(w, "user does not have permission to use this vnet", http.StatusForbidden)
			return
		}

		// This check ensures that interfaces can only be added to VMs that belong
		// to the same group as the vnet.
		if vm.OwnerType != "Group" || vm.OwnerID != n.OwnerID {
			http.Error(w, "VM does not belong to the same group as the vnet", http.StatusForbidden)
			return
		}
	}

	tmpFace := proxmox.Interface{
		VNetID:  req.VNetID,
		VlanTag: req.VlanTag,
		IPAdd:   req.IPAdd,
		Gateway: req.Gateway,
	}

	if tmpFace.Gateway != "" {
		// We need to check if there is already another interface with a gateway
		// on the same VM.
		mutex := getVMMutex(uint(vm.ID))
		mutex.Lock()
		defer mutex.Unlock()

		interfaces, err := db.GetInterfacesByVMID(vm.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, iface := range interfaces {
			if iface.Gateway != "" {
				http.Error(w, "only one interface with gateway allowed per VM", http.StatusBadRequest)
				return
			}
		}
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
	vm := mustGetVMFromContext(r)

	if vm.OwnerType == "Group" {
		role := mustGetUserRoleInGroupFromContext(r)
		if role == "member" {
			http.Error(w, "user does not have permission to update interface to this VM", http.StatusForbidden)
			return
		}
	}

	if vm.LifeTime.Before(time.Now()) {
		http.Error(w, "Cannot modify interface in expired VM", http.StatusConflict)
		return
	}

	iface := mustGetInterfaceFromContext(r)

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

	if n.OwnerType == "User" && n.OwnerID != userID {
		http.Error(w, "vnet does not belong to the user", http.StatusForbidden)
		return
	} else if n.OwnerType == "Group" {
		role, err := db.GetUserRoleInGroup(userID, n.OwnerID)
		if err != nil {
			if err == db.ErrNotFound {
				http.Error(w, "group not found or user not in group", http.StatusBadRequest)
				return
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if role == "member" {
			http.Error(w, "user does not have permission to use this vnet", http.StatusForbidden)
			return
		}
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
	vm := mustGetVMFromContext(r)

	if vm.OwnerType == "Group" {
		role := mustGetUserRoleInGroupFromContext(r)
		if role == "member" {
			http.Error(w, "user does not have permission to delete interface to this VM", http.StatusForbidden)
			return
		}
	}

	if vm.LifeTime.Before(time.Now()) {
		http.Error(w, "Cannot delete interface in expired VM", http.StatusConflict)
		return
	}

	iface := mustGetInterfaceFromContext(r)

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

type returnedGenericInterface struct {
	ID       uint   `json:"id"`
	VNetID   uint   `json:"vnet_id"`
	VNetName string `json:"vnet_name"`
	VlanTag  uint16 `json:"vlan_tag"`
	IPAdd    string `json:"ip_add"`
	Gateway  string `json:"gateway"`
	Status   string `json:"status"`
	VMID     uint   `json:"vm_id"`
	VMName   string `json:"vm_name"`

	GroupID   uint   `json:"group_id,omitempty"`
	GroupName string `json:"group_name,omitempty"`
	// User role in the group (e.g., "member", "admin").
	// User is the one requesting the VM.
	GroupRole string `json:"group_role,omitempty"`
}

func getAllInterfaces(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)
	ifaces, err := db.GetAllInterfacesWithExtrasByUserID(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pIfaces := make([]returnedGenericInterface, len(ifaces))
	for i, iface := range ifaces {
		pIfaces[i] = returnedGenericInterface{
			ID:        iface.ID,
			VNetID:    iface.VNetID,
			VNetName:  iface.VNetName,
			VlanTag:   iface.VlanTag,
			IPAdd:     iface.IPAdd,
			Gateway:   iface.Gateway,
			Status:    iface.Status,
			VMID:      uint(iface.VMID),
			VMName:    iface.VMName,
			GroupID:   iface.GroupID,
			GroupName: iface.GroupName,
			GroupRole: iface.GroupRole,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pIfaces)
}
