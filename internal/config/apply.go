package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// Configuration represents the JSON configuration schema.
type Configuration struct {
	Interface string   `json:"interface"`
	Addresses []string `json:"addresses"`
}

// Executor applies the provided configuration to the environment.
type Executor interface {
	Apply(Configuration) error
}

// Applier validates and forwards configurations to an Executor.
type Applier struct {
	executor Executor
}

// NewApplier creates an Applier.
func NewApplier(executor Executor) Applier {
	return Applier{executor: executor}
}

// Apply validates the configuration before invoking the executor.
func (a Applier) Apply(cfg Configuration) error {
	if a.executor == nil {
		return errors.New("configuration executor is not configured")
	}
	if cfg.Interface == "" {
		return errors.New("interface is required")
	}
	if len(cfg.Addresses) == 0 {
		return errors.New("at least one address is required")
	}
	return a.executor.Apply(cfg)
}

// Loader loads configuration files.
type Loader struct {
	readFile func(string) ([]byte, error)
}

// NewLoader creates a Loader backed by os.ReadFile.
func NewLoader() Loader {
	return Loader{readFile: os.ReadFile}
}

// NewLoaderWithReader creates a Loader with a custom read function.
func NewLoaderWithReader(reader func(string) ([]byte, error)) Loader {
	return Loader{readFile: reader}
}

// Load reads and parses a JSON configuration file.
func (l Loader) Load(path string) (Configuration, error) {
	if l.readFile == nil {
		return Configuration{}, errors.New("configuration reader is not configured")
	}
	if path == "" {
		return Configuration{}, errors.New("configuration path is required")
	}
	raw, err := l.readFile(path)
	if err != nil {
		return Configuration{}, err
	}
	var cfg Configuration
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Configuration{}, fmt.Errorf("parse configuration: %w", err)
	}
	return cfg, nil
}

// ConsoleExecutor writes the configuration steps to a writer.
type ConsoleExecutor struct {
	Writer io.Writer
}

// Apply prints the configuration to the writer.
func (c ConsoleExecutor) Apply(cfg Configuration) error {
	if c.Writer == nil {
		return errors.New("writer is not configured")
	}
	_, err := fmt.Fprintf(c.Writer, "Applying configuration to %s\n", cfg.Interface)
	if err != nil {
		return err
	}
	for _, addr := range cfg.Addresses {
		if _, err := fmt.Fprintf(c.Writer, " - %s\n", addr); err != nil {
			return err
		}
	}
	return nil
}
