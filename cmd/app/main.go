package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/user/goeth/internal/addresses"
	"github.com/user/goeth/internal/config"
	"github.com/user/goeth/internal/interfaces"
	"github.com/user/goeth/internal/monitor"
)

func main() {
	lister := interfaces.NewLister(interfaces.NetProvider{})
	viewer := addresses.NewViewer(addresses.NetProvider{})
	loader := config.NewLoader()
	applier := config.NewApplier(config.ConsoleExecutor{Writer: os.Stdout})

	root := newRootCommand(lister, viewer, loader, applier)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCommand(lister interfaces.Lister, viewer addresses.Viewer, loader config.Loader, applier config.Applier) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "goeth",
		Short: "Manage network interfaces and configuration",
	}
	cmd.AddCommand(newInterfacesCmd(lister))
	cmd.AddCommand(newAddressesCmd(viewer))
	cmd.AddCommand(newApplyCmd(loader, applier))
	cmd.AddCommand(newMonitorCmd(lister, viewer))
	return cmd
}

func newInterfacesCmd(lister interfaces.Lister) *cobra.Command {
	return &cobra.Command{
		Use:   "interfaces",
		Short: "List network interfaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			interfaces, err := lister.List()
			if err != nil {
				return err
			}
			if len(interfaces) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No interfaces found")
				return nil
			}
			for _, iface := range interfaces {
				fmt.Fprintf(cmd.OutOrStdout(), "%s (MTU=%d, HW=%s)\n", iface.Name, iface.MTU, iface.HardwareAddr)
			}
			return nil
		},
	}
}

func newAddressesCmd(viewer addresses.Viewer) *cobra.Command {
	var ifaceName string
	cmd := &cobra.Command{
		Use:   "addresses",
		Short: "Show addresses for an interface",
		RunE: func(cmd *cobra.Command, args []string) error {
			addrs, err := viewer.View(ifaceName)
			if err != nil {
				return err
			}
			if len(addrs) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No addresses for %s\n", ifaceName)
				return nil
			}
			for _, addr := range addrs {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", addr)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&ifaceName, "interface", "i", "", "Interface name")
	cmd.MarkFlagRequired("interface")
	return cmd
}

func newApplyCmd(loader config.Loader, applier config.Applier) *cobra.Command {
	var path string
	cmd := &cobra.Command{
		Use:   "apply-config",
		Short: "Apply configuration from a JSON file",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loader.Load(path)
			if err != nil {
				return err
			}
			if err := applier.Apply(cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Configuration applied to %s\n", cfg.Interface)
			return nil
		},
	}
	cmd.Flags().StringVarP(&path, "file", "f", "", "Path to JSON configuration file")
	cmd.MarkFlagRequired("file")
	return cmd
}

func newMonitorCmd(lister interfaces.Lister, viewer addresses.Viewer) *cobra.Command {
	var interval time.Duration
	var iface string
	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "Watch interfaces and addresses for changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			watcher := monitor.Watcher{
				Lister:    lister,
				Viewer:    viewer,
				Interval:  interval,
				Interface: iface,
				Writer:    cmd.OutOrStdout(),
			}
			if err := watcher.Run(ctx); err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				return err
			}
			return nil
		},
	}
	cmd.Flags().DurationVarP(&interval, "interval", "t", 5*time.Second, "Polling interval")
	cmd.Flags().StringVarP(&iface, "interface", "i", "", "Interface to monitor (all by default)")
	return cmd
}
