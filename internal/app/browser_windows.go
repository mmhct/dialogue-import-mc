package app

import (
	"fmt"
	"syscall"
	"unsafe"
)

func openWindowsBrowser(url string) error {
	op, _ := syscall.UTF16PtrFromString("open")
	target, _ := syscall.UTF16PtrFromString(url)
	p := syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")
	r, _, _ := p.Call(0, uintptr(unsafe.Pointer(op)), uintptr(unsafe.Pointer(target)), 0, 0, 1)
	if r <= 32 {
		return fmt.Errorf("ShellExecuteW: %d", r)
	}
	return nil
}

// The release uses the Windows GUI subsystem, so startup errors need a window.
func NotifyError(message string) {
	title, _ := syscall.UTF16PtrFromString("对白工坊 · DialogueForge")
	body, _ := syscall.UTF16PtrFromString(message)
	p := syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW")
	p.Call(0, uintptr(unsafe.Pointer(body)), uintptr(unsafe.Pointer(title)), 0x10)
}
