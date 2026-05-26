package main

import (
	"fmt"
	"syscall"
)

const PROCESS_SUSPEND_RESUME = 0x0800

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	ntdll    = syscall.NewLazyDLL("ntdll.dll")

	procOpenProcess      = kernel32.NewProc("OpenProcess")
	procCloseHandle      = kernel32.NewProc("CloseHandle")
	procNtSuspendProcess = ntdll.NewProc("NtSuspendProcess")
	procNtResumeProcess  = ntdll.NewProc("NtResumeProcess")
)

func abrirProceso(pid int) (syscall.Handle, error) {
	handle, _, err := procOpenProcess.Call(
		uintptr(PROCESS_SUSPEND_RESUME),
		uintptr(0),
		uintptr(uint32(pid)),
	)

	if handle == 0 {
		return 0, err
	}

	return syscall.Handle(handle), nil
}

func suspenderPID(pid int) error {
	handle, err := abrirProceso(pid)
	if err != nil {
		return err
	}

	defer procCloseHandle.Call(uintptr(handle))

	resultado, _, _ := procNtSuspendProcess.Call(uintptr(handle))

	if resultado != 0 {
		return fmt.Errorf("no se pudo suspender el proceso")
	}

	return nil
}

func reanudarPID(pid int) error {
	handle, err := abrirProceso(pid)
	if err != nil {
		return err
	}

	defer procCloseHandle.Call(uintptr(handle))

	resultado, _, _ := procNtResumeProcess.Call(uintptr(handle))

	if resultado != 0 {
		return fmt.Errorf("no se pudo reanudar el proceso")
	}

	return nil
}
