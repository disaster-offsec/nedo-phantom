package modules

import (
    "fmt"
    "os"
    "runtime"
)

type InfoModule struct{}

func (i *InfoModule) Name() string {
    return "info"
}

func (i *InfoModule) Execute(data []byte) ([]byte, error) {
    hostname, _ := os.Hostname()
    info := fmt.Sprintf(
        "hostname: %s\nOS: %s\nArch: %s\n",
        hostname,
        runtime.GOOS,
        runtime.GOARCH,
    )
    return []byte(info), nil
}
