package cmdutil

import (
	"os"

	"github.com/ryan-gang/kindle-send-daemon/internal/config"
	"github.com/ryan-gang/kindle-send-daemon/internal/util"
	"github.com/spf13/cobra"
)

// LoadConfigFromFlags loads configuration using the config flag from the command
func LoadConfigFromFlags(cmd *cobra.Command) (config.ConfigProvider, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return nil, err
	}

	return config.LoadProvider(configPath)
}

// LoadConfigOrExit loads configuration and exits with error message if it fails
func LoadConfigOrExit(cmd *cobra.Command) config.ConfigProvider {
	cfg, err := LoadConfigFromFlags(cmd)
	if err != nil {
		util.LogError(util.ConfigError, "loading configuration", err)
		return nil
	}
	return cfg
}

// CheckDaemonEnabledOrExit checks if daemon is enabled and exits with message if not
func CheckDaemonEnabledOrExit(cfg config.ConfigProvider) {
	if !cfg.IsDaemonEnabled() {
		util.Red.Println("Daemon is not enabled in configuration")
		util.Cyan.Println("Run 'kindle-send configure' to enable daemon mode")
		os.Exit(1)
	}
}
