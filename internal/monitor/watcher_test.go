package monitor

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/user/goeth/internal/addresses"
	"github.com/user/goeth/internal/interfaces"
)

type stubInterfaceProvider struct {
	interfaces []interfaces.Interface
	err        error
}

func (s stubInterfaceProvider) ListInterfaces() ([]interfaces.Interface, error) {
	if s.err != nil {
		return nil, s.err
	}
	return append([]interfaces.Interface(nil), s.interfaces...), nil
}

type stubAddressProvider struct {
	addrs map[string][]string
	err   error
}

func (s stubAddressProvider) InterfaceAddresses(name string) ([]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	return append([]string(nil), s.addrs[name]...), nil
}

func TestDiffInterfacesDetectsChanges(t *testing.T) {
	prev := map[string]interfaces.Interface{
		"eth0": {Name: "eth0", HardwareAddr: "aa:bb", MTU: 1500},
		"eth1": {Name: "eth1", HardwareAddr: "cc:dd", MTU: 1500, Flags: []string{"up"}},
	}
	curr := map[string]interfaces.Interface{
		"eth0": {Name: "eth0", HardwareAddr: "aa:bb", MTU: 1400},
		"eth2": {Name: "eth2", HardwareAddr: "ee:ff", MTU: 1500},
	}

	added, removed, updated := diffInterfaces(prev, curr)
	if len(added) != 1 || added[0].Name != "eth2" {
		t.Fatalf("expected eth2 to be added, got %#v", added)
	}
	if len(removed) != 1 || removed[0].Name != "eth1" {
		t.Fatalf("expected eth1 to be removed, got %#v", removed)
	}
	if len(updated) != 1 || updated[0].Name != "eth0" {
		t.Fatalf("expected eth0 update, got %#v", updated)
	}
}

func TestDiffAddressesDetectsChanges(t *testing.T) {
	prev := map[string][]string{
		"eth0": {"192.0.2.1/24", "2001:db8::1/64"},
		"eth1": {"192.0.2.2/24"},
	}
	curr := map[string][]string{
		"eth0": {"192.0.2.1/24", "198.51.100.1/24"},
		"eth2": {"203.0.113.5/24"},
	}

	changes := diffAddresses(prev, curr)
	if len(changes) != 3 {
		t.Fatalf("expected 3 changes, got %d", len(changes))
	}
	first := changes[0]
	if first.Name != "eth0" || len(first.Added) != 1 || len(first.Removed) != 1 {
		t.Fatalf("unexpected change for eth0: %#v", first)
	}
}

func TestWatcherReportsChanges(t *testing.T) {
	writer := &bytes.Buffer{}
	watcher := Watcher{
		Lister:   interfaces.NewLister(stubInterfaceProvider{interfaces: []interfaces.Interface{{Name: "eth0", HardwareAddr: "aa:bb", MTU: 1500}}}),
		Viewer:   addresses.NewViewer(stubAddressProvider{addrs: map[string][]string{"eth0": {"192.0.2.1/24"}}}),
		Interval: time.Millisecond,
		Writer:   writer,
		Now: func() time.Time {
			return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel to avoid ticker loop
	if err := watcher.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
	out := writer.String()
	if !strings.Contains(out, "monitoring started") {
		t.Fatalf("expected initial message, got %q", out)
	}
}
