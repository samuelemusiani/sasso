package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/go-ldap/ldap/v3"
)

var ldapURL = "ldap://localhost:3890"

func main() {
	l, err := ldap.DialURL(ldapURL)
	if err != nil {
		slog.With("url", ldapURL, "error", err).Error("Failed to connect to LDAP server")
		os.Exit(1)
	}
	defer l.Close()

	err = l.Bind("cn=admin,ou=people,dc=sasso,dc=com", "adminadmin")
	if err != nil {
		slog.With("error", err).Error("Failed to bind to LDAP server")
		os.Exit(1)
	}

	searchRequest := ldap.NewSearchRequest(
		"ou=people,dc=sasso,dc=com",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectClass=person)(uid=%s))", "mattia"),
		nil,
		// []string{"dn", "mail"},
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		slog.With("error", err).Error("Failed to search for user in LDAP")
		os.Exit(1)
	}
	// for _, attributes := range sr.Entries[0].Attributes {
	// 	slog.Info("Attribute", "name", attributes.Name, "values", attributes.Values)
	// }

	// fmt.Printf("Entries found:%v", *sr.Entries[0].Attributes)

	userDN := sr.Entries[0].DN
	err = l.Bind(userDN, "12345678")
	if err != nil {
		slog.With("error", err).Error("Failed to bind to LDAP server with user DN")
		os.Exit(1)
	}

	err = l.Bind("cn=admin,ou=people,dc=sasso,dc=com", "adminadmin")
	if err != nil {
		slog.With("error", err).Error("Failed to bind to LDAP server")
		os.Exit(1)
	}

	searchRequest2 := ldap.NewSearchRequest(
		"ou=groups,dc=sasso,dc=com",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectClass=groupOfNames)(cn=%s)(member=%s))", "pippone", userDN),
		// fmt.Sprintf("(&(objectClass=person)(uid=%s))", "mattia"),
		nil,
		// []string{"dn", "mail"},
		nil,
	)

	src, err := l.Search(searchRequest2)
	if err != nil {
		slog.With("error", err).Error("Failed to search for user in LDAP")
		os.Exit(1)
	}
	for _, entry := range src.Entries {
		slog.Info("DN", "dn", entry.DN)
		for _, attributes := range entry.Attributes {
			slog.Info("Attribute", "name", attributes.Name, "values", attributes.Values)
		}
	}

	if len(src.Entries) == 1 {
		slog.Info("User authenticated successfully", "dn", userDN)
	} else {
		slog.Error("User not found or multiple entries returned")
		os.Exit(1)
	}

}
