package api

import (
	"context"
	"errors"
	"net/http"
	"samuelemusiani/sasso/server/db"
	"samuelemusiani/sasso/server/proxmox"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
)

func getUserIDFromContext(r *http.Request) (uint, error) {
	_, claims, err := jwtauth.FromContext(r.Context())
	if err != nil {
		logger.Error("Failed to get claims from context", "error", err)
		return 0, err
	}
	// All JSON numbers are decoded into float64 by default
	userID, ok := claims[CLAIM_USER_ID].(float64)
	if !ok {
		logger.Error("User ID claim not found or not a float64", "claims", claims[CLAIM_USER_ID])
		return 0, errors.New("user ID claim not found or not a float64")
	}

	return uint(userID), nil
}

func mustGetUserIDFromContext(r *http.Request) uint {
	id, err := getUserIDFromContext(r)
	if err != nil {
		panic("mustGetUserIDFromContext: " + err.Error())
	}
	return id
}

// AdminAuthenticator is an authentication middleware to enforce access from the
// Verifier middleware request context values. The Authenticator sends a 401 Unauthorized
// response for any unverified tokens and passes the good ones through.
func AdminAuthenticator(ja *jwtauth.JWTAuth) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		hfn := func(w http.ResponseWriter, r *http.Request) {
			token, claims, err := jwtauth.FromContext(r.Context())

			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			if token == nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			userID, ok := claims[CLAIM_USER_ID].(float64)
			if !ok {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			user, err := db.GetUserByID(uint(userID))
			if err != nil {
				if err == db.ErrNotFound {
					http.Error(w, "User not found", http.StatusUnauthorized)
					return
				}
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Check if the user has admin role
			if user.Role != db.RoleAdmin {
				http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
				return
			}

			// Token is authenticated, pass it through
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(hfn)
	}
}

// This middleware validates that the VM being accessed belongs to the user making the request.
// DOES NOT CHECK PERMISSIONS INSIDE GROUPS, THAT MUST BE DONE IN THE HANDLERS.
func validateVMOwnership() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		hfn := func(w http.ResponseWriter, r *http.Request) {
			userID, err := getUserIDFromContext(r)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			svmID := chi.URLParam(r, "vmid")
			vmID, err := strconv.ParseUint(svmID, 10, 64)
			if err != nil {
				http.Error(w, "invalid vm id", http.StatusBadRequest)
				return
			}

			vm, err := proxmox.GetVMByID(vmID, userID)
			if err != nil {
				http.Error(w, "vm not found", http.StatusNotFound)
				return
			}

			var role string
			if vm.OwnerType == "User" && vm.OwnerID != userID {
				http.Error(w, "vm does not belong to the user", http.StatusForbidden)
				return
			}
			if vm.OwnerType == "Group" {
				role, err = db.GetUserRoleInGroup(userID, vm.OwnerID)
				if err != nil {
					if errors.Is(err, db.ErrNotFound) {
						http.Error(w, "vm does not belong to the user", http.StatusForbidden)
						return
					}
					logger.Error("failed to get user role in group", "error", err)
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
				// Role permission check is done in the handlers
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, "vm_id", vm)
			ctx = context.WithValue(ctx, "group_user_role", role)
			ctx = context.WithValue(ctx, "group_id", vm.OwnerID)

			next.ServeHTTP(w, r.WithContext(ctx))
		}

		return http.HandlerFunc(hfn)
	}
}

func mustGetVMFromContext(r *http.Request) *proxmox.VM {
	vm, ok := r.Context().Value("vm_id").(*proxmox.VM)
	if !ok {
		panic("getVMFromContext: vm_id not found in context")
	}
	return vm
}

func mustGetUserRoleInGroupFromContext(r *http.Request) string {
	role, ok := r.Context().Value("group_user_role").(string)
	if !ok {
		panic("mustGetUserRoleInGroupFromContext: group_user_role not found in context")
	}
	return role
}

// IMPORTANT: This only works under /vm/ paths
func mustGetGroupIDFromContext(r *http.Request) uint {
	groupID, ok := r.Context().Value("group_id").(uint)
	if !ok {
		panic("mustGetGroupIDFromContext: group_id not found in context")
	}
	return groupID
}

func validateInterfaceOwnership() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		hfn := func(w http.ResponseWriter, r *http.Request) {
			userID, err := getUserIDFromContext(r)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			sifaceID := chi.URLParam(r, "ifaceid")
			ifaceID, err := strconv.ParseUint(sifaceID, 10, 32)
			if err != nil {
				http.Error(w, "invalid interface id", http.StatusBadRequest)
				return
			}

			iface, err := db.GetInterfaceByID(uint(ifaceID))
			if err != nil {
				http.Error(w, "interface not found", http.StatusBadRequest)
				return
			}

			n, err := db.GetNetByID(iface.VNetID)
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
					if errors.Is(err, db.ErrNotFound) {
						http.Error(w, "vnet does not belong to the user", http.StatusForbidden)
						return
					}
					logger.Error("failed to get user role in group", "error", err)
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}

				if role == "member" {
					http.Error(w, "user does not have permission to use this vnet", http.StatusForbidden)
					return
				}
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, "interface_id", iface)

			next.ServeHTTP(w, r.WithContext(ctx))
		}

		return http.HandlerFunc(hfn)
	}
}

func mustGetInterfaceFromContext(r *http.Request) *db.Interface {
	iface, ok := r.Context().Value("interface_id").(*db.Interface)
	if !ok {
		panic("getInterfaceFromContext: interface_id not found in context")
	}
	return iface
}

func validateGroupOwnership() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		hfn := func(w http.ResponseWriter, r *http.Request) {
			userID, err := getUserIDFromContext(r)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			sgroupID := chi.URLParam(r, "groupid")
			groupID, err := strconv.ParseUint(sgroupID, 10, 32)
			if err != nil {
				http.Error(w, "invalid group id", http.StatusBadRequest)
				return
			}

			group, err := db.GetGroupByID(uint(groupID))
			if err != nil {
				if errors.Is(err, db.ErrNotFound) {
					http.Error(w, "group not found", http.StatusBadRequest)
					return
				}
				logger.Error("failed to get group by id", "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			userRole, err := db.GetUserRoleInGroup(userID, group.ID)
			if err != nil {
				if errors.Is(err, db.ErrNotFound) {
					http.Error(w, "group not found", http.StatusNotFound)
					return
				}
				logger.Error("failed to get user role in group", "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, "group", group)
			ctx = context.WithValue(ctx, "group_user_role", userRole)

			next.ServeHTTP(w, r.WithContext(ctx))
		}

		return http.HandlerFunc(hfn)
	}
}

// IMPORTANT: This only works under /group/ paths
func mustGetGroupFromContext(r *http.Request) *db.Group {
	group, ok := r.Context().Value("group").(*db.Group)
	if !ok {
		panic("getGroupFromContext: group not found in context")
	}
	return group
}
