//go:build windows

package credentials

import (
	"context"
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

const (
	credTypeGeneric     = 1
	credPersistLocalMac = 2
)

var (
	advapi32   = syscall.NewLazyDLL("advapi32.dll")
	credWriteW = advapi32.NewProc("CredWriteW")
	credReadW  = advapi32.NewProc("CredReadW")
	credDelete = advapi32.NewProc("CredDeleteW")
	credFree   = advapi32.NewProc("CredFree")
)

type windowsCredential struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        syscall.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

type Native struct{}

func target(name string) string { return "CyberPilot/" + name }

func (Native) Put(_ context.Context, name, secret string) (string, error) {
	targetName, err := syscall.UTF16PtrFromString(target(name))
	if err != nil {
		return "", fmt.Errorf("invalid credential name: %w", err)
	}
	userName, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return "", fmt.Errorf("invalid credential name: %w", err)
	}
	// Credential Manager accepts an opaque byte blob. UTF-8 avoids a trailing
	// NUL and the backing slice remains live for the duration of CredWriteW.
	blob := []byte(secret)
	credential := windowsCredential{Type: credTypeGeneric, TargetName: targetName, Persist: credPersistLocalMac, UserName: userName, CredentialBlobSize: uint32(len(blob))}
	if len(blob) > 0 {
		credential.CredentialBlob = &blob[0]
	}
	result, _, callErr := credWriteW.Call(uintptr(unsafe.Pointer(&credential)), 0)
	if result == 0 {
		return "", fmt.Errorf("store Windows credential: %w", callErr)
	}
	return "wincred:" + name, nil
}

func (Native) Get(_ context.Context, ref string) (string, error) {
	name := strings.TrimPrefix(ref, "wincred:")
	if name == ref || name == "" {
		return "", fmt.Errorf("invalid Windows credential reference")
	}
	targetName, err := syscall.UTF16PtrFromString(target(name))
	if err != nil {
		return "", err
	}
	var credential *windowsCredential
	result, _, callErr := credReadW.Call(uintptr(unsafe.Pointer(targetName)), credTypeGeneric, 0, uintptr(unsafe.Pointer(&credential)))
	if result == 0 {
		return "", fmt.Errorf("read Windows credential: %w", callErr)
	}
	defer credFree.Call(uintptr(unsafe.Pointer(credential)))
	if credential.CredentialBlobSize == 0 {
		return "", nil
	}
	value := unsafe.Slice(credential.CredentialBlob, credential.CredentialBlobSize)
	return string(value), nil
}

func (Native) Delete(_ context.Context, ref string) error {
	name := strings.TrimPrefix(ref, "wincred:")
	if name == ref || name == "" {
		return fmt.Errorf("invalid Windows credential reference")
	}
	targetName, err := syscall.UTF16PtrFromString(target(name))
	if err != nil {
		return err
	}
	result, _, callErr := credDelete.Call(uintptr(unsafe.Pointer(targetName)), credTypeGeneric, 0)
	if result == 0 {
		return fmt.Errorf("delete Windows credential: %w", callErr)
	}
	return nil
}
