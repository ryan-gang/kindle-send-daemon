package cmd

import (
	"os"

	"github.com/ryan-gang/kindle-send-daemon/internal/cmdutil"
	"github.com/ryan-gang/kindle-send-daemon/internal/daemon"
	"github.com/ryan-gang/kindle-send-daemon/internal/util"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(daemonCmd)

	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonRestartCmd)
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Daemon management commands",
	Long:  `Manage the kindle-send background daemon that monitors bookmark files and automatically sends content to your ereader.`,
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the kindle-send daemon",
	Long:  `Start the background daemon that will monitor the configured bookmark path and automatically send new bookmarks to your ereader every configured interval.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := cmdutil.LoadConfigOrExit(cmd)
		if cfg == nil {
			os.Exit(1)
		}

		cmdutil.CheckDaemonEnabledOrExit(cfg)

		d, err := daemon.NewDaemon(cfg)
		if err != nil {
			util.LogError(util.DaemonError, "creating daemon", err)
			os.Exit(1)
		}
		if err := d.Start(); err != nil {
			util.LogError(util.DaemonError, "starting daemon", err)
			os.Exit(1)
		}
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the kindle-send daemon",
	Long:  `Stop the running background daemon.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := cmdutil.LoadConfigOrExit(cmd)
		if cfg == nil {
			os.Exit(1)
		}

		d, err := daemon.NewDaemon(cfg)
		if err != nil {
			util.LogError(util.DaemonError, "creating daemon", err)
			os.Exit(1)
		}

		// Check if daemon is running first
		if err := d.Status(); err != nil {
			util.Red.Println("Daemon is not running")
			return
		}

		d.Stop()
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check daemon status",
	Long:  `Check if the kindle-send daemon is currently running and display its configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := cmdutil.LoadConfigOrExit(cmd)
		if cfg == nil {
			os.Exit(1)
		}

		d, err := daemon.NewDaemon(cfg)
		if err != nil {
			util.LogError(util.DaemonError, "creating daemon", err)
			os.Exit(1)
		}
		if err := d.Status(); err != nil {
			os.Exit(1)
		}
	},
}

var daemonRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the kindle-send daemon",
	Long:  `Stop and then start the kindle-send daemon.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := cmdutil.LoadConfigOrExit(cmd)
		if cfg == nil {
			os.Exit(1)
		}

		d, err := daemon.NewDaemon(cfg)
		if err != nil {
			util.LogError(util.DaemonError, "creating daemon", err)
			os.Exit(1)
		}

		// Stop if running
		if d.Status() == nil {
			util.Cyan.Println("Stopping existing daemon...")
			d.Stop()
		}

		// Start daemon
		util.Cyan.Println("Starting daemon...")
		if err := d.Start(); err != nil {
			util.LogError(util.DaemonError, "starting daemon", err)
			os.Exit(1)
		}
	},
}
