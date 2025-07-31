package cmd

import (
	"os"
	"strconv"

	"github.com/ryan-gang/kindle-send-daemon/internal/config"
	"github.com/ryan-gang/kindle-send-daemon/internal/util"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(configureCmd)
}

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure kindle-send settings",
	Long: `Configure kindle-send daemon settings including email configuration,
bookmark monitoring path, and check intervals.`,
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")

		if _, err := os.Stat(configPath); err != nil {
			util.CyanBold.Println("Creating new configuration...")
			cfg := config.CreateConfig()
			if err := config.Save(*cfg, configPath); err != nil {
				util.Red.Printf("Error saving configuration: %v\n", err)
				os.Exit(1)
			}
			util.Green.Printf("Configuration saved to %s\n", configPath)
		} else {
			util.CyanBold.Println("Updating existing configuration...")
			cfg, err := config.Load(configPath)
			if err != nil {
				util.Red.Printf("Error loading configuration: %v\n", err)
				os.Exit(1)
			}

			util.Cyan.Println("\nCurrent daemon settings:")
			util.Cyan.Printf("Daemon enabled: %t\n", cfg.DaemonEnabled)
			util.Cyan.Printf("Bookmark path: %s\n", cfg.BookmarkPath)
			util.Cyan.Printf("Check interval: %d minutes\n", cfg.CheckInterval)

			util.CyanBold.Println("\nUpdate daemon configuration? (y/n):")
			response := util.ScanlineTrim()

			if response == "y" || response == "Y" || response == "yes" {
				util.Cyan.Printf("Path to bookmark file/folder to monitor (current: %s, empty to disable): ", cfg.BookmarkPath)
				newPath := util.ScanlineTrim()

				if newPath == "" {
					cfg.DaemonEnabled = false
					cfg.BookmarkPath = ""
				} else {
					cfg.DaemonEnabled = true
					cfg.BookmarkPath = newPath

					util.Cyan.Printf("Check interval in minutes (current: %d): ", cfg.CheckInterval)
					intervalStr := util.ScanlineTrim()
					if intervalStr != "" {
						if interval, err := strconv.Atoi(intervalStr); err == nil && interval > 0 {
							cfg.CheckInterval = interval
						}
					}
				}

				// Save updated config
				if err := config.Save(cfg, configPath); err != nil {
					util.Red.Printf("Error saving configuration: %v\n", err)
					os.Exit(1)
				}

				util.Green.Println("Configuration updated successfully!")
			}
		}

		util.CyanBold.Println("\nNext steps:")
		if config.GetInstance().DaemonEnabled {
			util.Cyan.Println("- Run 'kindle-send daemon start' to start the background daemon")
			util.Cyan.Println("- Run 'kindle-send daemon status' to check daemon status")
		} else {
			util.Cyan.Println("- Daemon is disabled. Use 'kindle-send send <files/urls>' for one-time sending")
			util.Cyan.Println("- Run 'kindle-send configure' again to enable daemon mode")
		}
	},
}
