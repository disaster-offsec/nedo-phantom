package modules

import (
		"fmt"
)

type Module interface {
    Name() string
    Execute(data []byte) ([]byte, error)
}

type ModuleManager struct {
    modules map[string]Module
}

func NewModuleManager() *ModuleManager {
    return &ModuleManager{
        modules: make(map[string]Module),
    }
}

func (m *ModuleManager) Register(module Module) {
    m.modules[module.Name()] = module
}

func (m *ModuleManager) Execute(name string, data []byte) ([]byte, error) {
    module, ok := m.modules[name]
    if !ok {
        return nil, fmt.Errorf("module %s not found", name)
    }
    return module.Execute(data)
}

func (m *ModuleManager) Count() int {
    return len(m.modules)
}
