package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"samuelemusiani/sasso/internal"
	"samuelemusiani/sasso/server/db"
	"samuelemusiani/sasso/server/notify"
	"strconv"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/seancfoley/ipaddress-go/ipaddr"
)

type returnPortForward struct {
	ID       uint   `json:"id"`
	OutPort  uint16 `json:"out_port"`
	DestPort uint16 `json:"dest_port"`
	DestIP   string `json:"dest_ip"`
	Approved bool   `json:"approved"`
	Name     string `json:"name,omitempty"`
	IsGroup  bool   `json:"is_group,omitempty"`
}

func returnPortForwardFromDB(pf *db.PortForward) returnPortForward {
	return returnPortForward{
		ID:       pf.ID,
		OutPort:  pf.OutPort,
		DestPort: pf.DestPort,
		DestIP:   pf.DestIP,
		Approved: pf.Approved,
		Name:     pf.Name,
	}
}

func returnPortForwardsFromDB(pfs []db.PortForward) []returnPortForward {
	rpf := make([]returnPortForward, len(pfs))
	for i, pf := range pfs {
		rpf[i] = returnPortForwardFromDB(&pf)
	}
	return rpf
}

func returnAdminPortForwardFromDB(pf *db.PortForward) returnPortForward {
	return returnPortForward{
		ID:       pf.ID,
		OutPort:  pf.OutPort,
		DestPort: pf.DestPort,
		DestIP:   pf.DestIP,
		Approved: pf.Approved,
		Name:     pf.Name,
		IsGroup:  pf.Group,
	}
}

func returnAdminPortForwardsFromDB(pfs []db.PortForward) []returnPortForward {
	rpf := make([]returnPortForward, len(pfs))
	for i, pf := range pfs {
		rpf[i] = returnAdminPortForwardFromDB(&pf)
	}
	return rpf
}

func listPortForwards(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	pfs, err := db.GetPortForwardsByUserID(userID)
	if err != nil {
		http.Error(w, "Failed to get port forwards", http.StatusInternalServerError)
		return
	}

	gpfs, err := db.GetGroupPortForwardsByUserID(userID)
	if err != nil {
		http.Error(w, "Failed to get group port forwards", http.StatusInternalServerError)
		return
	}

	pfs = append(pfs, gpfs...)

	err = json.NewEncoder(w).Encode(returnPortForwardsFromDB(pfs))
	if err != nil {
		http.Error(w, "Failed to encode port forwards", http.StatusInternalServerError)
		return
	}
}

type createPortForwardRequest struct {
	DestPort uint16 `json:"dest_port"`
	DestIP   string `json:"dest_ip"`
}

var randomPortMutex = sync.Mutex{}

func addPortForward(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	var req createPortForwardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.DestPort < 1 || req.DestPort > 65535 {
		http.Error(w, "DestPort must be between 1 and 65535", http.StatusBadRequest)
		return
	}

	addr := ipaddr.NewIPAddressString(req.DestIP)
	if !addr.IsValid() {
		http.Error(w, "DestIP is not a valid IP address", http.StatusBadRequest)
		return
	}

	if addr.IsPrefixed() {
		http.Error(w, "DestIP must be a single IP address, not a subnet", http.StatusBadRequest)
		return
	}

	// There is a time of check/time of use problem here. There is the
	// small possiblity that after checking that the DestIP is in one of the user's
	// subnets, the user deletes that subnet and then adds a port forward to an
	// IP that is no longer in any of their subnets.
	// To avoid this we use a global mutex based on user ID.

	m := getNetMutex(userID)
	m.Lock()
	defer m.Unlock()

	subnets, err := db.GetSubnetsByUserID(userID)
	if err != nil {
		http.Error(w, "Failed to get user subnets", http.StatusInternalServerError)
		return
	}

	foundPersonal, foundGroup := false, false
	foundSubnet := ""
	// Check personal subnets first
	for _, s := range subnets {
		subnet := ipaddr.NewIPAddressString(s)
		ip := ipaddr.NewIPAddressString(req.DestIP)
		if subnet.Contains(ip) && !ip.GetAddress().Equal(subnet.GetAddress().GetLower()) {
			foundPersonal = true
			foundSubnet = s
		}
	}

	if !foundPersonal {
		gsubnets, err := db.GetSubnetsFromGroupsWhereUserIsAdminOrOwner(userID)
		if err != nil {
			http.Error(w, "Failed to get group subnets", http.StatusInternalServerError)
			return
		}
		// Then check group subnets
		for _, s := range gsubnets {
			subnet := ipaddr.NewIPAddressString(s)
			ip := ipaddr.NewIPAddressString(req.DestIP)
			if subnet.Contains(ip) && !ip.GetAddress().Equal(subnet.GetAddress().GetLower()) {
				foundGroup = true
				foundSubnet = s
			}
		}
	}

	if !foundPersonal && !foundGroup {
		http.Error(w, "DestIP is not in any of your subnets", http.StatusBadRequest)
		return
	}

	isGatewayOrBroadcast, err := db.IsAddressAGatewayOrBroadcast(req.DestIP)
	if err != nil {
		http.Error(w, "Failed to check if DestIP is a gateway or broadcast address", http.StatusInternalServerError)
		return
	}
	if isGatewayOrBroadcast {
		http.Error(w, "DestIP cannot be a gateway or broadcast address", http.StatusBadRequest)
		return
	}

	randomPortMutex.Lock()
	defer randomPortMutex.Unlock()

	// TODO: Make this values configurable
	randPort, err := db.GetRandomAvailableOutPort(portForwards.MinPort, portForwards.MaxPort)
	if err != nil {
		http.Error(w, "Failed to get random available out port", http.StatusInternalServerError)
		return
	}

	net, err := db.GetVNetBySubnet(foundSubnet)
	if err != nil {
		http.Error(w, "Failed to get VNet for subnet", http.StatusInternalServerError)
		return
	}

	var pf *db.PortForward
	if net.OwnerType == "Group" {
		pf, err = db.AddPortForwardForGroup(randPort, req.DestPort, req.DestIP, foundSubnet, net.OwnerID)
	} else {
		pf, err = db.AddPortForwardForUser(randPort, req.DestPort, req.DestIP, foundSubnet, userID)
	}

	if err != nil {
		http.Error(w, "Failed to add port forward", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(returnPortForwardFromDB(pf))
	w.WriteHeader(http.StatusCreated)
}

func deletePortForward(w http.ResponseWriter, r *http.Request) {
	userID := mustGetUserIDFromContext(r)

	sportForwardID := chi.URLParam(r, "id")
	portForwardID, err := strconv.ParseUint(sportForwardID, 10, 32)
	if err != nil {
		http.Error(w, "Invalid port forward ID", http.StatusBadRequest)
		return
	}

	pf, err := db.GetPortForwardByID(uint(portForwardID))
	if err != nil {
		http.Error(w, "Port forward not found", http.StatusNotFound)
		return
	}

	if pf.OwnerType == "Group" {
		role, err := db.GetUserRoleInGroup(userID, pf.OwnerID)
		if err != nil {
			http.Error(w, "Failed to get user role in group", http.StatusInternalServerError)
			return
		}
		if role != "owner" && role != "admin" {
			http.Error(w, "Port forward does not belong to the user's group", http.StatusForbidden)
			return
		}
	} else {
		if pf.OwnerID != userID {
			http.Error(w, "Port forward does not belong to the user", http.StatusForbidden)
			return
		}
	}

	if err := db.DeletePortForward(uint(portForwardID)); err != nil {
		http.Error(w, "Failed to delete port forward", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type approvePortForwardRequest struct {
	Approve bool `json:"approve"`
}

func approvePortForward(w http.ResponseWriter, r *http.Request) {
	sportForwardID := chi.URLParam(r, "id")
	portForwardID, err := strconv.ParseUint(sportForwardID, 10, 32)
	if err != nil {
		http.Error(w, "Invalid port forward ID", http.StatusBadRequest)
		return
	}

	var req approvePortForwardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := db.UpdatePortForwardApproval(uint(portForwardID), req.Approve); err != nil {
		http.Error(w, "Failed to approve port forward", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)

	pf, err := db.GetPortForwardByID(uint(portForwardID))
	if err != nil {
		logger.Error("Failed to get port forward after approval", "pfID", portForwardID, "error", err)
		return
	}

	var l *slog.Logger

	if pf.OwnerType == "Group" {
		err = notify.SendPortForwardNotificationToGroup(pf.OwnerID, *pf)
		l = logger.With("groupID", pf.OwnerID)
	} else {
		err = notify.SendPortForwardNotification(pf.OwnerID, *pf)
		l = logger.With("userID", pf.OwnerID)
	}
	if err != nil {
		l.Error("Failed to send port forward notification", "pfID", portForwardID, "error", err)
	}
}

func listAllPortForwards(w http.ResponseWriter, r *http.Request) {
	portForwards, err := db.GetPortForwardsWithNames()
	if err != nil {
		http.Error(w, "Failed to get port forwards", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(returnAdminPortForwardsFromDB(portForwards))
	if err != nil {
		http.Error(w, "Failed to encode port forwards", http.StatusInternalServerError)
		return
	}
}

func internalListProtForwards(w http.ResponseWriter, r *http.Request) {
	portForwards, err := db.GetApprovedPortForwards()
	if err != nil {
		http.Error(w, "Failed to get port forwards", http.StatusInternalServerError)
		return
	}

	rpf := make([]internal.PortForward, len(portForwards))
	for i, pf := range portForwards {
		rpf[i] = internal.PortForward{
			ID:       pf.ID,
			OutPort:  pf.OutPort,
			DestPort: pf.DestPort,
			DestIP:   pf.DestIP,
		}
	}

	err = json.NewEncoder(w).Encode(rpf)
	if err != nil {
		http.Error(w, "Failed to encode port forwards", http.StatusInternalServerError)
		return
	}
}
