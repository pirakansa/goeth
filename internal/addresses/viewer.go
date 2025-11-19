package addresses

import (
	"errors"
	"net"
	"sort"
)

// Provider retrieves addresses for a given interface.
type Provider interface {
	InterfaceAddresses(name string) ([]string, error)
}

// Viewer exposes address lookup behavior.
type Viewer struct {
	provider Provider
}

// NewViewer creates a Viewer backed by provider.
func NewViewer(provider Provider) Viewer {
	return Viewer{provider: provider}
}

// View returns the addresses for the requested interface.
func (v Viewer) View(name string) ([]string, error) {
	if v.provider == nil {
		return nil, errors.New("address provider is not configured")
	}
	if name == "" {
		return nil, errors.New("interface name is required")
	}
	addrs, err := v.provider.InterfaceAddresses(name)
	if err != nil {
		return nil, err
	}
	sort.Strings(addrs)
	return addrs, nil
}

// NetProvider fetches addresses using the net package.
type NetProvider struct{}

// InterfaceAddresses returns addresses for the interface.
func (NetProvider) InterfaceAddresses(name string) ([]string, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, err
	}
	list, err := iface.Addrs()
	if err != nil {
		return nil, err
	}
	addrs := make([]string, 0, len(list))
	for _, addr := range list {
		addrs = append(addrs, addr.String())
	}
	return addrs, nil
}
