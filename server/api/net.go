package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"samuelemusiani/sasso/internal"
	"samuelemusiani/sasso/server/db"
	"samuelemusiani/sasso/server/proxmox"
	"strconv"

	"github.com/go-chi/chi/v5"
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

	Subnet  string `json:"subnet"`
	Gateway string `json:"gateway"`

	GroupID   uint   `json:"group_id,omitempty"` // If the net belongs to a
	GroupName string `json:"group_name,omitempty"`
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
		if err == proxmox.ErrInsufficientResources {
			http.Error(w, "Insufficient resources", http.StatusForbidden)
		} else if err == proxmox.ErrNotFound {
			http.Error(w, "Group not found", http.StatusBadRequest)
		} else if err == proxmox.ErrVNetNameExists {
			http.Error(w, "Network name already exists", http.StatusBadRequest)
		} else if err == proxmox.ErrPermissionDenied {
			http.Error(w, "Permission denied", http.StatusForbidden)
		} else {
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
	json.NewEncoder(w).Encode(returnableNet)
}

func listNets(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	nets, err := db.GetNetsByUserID(userID)
	if err != nil {
		http.Error(w, "Failed to get networks", http.StatusInternalServerError)
		return
	}

	returnableNets := make([]returnNet, len(nets))
	for i, net := range nets {
		returnableNets[i] = returnNet{
			ID:        net.ID,
			Name:      net.Alias,
			Status:    net.Status,
			VlanAware: net.VlanAware,
			Subnet:    net.Subnet,
			Gateway:   net.Gateway,
		}
	}

	// TODO: Optimize this
	groups, err := db.GetGroupsByUserID(userID)
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
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(returnableNets)
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
		if err == proxmox.ErrVNetNotFound {
			http.Error(w, "Net not found", http.StatusNotFound)
		} else if err == proxmox.ErrVNetHasActiveInterfaces {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else if err == proxmox.ErrPermissionDenied {
			http.Error(w, "Permission denied", http.StatusForbidden)
		} else {
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
		if err == proxmox.ErrVNetNotFound {
			http.Error(w, "Net not found", http.StatusNotFound)
		} else if err == proxmox.ErrVNetNameExists || err == proxmox.ErrVNetHasTaggedInterfaces {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else if err == proxmox.ErrPermissionDenied {
			http.Error(w, "Permission denied", http.StatusForbidden)
		} else {
			http.Error(w, "Failed to update net", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func internalListNets(w http.ResponseWriter, r *http.Request) {
	nets, err := db.GetVNetsWithStatus(string(proxmox.VNetStatusReady))
	if err != nil {
		slog.Error("Failed to get all nets", "err", err)
		http.Error(w, "Failed to get networks", http.StatusInternalServerError)
		return
	}

	var returnNets []internal.Net
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
