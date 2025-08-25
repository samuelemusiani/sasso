package proxmox

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const base62Alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// EncodeBase62 encodes a uint32 into a base62 string
func EncodeBase62(num uint32) string {
	if num == 0 {
		return string(base62Alphabet[0])
	}
	var sb strings.Builder
	for num > 0 {
		remainder := num % 62
		sb.WriteByte(base62Alphabet[remainder])
		num /= 62
	}
	// reverse since we construct in reverse order
	runes := []rune(sb.String())
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// DecodeBase62 decodes a base62 string into a uint32
func DecodeBase62(s string) (uint32, error) {
	var num uint32
	for _, c := range s {
		index := strings.IndexRune(base62Alphabet, c)
		if index == -1 {
			return 0, fmt.Errorf("invalid character: %c", c)
		}
		num = num*62 + uint32(index)
	}
	return num, nil
}

type Storage struct {
	Name    string
	VMID    uint32
	File    string
	Discard bool
	Size    uint
}

var (
	ErrInvalidStorageString = errors.New("invalid storage string")
)

// Parses a string like "storage0:1011/vm-1011-disk-1.qcow2,discard=on,size=4G"
func parseStorageFromString(s string) (*Storage, error) {
	var st Storage

	// Split name/path and options
	parts := strings.SplitN(s, ",", 2)
	if len(parts) < 1 {
		return nil, ErrInvalidStorageString
	}

	// "storage0:1011/vm-1011-disk-1.qcow2"
	np := parts[0]
	npParts := strings.SplitN(np, ":", 2)
	if len(npParts) != 2 {
		err := errors.Join(ErrInvalidStorageString, errors.New("Missing ':'"))
		return nil, err
	}
	st.Name = npParts[0]

	// "1011/vm-1011-disk-1.qcow2"
	vmFileParts := strings.SplitN(npParts[1], "/", 2)
	if len(vmFileParts) != 2 {
		err := errors.Join(ErrInvalidStorageString, errors.New("invalid VM/file format"))
		return nil, err
	}
	vmid, err := strconv.ParseUint(vmFileParts[0], 10, 32)
	if err != nil {
		err := errors.Join(ErrInvalidStorageString, errors.New("invalid VMID"))
		return nil, err
	}
	st.VMID = uint32(vmid)
	st.File = vmFileParts[1]

	if len(parts) < 2 {
		return &st, nil
	}

	options := strings.Split(parts[1], ",")
	for _, opt := range options {
		kv := strings.SplitN(opt, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "discard":
			st.Discard = (kv[1] == "on")
		case "size":
			sizeStr := strings.TrimSuffix(kv[1], "G")
			val, err := strconv.ParseUint(sizeStr, 10, 32)
			if err != nil {
				err := errors.Join(ErrInvalidStorageString, fmt.Errorf("invalid size: %v", err))
				return nil, err
			}
			st.Size = uint(val)
		}
	}

	return &st, nil
}
