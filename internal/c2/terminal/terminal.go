package terminal

import (
    "fmt"
    "os"
    "syscall"
    "unsafe"
)

type Terminal struct {
    fd              uintptr
    originalTermios syscall.Termios
}

func NewTerminal() *Terminal {
    return &Terminal{
        fd: os.Stdin.Fd(),
    }
}

// Raw включает raw-режим
func (t *Terminal) Raw() error {
    var termios syscall.Termios
    if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, t.fd, syscall.TCGETS, uintptr(unsafe.Pointer(&termios))); errno != 0 {
        return fmt.Errorf("TCGETS: %v", errno)
    }
    t.originalTermios = termios

    termios.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.ISIG
    termios.Iflag &^= syscall.IXON | syscall.ICRNL
    termios.Oflag &^= syscall.OPOST
    termios.Cflag |= syscall.CS8

    if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, t.fd, syscall.TCSETS, uintptr(unsafe.Pointer(&termios))); errno != 0 {
        return fmt.Errorf("TCSETS: %v", errno)
    }
    return nil
}

// Restore восстанавливает терминал
func (t *Terminal) Restore() error {
    if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, t.fd, syscall.TCSETS, uintptr(unsafe.Pointer(&t.originalTermios))); errno != 0 {
        return fmt.Errorf("TCSETS: %v", errno)
    }
    return nil
}
