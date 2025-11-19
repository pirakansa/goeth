package interfaces

import (
	"errors"
	"net"
	"sort"
	"strings"
)

// Interface represents the properties of a network interface.
type Interface struct {
	Name         string
	HardwareAddr string
	MTU          int
	Flags        []string
}

// Provider retrieves interface information from the environment.
type Provider interface {
	ListInterfaces() ([]Interface, error)
}

// Lister is responsible for listing interfaces using a Provider.
type Lister struct {
	provider Provider
}

// NewLister creates a Lister with the given Provider.
func NewLister(provider Provider) Lister {
	return Lister{provider: provider}
}

// List returns all network interfaces.
func (l Lister) List() ([]Interface, error) {
	if l.provider == nil {
		return nil, errors.New("interfaces provider is not configured")
	}
	interfaces, err := l.provider.ListInterfaces()
	if err != nil {
		return nil, err
	}
	sort.SliceStable(interfaces, func(i, j int) bool {
		return interfaces[i].Name < interfaces[j].Name
	})
	return interfaces, nil
}

// NetProvider retrieves interface details using the net package.
type NetProvider struct{}

// ListInterfaces fetches interfaces from the operating system.
func (NetProvider) ListInterfaces() ([]Interface, error) {
	list, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	results := make([]Interface, 0, len(list))
	for _, iface := range list {
		flags := strings.Split(iface.Flags.String(), "|")
		if len(flags) == 1 && flags[0] == "" {
			flags = nil
		}
		results = append(results, Interface{
			Name:         iface.Name,
			HardwareAddr: iface.HardwareAddr.String(),
			MTU:          iface.MTU,
			Flags:        flags,
		})
	}
	return results, nil
}
