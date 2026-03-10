package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const version = "1.0.0"

// Config holds all server configuration options.
type Config struct {
	Root        string
	Port        int
	Address     string
	NoListing   bool
	Silent      bool
	NoGzip      bool
	Cache       int
	Username    string
	Password    string
	TLS         bool
	Cert        string
	Key         string
	CORS        bool
	NoDotfiles  bool
	Timeout     int
	Ext         string
	OpenBrowser bool
	UTC         bool
	Symlinks    bool
	SPA         bool
}

var cfg Config

var rootCmd = &cobra.Command{
	Use:   "goserve [path]",
	Short: "A fast, simple HTTP file server for local development",
	Long: `goserve serves a directory over HTTP, similar to the http-server npm package.

By default it serves the current directory on port 8080 with directory
listing, gzip compression, caching, and full request logging enabled.`,
	Version:       version,
	Args:          cobra.MaximumNArgs(1),
	RunE:          run,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	f := rootCmd.Flags()
	f.IntVarP(&cfg.Port, "port", "p", 8080, "Port to listen on")
	f.StringVarP(&cfg.Address, "address", "a", "", "Address to bind to (default: all interfaces)")
	f.BoolVarP(&cfg.NoListing, "no-listing", "d", false, "Disable directory listing")
	f.BoolVarP(&cfg.Silent, "silent", "s", false, "Suppress all log output")
	f.BoolVar(&cfg.NoGzip, "no-gzip", false, "Disable gzip compression")
	f.IntVarP(&cfg.Cache, "cache", "c", -1, "Cache max-age in seconds (-1 to disable caching)")
	f.StringVar(&cfg.Username, "username", "", "Username for basic authentication")
	f.StringVar(&cfg.Password, "password", "", "Password for basic authentication")
	f.BoolVarP(&cfg.TLS, "tls", "S", false, "Enable TLS/HTTPS")
	f.StringVarP(&cfg.Cert, "cert", "C", "cert.pem", "Path to TLS certificate file")
	f.StringVarP(&cfg.Key, "key", "K", "key.pem", "Path to TLS key file")
	f.BoolVar(&cfg.CORS, "cors", false, "Enable CORS (Access-Control-Allow-Origin: *)")
	f.BoolVar(&cfg.NoDotfiles, "no-dotfiles", false, "Hide dotfiles and deny access to them")
	f.IntVarP(&cfg.Timeout, "timeout", "t", 120, "Connection timeout in seconds (0 to disable)")
	f.StringVarP(&cfg.Ext, "ext", "e", "html", "Default file extension when none supplied")
	f.BoolVarP(&cfg.OpenBrowser, "open", "o", false, "Open browser after starting the server")
	f.BoolVar(&cfg.UTC, "utc", false, "Use UTC time format in log messages")
	f.BoolVar(&cfg.Symlinks, "symlinks", false, "Follow symbolic links")
	f.BoolVar(&cfg.SPA, "spa", false, "Serve root index.html for all unmatched paths (SPA mode)")
}

func run(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		cfg.Root = args[0]
	} else {
		cfg.Root = "."
	}
	return Start(&cfg)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
