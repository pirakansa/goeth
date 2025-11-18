package config

import (
	"errors"
	"strings"
	"testing"
)

type mockExecutor struct {
	cfg Configuration
	err error
}

func (m *mockExecutor) Apply(cfg Configuration) error {
	m.cfg = cfg
	return m.err
}

func TestApplierApplyValidates(t *testing.T) {
	applier := NewApplier(&mockExecutor{})
	err := applier.Apply(Configuration{Interface: "", Addresses: []string{"10.0.0.1/24"}})
	if err == nil {
		t.Fatal("expected validation error when interface is missing")
	}

	err = applier.Apply(Configuration{Interface: "eth0"})
	if err == nil {
		t.Fatal("expected validation error when addresses are missing")
	}
}

func TestApplierDelegatesToExecutor(t *testing.T) {
    exec := &mockExecutor{}
    applier := NewApplier(exec)
    cfg := Configuration{Interface: "eth0", Addresses: []string{"10.0.0.2/24"}}
    if err := applier.Apply(cfg); err != nil {
        t.Fatalf("Apply() error = %v", err)
    }
    if exec.cfg.Interface != cfg.Interface {
        t.Fatalf("expected interface %s, got %s", cfg.Interface, exec.cfg.Interface)
    }
    if len(exec.cfg.Addresses) != len(cfg.Addresses) || exec.cfg.Addresses[0] != cfg.Addresses[0] {
        t.Fatalf("expected addresses %v, got %v", cfg.Addresses, exec.cfg.Addresses)
    }
}

type readFunc func(string) ([]byte, error)

func (r readFunc) Read(path string) ([]byte, error) {
	return r(path)
}

func TestLoaderLoad(t *testing.T) {
	loader := NewLoaderWithReader(func(string) ([]byte, error) {
		return []byte(`{"interface":"eth0","addresses":["10.0.0.3/24"]}`), nil
	})
	cfg, err := loader.Load("config.json")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Interface != "eth0" || len(cfg.Addresses) != 1 {
		t.Fatalf("unexpected configuration: %#v", cfg)
	}
}

func TestLoaderRequiresPath(t *testing.T) {
	loader := NewLoaderWithReader(func(string) ([]byte, error) {
		return nil, nil
	})
	if _, err := loader.Load(""); err == nil {
		t.Fatal("expected error when path is missing")
	}
}

func TestConsoleExecutorWrites(t *testing.T) {
	var buf strings.Builder
	exec := ConsoleExecutor{Writer: &buf}
	cfg := Configuration{Interface: "eth0", Addresses: []string{"10.0.0.4/24", "192.168.1.2/24"}}
	if err := exec.Apply(cfg); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Applying configuration to eth0") {
		t.Fatalf("unexpected output: %s", output)
	}
	if !strings.Contains(output, "10.0.0.4/24") || !strings.Contains(output, "192.168.1.2/24") {
		t.Fatalf("missing addresses in output: %s", output)
	}
}

func TestConsoleExecutorRequiresWriter(t *testing.T) {
	exec := ConsoleExecutor{}
	if err := exec.Apply(Configuration{}); err == nil {
		t.Fatal("expected error when writer is missing")
	}
}

func TestApplierRequiresExecutor(t *testing.T) {
	var applier Applier
	if err := applier.Apply(Configuration{Interface: "eth0", Addresses: []string{"10.0.0.1/24"}}); err == nil {
		t.Fatal("expected error when executor is missing")
	}
}

func TestLoaderReaderError(t *testing.T) {
	loader := NewLoaderWithReader(func(string) ([]byte, error) {
		return nil, errors.New("boom")
	})
	if _, err := loader.Load("config.json"); err == nil {
		t.Fatal("expected read error")
	}
}

func TestLoaderInvalidJSON(t *testing.T) {
	loader := NewLoaderWithReader(func(string) ([]byte, error) {
		return []byte("not-json"), nil
	})
	if _, err := loader.Load("config.json"); err == nil {
		t.Fatal("expected parse error")
	}
}
