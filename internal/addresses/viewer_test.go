package addresses

import (
	"errors"
	"reflect"
	"testing"
)

type mockProvider struct {
	addrs map[string][]string
	err   error
}

func (m mockProvider) InterfaceAddresses(name string) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	if list, ok := m.addrs[name]; ok {
		result := make([]string, len(list))
		copy(result, list)
		return result, nil
	}
	return nil, errors.New("interface not found")
}

func TestViewerViewSorts(t *testing.T) {
	provider := mockProvider{addrs: map[string][]string{"eth0": {"192.168.0.2/24", "10.0.0.2/24"}}}
	v := NewViewer(provider)
	got, err := v.View("eth0")
	if err != nil {
		t.Fatalf("View() error = %v", err)
	}
	want := []string{"10.0.0.2/24", "192.168.0.2/24"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("View() = %#v, want %#v", got, want)
	}
}

func TestViewerRequiresInterface(t *testing.T) {
	v := NewViewer(mockProvider{})
	if _, err := v.View(""); err == nil {
		t.Fatal("expected error when interface is missing")
	}
}

func TestViewerProviderError(t *testing.T) {
	v := NewViewer(mockProvider{err: errors.New("boom")})
	if _, err := v.View("eth0"); err == nil {
		t.Fatal("expected error")
	}
}
