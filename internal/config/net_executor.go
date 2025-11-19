package config

import (
	"errors"
	"fmt"

	"github.com/vishvananda/netlink"
)

// NetlinkProvider exposes the subset of netlink APIs needed by the executor.
type NetlinkProvider interface {
	LinkByName(name string) (netlink.Link, error)
	AddrList(link netlink.Link, family int) ([]netlink.Addr, error)
	AddrAdd(link netlink.Link, addr *netlink.Addr) error
	AddrDel(link netlink.Link, addr *netlink.Addr) error
}

// NetlinkExecutor applies configurations using a NetlinkProvider.
type NetlinkExecutor struct {
	Provider NetlinkProvider
}

// NewNetlinkExecutor creates an executor backed by provider.
func NewNetlinkExecutor(provider NetlinkProvider) NetlinkExecutor {
	return NetlinkExecutor{Provider: provider}
}

// Apply ensures the provided configuration is reflected on the interface.
func (n NetlinkExecutor) Apply(cfg Configuration) error {
	if n.Provider == nil {
		return errors.New("netlink provider is not configured")
	}
	link, err := n.Provider.LinkByName(cfg.Interface)
	if err != nil {
		return fmt.Errorf("lookup interface %q: %w", cfg.Interface, err)
	}
	desired, families, err := parseDesiredAddresses(cfg.Addresses)
	if err != nil {
		return err
	}
	current, err := n.collectCurrent(link, families)
	if err != nil {
		return err
	}
	for key, addr := range desired {
		if _, ok := current[key]; ok {
			continue
		}
		if err := n.Provider.AddrAdd(link, addr); err != nil {
			return fmt.Errorf("add address %s: %w", key, err)
		}
	}
	for key, addr := range current {
		if _, ok := desired[key]; ok {
			continue
		}
		if err := n.Provider.AddrDel(link, addr); err != nil {
			return fmt.Errorf("remove address %s: %w", key, err)
		}
	}
	return nil
}

func (n NetlinkExecutor) collectCurrent(link netlink.Link, families []int) (map[string]*netlink.Addr, error) {
	current := make(map[string]*netlink.Addr)
	for _, family := range families {
		addrs, err := n.Provider.AddrList(link, family)
		if err != nil {
			return nil, fmt.Errorf("list addresses for family %d: %w", family, err)
		}
		for i := range addrs {
			addrCopy := addrs[i]
			current[addrCopy.IPNet.String()] = &addrCopy
		}
	}
	return current, nil
}

func parseDesiredAddresses(raw []string) (map[string]*netlink.Addr, []int, error) {
	desired := make(map[string]*netlink.Addr, len(raw))
	familySet := make(map[int]struct{})
	var families []int
	for _, addrStr := range raw {
		addr, err := netlink.ParseAddr(addrStr)
		if err != nil {
			return nil, nil, fmt.Errorf("parse address %q: %w", addrStr, err)
		}
		desired[addr.String()] = addr
		fam := addrFamily(addr)
		if _, ok := familySet[fam]; !ok {
			familySet[fam] = struct{}{}
			families = append(families, fam)
		}
	}
	return desired, families, nil
}

func addrFamily(addr *netlink.Addr) int {
	if addr.IP.To4() != nil {
		return netlink.FAMILY_V4
	}
	return netlink.FAMILY_V6
}

// NetlinkAPI uses github.com/vishvananda/netlink to make changes.
type NetlinkAPI struct{}

// LinkByName retrieves a link by name.
func (NetlinkAPI) LinkByName(name string) (netlink.Link, error) {
	return netlink.LinkByName(name)
}

// AddrList returns the addresses for the link/family.
func (NetlinkAPI) AddrList(link netlink.Link, family int) ([]netlink.Addr, error) {
	return netlink.AddrList(link, family)
}

// AddrAdd adds an address to the link.
func (NetlinkAPI) AddrAdd(link netlink.Link, addr *netlink.Addr) error {
	return netlink.AddrAdd(link, addr)
}

// AddrDel removes an address from the link.
func (NetlinkAPI) AddrDel(link netlink.Link, addr *netlink.Addr) error {
	return netlink.AddrDel(link, addr)
}
