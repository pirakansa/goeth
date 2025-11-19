package monitor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/user/goeth/internal/addresses"
	"github.com/user/goeth/internal/interfaces"
)

// Watcher polls the operating system for interface information and reports changes.
type Watcher struct {
	// Lister provides the current interface list.
	Lister interfaces.Lister
	// Viewer provides the addresses for a given interface.
	Viewer addresses.Viewer
	// Interval controls how frequently the state is refreshed.
	Interval time.Duration
	// Interface restricts monitoring to a single interface. When empty all interfaces are monitored.
	Interface string
	// Writer receives human-readable change notifications.
	Writer io.Writer
	// Now overrides the time source (used in tests).
	Now func() time.Time
}

type snapshot struct {
	interfaces map[string]interfaces.Interface
	addresses  map[string][]string
}

// Run starts the monitoring loop until the context is cancelled or an error occurs.
func (w Watcher) Run(ctx context.Context) error {
	if w.Writer == nil {
		return errors.New("writer is required")
	}
	if w.Interval <= 0 {
		return errors.New("interval must be positive")
	}

	current, err := w.collect()
	if err != nil {
		return err
	}
	w.printInitial(current)

	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			next, err := w.collect()
			if err != nil {
				return err
			}
			w.reportChanges(current, next)
			current = next
		}
	}
}

func (w Watcher) collect() (snapshot, error) {
	list, err := w.Lister.List()
	if err != nil {
		return snapshot{}, err
	}
	snap := snapshot{
		interfaces: make(map[string]interfaces.Interface),
		addresses:  make(map[string][]string),
	}
	for _, iface := range list {
		if w.Interface != "" && iface.Name != w.Interface {
			continue
		}
		snap.interfaces[iface.Name] = iface
		addrs, err := w.Viewer.View(iface.Name)
		if err != nil {
			return snapshot{}, err
		}
		sort.Strings(addrs)
		snap.addresses[iface.Name] = addrs
	}
	if w.Interface != "" {
		if _, ok := snap.interfaces[w.Interface]; !ok {
			snap.addresses[w.Interface] = nil
		}
	}
	return snap, nil
}

func (w Watcher) printInitial(snap snapshot) {
	fmt.Fprintf(w.Writer, "[%s] monitoring started (interval %s)\n", w.timestamp(), w.Interval)
	if w.Interface != "" {
		fmt.Fprintf(w.Writer, " - filter: %s\n", w.Interface)
	}
	if len(snap.interfaces) == 0 {
		if w.Interface == "" {
			fmt.Fprintln(w.Writer, "No interfaces detected yet")
		} else {
			fmt.Fprintf(w.Writer, "Waiting for %s to appear...\n", w.Interface)
		}
		return
	}
	names := make([]string, 0, len(snap.interfaces))
	for name := range snap.interfaces {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		iface := snap.interfaces[name]
		fmt.Fprintf(w.Writer, " - %s (MTU=%d, HW=%s)\n", iface.Name, iface.MTU, iface.HardwareAddr)
		addrs := snap.addresses[name]
		if len(addrs) == 0 {
			fmt.Fprintf(w.Writer, "   addresses: none\n")
			continue
		}
		fmt.Fprintf(w.Writer, "   addresses: %s\n", strings.Join(addrs, ", "))
	}
}

func (w Watcher) reportChanges(prev, curr snapshot) {
	added, removed, updated := diffInterfaces(prev.interfaces, curr.interfaces)
	for _, iface := range added {
		fmt.Fprintf(w.Writer, "[%s] interface %s added (MTU=%d, HW=%s)\n", w.timestamp(), iface.Name, iface.MTU, iface.HardwareAddr)
	}
	for _, iface := range removed {
		fmt.Fprintf(w.Writer, "[%s] interface %s removed\n", w.timestamp(), iface.Name)
	}
	for _, change := range updated {
		diffs := describeInterfaceChange(change.Before, change.After)
		fmt.Fprintf(w.Writer, "[%s] interface %s updated: %s\n", w.timestamp(), change.Name, strings.Join(diffs, ", "))
	}
	for _, change := range diffAddresses(prev.addresses, curr.addresses) {
		if len(change.Added) > 0 {
			fmt.Fprintf(w.Writer, "[%s] %s addresses added: %s\n", w.timestamp(), change.Name, strings.Join(change.Added, ", "))
		}
		if len(change.Removed) > 0 {
			fmt.Fprintf(w.Writer, "[%s] %s addresses removed: %s\n", w.timestamp(), change.Name, strings.Join(change.Removed, ", "))
		}
	}
}

func (w Watcher) timestamp() string {
	now := w.Now
	if now == nil {
		now = time.Now
	}
	return now().Format(time.RFC3339)
}

type interfaceChange struct {
	Name   string
	Before interfaces.Interface
	After  interfaces.Interface
}

func diffInterfaces(prev, curr map[string]interfaces.Interface) (added, removed []interfaces.Interface, updated []interfaceChange) {
	for name, iface := range curr {
		p, ok := prev[name]
		if !ok {
			added = append(added, iface)
			continue
		}
		if !sameInterface(p, iface) {
			updated = append(updated, interfaceChange{Name: name, Before: p, After: iface})
		}
	}
	for name, iface := range prev {
		if _, ok := curr[name]; !ok {
			removed = append(removed, iface)
		}
	}
	sort.Slice(added, func(i, j int) bool { return added[i].Name < added[j].Name })
	sort.Slice(removed, func(i, j int) bool { return removed[i].Name < removed[j].Name })
	sort.Slice(updated, func(i, j int) bool { return updated[i].Name < updated[j].Name })
	return added, removed, updated
}

func sameInterface(a, b interfaces.Interface) bool {
	if a.Name != b.Name || a.HardwareAddr != b.HardwareAddr || a.MTU != b.MTU {
		return false
	}
	if len(a.Flags) != len(b.Flags) {
		return false
	}
	for i := range a.Flags {
		if a.Flags[i] != b.Flags[i] {
			return false
		}
	}
	return true
}

func describeInterfaceChange(before, after interfaces.Interface) []string {
	var changes []string
	if before.MTU != after.MTU {
		changes = append(changes, fmt.Sprintf("MTU %d→%d", before.MTU, after.MTU))
	}
	if before.HardwareAddr != after.HardwareAddr {
		changes = append(changes, fmt.Sprintf("HW %s→%s", before.HardwareAddr, after.HardwareAddr))
	}
	if !equalStrings(before.Flags, after.Flags) {
		changes = append(changes, fmt.Sprintf("flags [%s]→[%s]", strings.Join(before.Flags, ","), strings.Join(after.Flags, ",")))
	}
	if len(changes) == 0 {
		changes = append(changes, "no visible field differences")
	}
	return changes
}

type addressChange struct {
	Name    string
	Added   []string
	Removed []string
}

func diffAddresses(prev, curr map[string][]string) []addressChange {
	seen := make(map[string]struct{})
	for name := range prev {
		seen[name] = struct{}{}
	}
	for name := range curr {
		seen[name] = struct{}{}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)

	var changes []addressChange
	for _, name := range names {
		added, removed := diffStringSets(prev[name], curr[name])
		if len(added) == 0 && len(removed) == 0 {
			continue
		}
		changes = append(changes, addressChange{Name: name, Added: added, Removed: removed})
	}
	return changes
}

func diffStringSets(old, new []string) (added, removed []string) {
	oldSet := make(map[string]struct{}, len(old))
	for _, val := range old {
		oldSet[val] = struct{}{}
	}
	newSet := make(map[string]struct{}, len(new))
	for _, val := range new {
		newSet[val] = struct{}{}
	}
	for val := range newSet {
		if _, ok := oldSet[val]; !ok {
			added = append(added, val)
		}
	}
	for val := range oldSet {
		if _, ok := newSet[val]; !ok {
			removed = append(removed, val)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
