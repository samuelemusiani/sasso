package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/seancfoley/ipaddress-go/ipaddr"
	"samuelemusiani/sasso/internal"
	"samuelemusiani/sasso/server/db"
	"samuelemusiani/sasso/server/proxmox"
)

type createNetRequest struct {
	Name      string `json:"name"`
	VlanAware bool   `json:"vlanaware"`
	GroupID   *uint  `json:"group_id,omitempty"`
}

type returnNet struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	VlanAware bool   `json:"vlanaware"`

	Subnet    string `json:"subnet"`
	Gateway   string `json:"gateway"`
	Broadcast string `json:"broadcast"`

	GroupID   uint   `json:"group_id,omitempty"` // If the net belongs to a
	GroupName string `json:"group_name,omitempty"`
	GroupRole string `json:"group_role,omitempty"`
}

func createNet(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	var req createNetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)

		return
	}

	if req.Name == "" {
		http.Error(w, "Network name is required", http.StatusBadRequest)

		return
	}

	net, err := proxmox.CreateNewNet(userID, req.Name, req.VlanAware, req.GroupID)
	if err != nil {
		switch {
		case errors.Is(err, proxmox.ErrInsufficientResources):
			http.Error(w, "Insufficient resources", http.StatusForbidden)
		case errors.Is(err, proxmox.ErrNotFound):
			http.Error(w, "Group not found", http.StatusBadRequest)
		case errors.Is(err, proxmox.ErrVNetNameExists):
			http.Error(w, "Network name already exists", http.StatusBadRequest)
		case errors.Is(err, proxmox.ErrPermissionDenied):
			http.Error(w, "Permission denied", http.StatusForbidden)
		default:
			http.Error(w, "Failed to create network", http.StatusInternalServerError)
		}

		return
	}

	returnableNet := returnNet{
		ID:        net.ID,
		Name:      net.Alias, // This is correct. For the user the name is the alias.
		Status:    net.Status,
		VlanAware: net.VlanAware,
	}

	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(returnableNet)
	if err != nil {
		logger.Error("Failed to encode new net response", "error", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)

		return
	}
}

func listNets(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	nets, err := db.GetNetsByUserID(userID)
	if err != nil {
		http.Error(w, "Failed to get networks", http.StatusInternalServerError)

		return
	}

	f := func(s, errMsg string) string {
		tmp := strings.SplitN(s, "/", 2)
		if len(tmp) != 2 {
			logger.Error(errMsg, "value", s)

			return s
		}

		return tmp[0]
	}

	returnableNets := make([]returnNet, 0, len(nets))
	for _, net := range nets {
		var gtw, broad string
		if net.Gateway == "" && net.Broadcast == "" {
			// This is a new net and the gateway and broadcast have not been set yet.
			// To avoid logging errors, we just return empty strings.
			gtw = ""
			broad = ""
		} else {
			gtw = f(net.Gateway, "Invalid gateway format")
			broad = f(net.Broadcast, "Invalid broadcast format")
		}

		returnableNets = append(returnableNets, returnNet{
			ID:        net.ID,
			Name:      net.Alias,
			Status:    net.Status,
			VlanAware: net.VlanAware,
			Subnet:    net.Subnet,
			Gateway:   gtw,
			Broadcast: broad,
		})
	}

	groups, err := db.GetGroupsByUserID(userID)
	if err != nil {
		logger.Error("Failed to get groups by user ID", "userID", userID, "err", err)
		http.Error(w, "Failed to get networks", http.StatusInternalServerError)

		return
	}

	for _, g := range groups {
		groupNets, err := db.GetNetsByGroupID(g.ID)
		if err != nil {
			http.Error(w, "Failed to get networks", http.StatusInternalServerError)

			return
		}

		for _, net := range groupNets {
			returnableNets = append(returnableNets, returnNet{
				ID:        net.ID,
				Name:      net.Alias,
				Status:    net.Status,
				VlanAware: net.VlanAware,
				Subnet:    net.Subnet,
				Gateway:   net.Gateway,
				GroupID:   g.ID,
				GroupName: g.Name,
				GroupRole: g.Role,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(returnableNets)
	if err != nil {
		logger.Error("Failed to encode nets", "error", err)
		http.Error(w, "Failed to encode networks", http.StatusInternalServerError)

		return
	}
}

func deleteNet(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	netIDStr := chi.URLParam(r, "id")

	netID, err := strconv.ParseUint(netIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid net ID", http.StatusBadRequest)

		return
	}

	m := getNetMutex(userID)

	m.Lock()
	defer m.Unlock()

	if err := proxmox.DeleteNet(userID, uint(netID)); err != nil {
		switch {
		case errors.Is(err, proxmox.ErrVNetNotFound):
			http.Error(w, "Net not found", http.StatusNotFound)
		case errors.Is(err, proxmox.ErrVNetHasActiveInterfaces):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, proxmox.ErrPermissionDenied):
			http.Error(w, "Permission denied", http.StatusForbidden)
		default:
			http.Error(w, "Failed to delete net", http.StatusInternalServerError)
		}

		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func updateNet(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	vnetIDStr := chi.URLParam(r, "id")

	vnetID, err := strconv.ParseUint(vnetIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid net ID", http.StatusBadRequest)

		return
	}

	var req createNetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)

		return
	}

	err = proxmox.UpdateNet(userID, uint(vnetID), req.Name, req.VlanAware)
	if err != nil {
		switch {
		case errors.Is(err, proxmox.ErrVNetNotFound):
			http.Error(w, "Net not found", http.StatusNotFound)
		case errors.Is(err, proxmox.ErrVNetNameExists), errors.Is(err, proxmox.ErrVNetHasTaggedInterfaces):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, proxmox.ErrPermissionDenied):
			http.Error(w, "Permission denied", http.StatusForbidden)
		default:
			http.Error(w, "Failed to update net", http.StatusInternalServerError)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type requestIPCheck struct {
	VNetID  uint   `json:"vnet_id"`
	VlanTag uint16 `json:"vlan_tag"`
	IP      string `json:"ip"`
}

type responseIPCheck struct {
	InUse bool `json:"in_use"`
}

func checkIfIPInUse(w http.ResponseWriter, r *http.Request) {
	var req requestIPCheck
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)

		return
	}

	req.IP = strings.TrimSpace(req.IP)

	if req.IP == "" {
		http.Error(w, "IP address is required", http.StatusBadRequest)

		return
	}

	reqIPAdd := ipaddr.NewIPAddressString(req.IP)
	if !reqIPAdd.IsValid() {
		http.Error(w, "Invalid IP address format", http.StatusBadRequest)

		return
	}

	if req.VlanTag > 4095 {
		http.Error(w, "VLAN tag must be between 0 and 4095", http.StatusBadRequest)

		return
	}

	vnet, err := db.GetNetByID(req.VNetID)
	if err != nil {
		slog.Error("Failed to get VNet by ID", "vnetID", req.VNetID, "err", err)

		if errors.Is(err, db.ErrNotFound) {
			http.Error(w, "VNet not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to get VNet", http.StatusInternalServerError)
		}

		return
	}

	if !vnet.VlanAware && req.VlanTag != 0 {
		http.Error(w, "VLAN tag must be 0 for non-VLAN-aware VNets", http.StatusBadRequest)

		return
	}

	userID := mustGetUserIDFromContext(r)
	if vnet.OwnerType == "User" && vnet.OwnerID != userID {
		http.Error(w, "VNet does not belong to the user", http.StatusForbidden)

		return
	} else if vnet.OwnerType == "Group" {
		_, err := db.GetUserRoleInGroup(userID, vnet.OwnerID)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				http.Error(w, "Group not found or user not in group", http.StatusBadRequest)

				return
			}

			slog.Error("Failed to get user role in group", "userID", userID, "groupID", vnet.OwnerID, "err", err)
			http.Error(w, "Failed to check permissions", http.StatusInternalServerError)

			return
		}
	}

	used, err := db.ExistsIPInVNetWithVlanTag(req.VNetID, req.VlanTag, req.IP)
	if err != nil {
		slog.Error("Failed to check if IP is in use", "err", err)
		http.Error(w, "Failed to check IP", http.StatusInternalServerError)

		return
	}

	if err := json.NewEncoder(w).Encode(responseIPCheck{InUse: used}); err != nil {
		slog.Error("Failed to encode IP check response", "err", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)

		return
	}
}

func internalListNets(w http.ResponseWriter, _ *http.Request) {
	nets, err := db.GetVNetsWithStatus(string(proxmox.VNetStatusReady))
	if err != nil {
		slog.Error("Failed to get all nets", "err", err)
		http.Error(w, "Failed to get networks", http.StatusInternalServerError)

		return
	}

	returnNets := make([]internal.Net, 0, len(nets))
	for _, n := range nets {
		var users []uint

		if n.OwnerType == "Group" {
			groupUsers, err := db.GetUserIDsByGroupID(n.OwnerID)
			if err != nil {
				slog.Error("Failed to get users by group ID", "groupID", n.OwnerID, "err", err)
				http.Error(w, "Failed to get networks", http.StatusInternalServerError)

				return
			}

			users = append(users, groupUsers...)
		} else {
			// OwnerType == "User"
			users = append(users, n.OwnerID)
		}

		returnNets = append(returnNets, internal.Net{
			ID:        n.ID,
			Zone:      n.Zone,
			Name:      n.Name,
			Tag:       n.Tag,
			Subnet:    n.Subnet,
			Gateway:   n.Gateway,
			Broadcast: n.Broadcast,
			UserIDs:   users,
		})
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(returnNets)
	if err != nil {
		slog.Error("Failed to encode nets", "err", err)
		http.Error(w, "Failed to encode networks", http.StatusInternalServerError)

		return
	}
}

func internalUpdateNet(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		slog.Error("Invalid net ID", "err", err)
		http.Error(w, "Invalid net ID", http.StatusBadRequest)

		return
	}

	var n internal.Net
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		slog.Error("Failed to decode net", "err", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)

		return
	}

	dbNet, err := db.GetNetByID(uint(id))
	if err != nil {
		slog.Error("Failed to get net by ID", "netID", id, "err", err)
		http.Error(w, "Net not found", http.StatusNotFound)

		return
	}

	dbNet.Subnet = n.Subnet
	dbNet.Gateway = n.Gateway
	dbNet.Broadcast = n.Broadcast

	if err := db.UpdateVNet(dbNet); err != nil {
		slog.Error("Failed to update net", "netID", id, "err", err)
		http.Error(w, "Failed to update net", http.StatusInternalServerError)

		return
	}
}
