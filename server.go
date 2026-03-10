package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

// Start validates config, binds a TCP listener, starts the HTTP server,
// and blocks until an OS interrupt or a fatal error occurs.
func Start(cfg *Config) error {
	info, err := os.Stat(cfg.Root)
	if err != nil {
		return fmt.Errorf("cannot access %q: %w", cfg.Root, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", cfg.Root)
	}
	if (cfg.Username == "") != (cfg.Password == "") {
		return fmt.Errorf("--username and --password must both be provided together")
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Address, cfg.Port))
	if err != nil {
		return fmt.Errorf("cannot listen on port %d: %w", cfg.Port, err)
	}

	scheme := "http"
	if cfg.TLS {
		scheme = "https"
	}

	if !cfg.Silent {
		printBanner(cfg, scheme)
	}

	srv := &http.Server{Handler: buildHandler(cfg)}
	if cfg.Timeout > 0 {
		d := time.Duration(cfg.Timeout) * time.Second
		srv.ReadTimeout = d
		srv.WriteTimeout = d
		srv.IdleTimeout = d * 2
	}

	if cfg.OpenBrowser {
		go openBrowser(fmt.Sprintf("%s://127.0.0.1:%d", scheme, cfg.Port))
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	serveErr := make(chan error, 1)
	go func() {
		if cfg.TLS {
			serveErr <- srv.ServeTLS(ln, cfg.Cert, cfg.Key)
		} else {
			serveErr <- srv.Serve(ln)
		}
	}()

	select {
	case err := <-serveErr:
		if err != http.ErrServerClosed {
			return err
		}
	case <-quit:
		if !cfg.Silent {
			fmt.Println("\nShutting down...")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	}
	return nil
}

func printBanner(cfg *Config, scheme string) {
	fmt.Printf("\ngoserve v%s — serving %q\n\n", version, cfg.Root)
	fmt.Printf("  http://127.0.0.1:%d\n", cfg.Port)
	if cfg.Address == "" {
		if addrs, err := localAddresses(); err == nil {
			for _, addr := range addrs {
				fmt.Printf("  %s://%s:%d\n", scheme, addr, cfg.Port)
			}
		}
	}
	fmt.Println()

	flags := []struct{ label, val string }{
		{"Gzip", func() string {
			if cfg.NoGzip {
				return "disabled"
			}
			return "enabled"
		}()},
		{"Cache", func() string {
			if cfg.Cache < 0 {
				return "disabled"
			}
			return fmt.Sprintf("%ds max-age", cfg.Cache)
		}()},
		{"Listing", func() string {
			if cfg.NoListing {
				return "disabled"
			}
			return "enabled"
		}()},
		{"Auth", func() string {
			if cfg.Username != "" {
				return "basic"
			}
			return "none"
		}()},
		{"CORS", func() string {
			if cfg.CORS {
				return "enabled"
			}
			return "disabled"
		}()},
	}
	for _, f := range flags {
		fmt.Printf("  %-10s %s\n", f.label+":", f.val)
	}
	if cfg.TLS {
		fmt.Printf("  %-10s %s / %s\n", "TLS:", cfg.Cert, cfg.Key)
	}
	fmt.Print("\n  Hit CTRL-C to stop\n\n")
}

// localAddresses returns non-loopback IPv4 addresses of the host.
func localAddresses() ([]string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var addrs []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		ifAddrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range ifAddrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip4 := ip.To4(); ip4 != nil {
				addrs = append(addrs, ip4.String())
			}
		}
	}
	return addrs, nil
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd, args = "open", []string{url}
	case "windows":
		cmd, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd, args = "xdg-open", []string{url}
	}
	_ = exec.Command(cmd, args...).Start()
}
