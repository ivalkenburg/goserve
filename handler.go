package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/klauspost/compress/gzhttp"
)

// buildHandler chains all configured middleware around the core file handler.
// Order (outermost → innermost): logging → auth → [mux: SSE | gzip → cache → cors → inject → file]
func buildHandler(cfg *Config, r *reloader) http.Handler {
	// Build the file-serving side of the chain.
	var fileH http.Handler = &fileHandler{cfg: cfg}
	if r != nil {
		fileH = liveReloadMiddleware(fileH)
	}
	if cfg.CORS {
		fileH = corsMiddleware(fileH)
	}
	fileH = cacheMiddleware(fileH, cfg)
	if !cfg.NoGzip {
		fileH = gzhttp.GzipHandler(fileH)
	}

	// When live reload is active, route /_goserve/reload to the SSE handler
	// so it bypasses gzip and cache (but still goes through auth and logging).
	var h http.Handler
	if r != nil {
		mux := http.NewServeMux()
		mux.Handle(reloadSSEPath, r)
		mux.Handle("/", fileH)
		h = mux
	} else {
		h = fileH
	}

	if cfg.Username != "" {
		h = basicAuthMiddleware(h, cfg.Username, cfg.Password)
	}
	if !cfg.Silent {
		h = loggingMiddleware(h, cfg)
	}
	return h
}

// statusRecorder wraps http.ResponseWriter to capture the HTTP status code
// and bytes written for access logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
	size   int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.size += n
	return n, err
}

// Flush implements http.Flusher so streaming responses work through the recorder.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func loggingMiddleware(next http.Handler, cfg *Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		elapsed := time.Since(start)

		ts := start
		if cfg.UTC {
			ts = start.UTC()
		}
		fmt.Printf("[%s] %s \"%s %s %s\" %d %s\n",
			ts.Format("2006-01-02 15:04:05"),
			clientIP(r),
			r.Method,
			r.URL.RequestURI(),
			r.Proto,
			rec.status,
			elapsed.Round(time.Millisecond),
		)
	})
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.SplitN(fwd, ",", 2)[0])
	}
	if real := r.Header.Get("X-Real-Ip"); real != "" {
		return real
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func cacheMiddleware(next http.Handler, cfg *Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case cfg.Cache < 0:
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
		case cfg.Cache > 0:
			w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", cfg.Cache))
		}
		next.ServeHTTP(w, r)
	})
}

func basicAuthMiddleware(next http.Handler, username, password string) http.Handler {
	const realm = `Basic realm="goserve"`
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != username || p != password {
			w.Header().Set("WWW-Authenticate", realm)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// fileHandler is the core handler that serves files and directory listings.
type fileHandler struct {
	cfg *Config
}

func (fh *fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// path.Clean resolves ".." so callers cannot escape the root.
	urlPath := path.Clean("/" + r.URL.Path)

	// Deny access to dotfiles when configured.
	if fh.cfg.NoDotfiles {
		for seg := range strings.SplitSeq(urlPath, "/") {
			if len(seg) > 0 && seg[0] == '.' {
				http.NotFound(w, r)
				return
			}
		}
	}

	fsPath := filepath.Join(fh.cfg.Root, filepath.FromSlash(urlPath))

	fi, err := os.Lstat(fsPath)
	if err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if !fh.cfg.Symlinks {
			http.NotFound(w, r)
			return
		}
		fi, err = os.Stat(fsPath) // follow the symlink
	}
	if os.IsNotExist(err) {
		// Try appending the default extension (e.g. /about → /about.html).
		if fh.cfg.Ext != "" {
			candidate := fsPath + "." + fh.cfg.Ext
			if ci, err2 := os.Stat(candidate); err2 == nil && !ci.IsDir() {
				http.ServeFile(w, r, candidate)
				return
			}
		}
		// SPA fallback: serve root index.html so client-side routing works.
		if fh.cfg.SPA {
			if idx := filepath.Join(fh.cfg.Root, "index.html"); fileExists(idx) {
				http.ServeFile(w, r, idx)
				return
			}
		}
		// Serve a custom 404 page if present in the root.
		if p404 := filepath.Join(fh.cfg.Root, "404.html"); fileExists(p404) {
			if content, err := os.ReadFile(p404); err == nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusNotFound)
				w.Write(content) //nolint:errcheck
				return
			}
		}
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if fi.IsDir() {
		// Redirect bare directory paths to their trailing-slash form so that
		// relative links in served HTML resolve correctly.
		if !strings.HasSuffix(r.URL.Path, "/") {
			http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
			return
		}
		// Prefer index.html over directory listing.
		if idx := filepath.Join(fsPath, "index.html"); fileExists(idx) {
			http.ServeFile(w, r, idx)
			return
		}
		if fh.cfg.NoListing {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		serveDirectoryListing(w, r, fsPath, urlPath, fh.cfg)
		return
	}

	http.ServeFile(w, r, fsPath)
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}
