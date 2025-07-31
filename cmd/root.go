package cmd

import (
	"fmt"
	"os"

	"github.com/nikhil1raghav/kindle-send/internal/config"
	"github.com/nikhil1raghav/kindle-send/internal/util"
	"github.com/spf13/cobra"
)

func init() {
	var configPath string
	configPath, err := config.DefaultConfigPath()
	if err != nil {
		util.Red.Println("Error setting default config path: ", err)
		os.Exit(1)
	}
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", configPath, "Path to config file")

}

var rootCmd = &cobra.Command{
	Use:   "kindle-send",
	Short: "Background daemon for sending documents and webpages to your ereader",
	Long: `kindle-send is a background daemon that continuously monitors bookmark files
and automatically sends new content to your ereader. It can also be used for
one-time sending of files, documents, and webpages.

The daemon monitors a configured bookmark file/folder and automatically:
- Downloads webpages and converts them to ebooks
- Sends the converted content to your ereader via email
- Keeps track of processed bookmarks to avoid duplicates

Complete documentation is available at https://github.com/nikhil1raghav/kindle-send`,
	Run: func(cmd *cobra.Command, args []string) {
		// Show help if no command is provided
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
