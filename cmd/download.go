package cmd

import (
	"os"

	"github.com/lithammer/dedent"
	"github.com/ryan-gang/kindle-send-daemon/internal/classifier"
	"github.com/ryan-gang/kindle-send-daemon/internal/cmdutil"
	"github.com/ryan-gang/kindle-send-daemon/internal/handler"
	"github.com/ryan-gang/kindle-send-daemon/internal/util"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(downloadCmd)
}

var (
	helpDownload = `Downloads the webpage or collection of webpages from given arguments
that can be a standalone link or a text file containing multiple links.
Supports multiple arguments. Each argument is downloaded as a separate file.`

	exampleDownload = dedent.Dedent(`
		# Download a single webpage
		kindle-send download "http://paulgraham.com/alien.html"

		# Download multiple webpages
		kindle-send download "http://paulgraham.com/alien.html" "http://paulgraham.com/hwh.html"

		# Download webpage and collection of webpages
		kindle-send download "http://paulgraham.com/alien.html" links.txt`,
	)
)

var downloadCmd = &cobra.Command{
	Use:     "download [LINK1] [LINK2] [FILE1] [FILE2]",
	Short:   "Download the webpage as ebook and save locally",
	Long:    helpDownload,
	Example: exampleDownload,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := cmdutil.LoadConfigOrExit(cmd)
		if cfg == nil {
			return
		}

		downloadRequests := classifier.Classify(args)
		downloadedRequests := handler.Queue(downloadRequests)

		util.CyanBold.Printf("Downloaded %d files :\n", len(downloadRequests))
		for idx, req := range downloadedRequests {
			fileInfo, err := os.Stat(req.Path)
			if err != nil {
				util.Red.Printf("Error getting file info for %s: %v\n", req.Path, err)
				continue
			}
			util.Cyan.Printf("%d. %s\n", idx+1, fileInfo.Name())
		}

	},
}
