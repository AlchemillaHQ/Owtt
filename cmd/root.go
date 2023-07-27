package cmd

import (
	"os"

	"github.com/AlchemillaHQ/owtt/tracker"
	"github.com/AlchemillaHQ/owtt/utils"
	"github.com/spf13/cobra"
)

var port int
var sslKey, sslCert string
var debug bool
var maxOffers int
var announceInterval int
var reverseProxy bool
var rateLimit int
var rateLimitExpiry int

var rootCmd = &cobra.Command{
	Use:   "owtt",
	Short: "A go based webtorrent tracker",
	Run: func(cmd *cobra.Command, args []string) {
		if port <= 0 || port > 65535 {
			utils.LogError("Port must be between 1 and 65535")
			os.Exit(1)
		}

		utils.InitLogging(debug)

		tracker := tracker.NewWebtorrentTracker(port, "0.0.0.0", sslKey, sslCert, maxOffers, announceInterval, reverseProxy, rateLimit, rateLimitExpiry)
		tracker.Run()
	},
}

func Execute() {
	rootCmd.Flags().IntVarP(&port, "port", "p", 8000, "Port")
	rootCmd.Flags().StringVarP(&sslKey, "ssl-key", "k", "", "SSL key")
	rootCmd.Flags().StringVarP(&sslCert, "ssl-cert", "c", "", "SSL certificate")
	rootCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Debug")
	rootCmd.Flags().IntVarP(&maxOffers, "max-offers", "m", 20, "Max offers")
	rootCmd.Flags().IntVarP(&announceInterval, "announce-interval", "a", 15, "Announce interval")
	rootCmd.Flags().BoolVarP(&reverseProxy, "reverse-proxy", "r", false, "Reverse proxy")
	rootCmd.Flags().IntVarP(&rateLimit, "rate-limit", "l", 10, "Rate limit")
	rootCmd.Flags().IntVarP(&rateLimitExpiry, "rate-limit-expiry", "e", 2, "Rate limit expiration in seconds")

	if err := rootCmd.Execute(); err != nil {
		utils.LogError(err.Error())
		os.Exit(1)
	}
}
