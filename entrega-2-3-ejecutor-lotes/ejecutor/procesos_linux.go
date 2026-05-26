package main

import "syscall"

func suspenderPID(pid int) error {
	return syscall.Kill(pid, syscall.SIGSTOP)
}

func reanudarPID(pid int) error {
	return syscall.Kill(pid, syscall.SIGCONT)
}
