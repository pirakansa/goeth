package interfaces

import (
	"errors"
	"reflect"
	"testing"
)

type mockProvider struct {
	interfaces []Interface
	err        error
}

func (m mockProvider) ListInterfaces() ([]Interface, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.interfaces, nil
}

func TestListerListSortsByName(t *testing.T) {
	l := NewLister(mockProvider{interfaces: []Interface{{Name: "eth1"}, {Name: "eth0"}}})
	got, err := l.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	want := []Interface{{Name: "eth0"}, {Name: "eth1"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %#v, want %#v", got, want)
	}
}

func TestListerListProviderError(t *testing.T) {
	l := NewLister(mockProvider{err: errors.New("boom")})
	if _, err := l.List(); err == nil {
		t.Fatal("expected error")
	}
}

func TestListerListNoProvider(t *testing.T) {
	var l Lister
	if _, err := l.List(); err == nil {
		t.Fatal("expected error when provider is missing")
	}
}
