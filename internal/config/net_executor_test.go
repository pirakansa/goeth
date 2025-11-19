package config

import (
	"errors"
	"testing"

	"github.com/vishvananda/netlink"
)

type fakeLink struct{ netlink.LinkAttrs }

func (f *fakeLink) Attrs() *netlink.LinkAttrs { return &f.LinkAttrs }
func (f *fakeLink) Type() string              { return "fake" }

type mockNetlinkProvider struct {
	link    netlink.Link
	linkErr error

	lists   map[int][]netlink.Addr
	listErr map[int]error

	addErr error
	delErr error

	added   []string
	removed []string
}

func (m *mockNetlinkProvider) LinkByName(name string) (netlink.Link, error) {
	if m.linkErr != nil {
		return nil, m.linkErr
	}
	if m.link == nil {
		return &fakeLink{}, nil
	}
	return m.link, nil
}

func (m *mockNetlinkProvider) AddrList(link netlink.Link, family int) ([]netlink.Addr, error) {
	if m.listErr != nil {
		if err, ok := m.listErr[family]; ok && err != nil {
			return nil, err
		}
	}
	if m.lists == nil {
		return nil, nil
	}
	return m.lists[family], nil
}

func (m *mockNetlinkProvider) AddrAdd(link netlink.Link, addr *netlink.Addr) error {
	if m.addErr != nil {
		return m.addErr
	}
	m.added = append(m.added, addr.String())
	return nil
}

func (m *mockNetlinkProvider) AddrDel(link netlink.Link, addr *netlink.Addr) error {
	if m.delErr != nil {
		return m.delErr
	}
	m.removed = append(m.removed, addr.String())
	return nil
}

func TestNetlinkExecutorApplyAddsAndRemoves(t *testing.T) {
	provider := &mockNetlinkProvider{
		lists: map[int][]netlink.Addr{
			netlink.FAMILY_V4: {
				mustAddr(t, "192.0.2.5/24"),
				mustAddr(t, "192.0.2.10/24"),
			},
			netlink.FAMILY_V6: {
				mustAddr(t, "2001:db8::5/64"),
			},
		},
	}
	exec := NetlinkExecutor{Provider: provider}
	cfg := Configuration{Interface: "eth0", Addresses: []string{"192.0.2.10/24", "192.0.2.20/24", "2001:db8::10/64"}}
	if err := exec.Apply(cfg); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if len(provider.added) != 2 || !contains(provider.added, "192.0.2.20/24") ||
		!contains(provider.added, "2001:db8::10/64") {
		t.Fatalf("unexpected added addresses: %v", provider.added)
	}
	if len(provider.removed) != 2 || !contains(provider.removed, "192.0.2.5/24") ||
		!contains(provider.removed, "2001:db8::5/64") {
		t.Fatalf("unexpected removed addresses: %v", provider.removed)
	}
}

func TestNetlinkExecutorApplyPropagatesLinkError(t *testing.T) {
	provider := &mockNetlinkProvider{linkErr: errors.New("boom")}
	exec := NetlinkExecutor{Provider: provider}
	cfg := Configuration{Interface: "eth0", Addresses: []string{"192.0.2.1/24"}}
	if err := exec.Apply(cfg); err == nil {
		t.Fatal("expected error when link lookup fails")
	}
}

func TestNetlinkExecutorApplyValidatesAddresses(t *testing.T) {
	exec := NetlinkExecutor{Provider: &mockNetlinkProvider{}}
	cfg := Configuration{Interface: "eth0", Addresses: []string{"not-an-ip"}}
	if err := exec.Apply(cfg); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestNetlinkExecutorAddError(t *testing.T) {
	provider := &mockNetlinkProvider{addErr: errors.New("add-failed")}
	exec := NetlinkExecutor{Provider: provider}
	cfg := Configuration{Interface: "eth0", Addresses: []string{"192.0.2.1/24"}}
	if err := exec.Apply(cfg); err == nil {
		t.Fatal("expected add error")
	}
}

func TestNetlinkExecutorRemoveError(t *testing.T) {
	provider := &mockNetlinkProvider{
		lists: map[int][]netlink.Addr{
			netlink.FAMILY_V4: {mustAddr(t, "192.0.2.5/24")},
		},
		delErr: errors.New("del-failed"),
	}
	exec := NetlinkExecutor{Provider: provider}
	cfg := Configuration{Interface: "eth0", Addresses: []string{"192.0.2.10/24"}}
	if err := exec.Apply(cfg); err == nil {
		t.Fatal("expected delete error")
	}
}

func TestNetlinkExecutorListError(t *testing.T) {
	provider := &mockNetlinkProvider{
		listErr: map[int]error{netlink.FAMILY_V4: errors.New("boom")},
	}
	exec := NetlinkExecutor{Provider: provider}
	cfg := Configuration{Interface: "eth0", Addresses: []string{"192.0.2.10/24"}}
	if err := exec.Apply(cfg); err == nil {
		t.Fatal("expected list error")
	}
}

func mustAddr(t *testing.T, cidr string) netlink.Addr {
	t.Helper()
	addr, err := netlink.ParseAddr(cidr)
	if err != nil {
		t.Fatalf("ParseAddr(%s) error = %v", cidr, err)
	}
	return *addr
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
