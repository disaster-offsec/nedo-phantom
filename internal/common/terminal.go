package common

// TerminalSize структура для размера окна
type TerminalSize struct {
    Rows uint16
    Cols uint16
}

// GetTerminalSize возвращает размер текущего терминала
func GetTerminalSize(fd uintptr) (*TerminalSize, error) {
    // TODO: syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCGWINSZ, ...)
    return &TerminalSize{Rows: 24, Cols: 80}, nil
}

// SetTerminalSize устанавливает размер терминала
func SetTerminalSize(fd uintptr, rows, cols uint16) error {
    // TODO: syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCSWINSZ, ...)
    return nil
}
