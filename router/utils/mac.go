package utils

import (
	"github.com/seancfoley/ipaddress-go/ipaddr"
)

func AreMACsEqual(a, b string) bool {
	am := ipaddr.NewMACAddressString(a)
	bm := ipaddr.NewMACAddressString(b)

	return am.Compare(bm) == 0
}
