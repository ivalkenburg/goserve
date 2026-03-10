package main

import (
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"sort"
	"strings"
)

//go:embed listing.html
var listingHTML string

var listingTmpl = template.Must(template.New("listing").Parse(listingHTML))

type breadcrumb struct {
	Name   string
	Href   string
	IsLast bool
}

type dirEntry struct {
	Name      string
	IsDir     bool
	IsSymlink bool
	Ext       string
	Size      string
	SizeBytes int64
	ModTime   string
}

type listingData struct {
	Path        string
	Breadcrumbs []breadcrumb
	Entries     []dirEntry
	Version     string
}

func buildBreadcrumbs(urlPath string) []breadcrumb {
	if urlPath == "/" {
		return nil
	}
	parts := strings.Split(strings.Trim(urlPath, "/"), "/")
	crumbs := make([]breadcrumb, 0, len(parts))
	href := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		href += "/" + p
		crumbs = append(crumbs, breadcrumb{Name: p, Href: href + "/"})
	}
	if len(crumbs) > 0 {
		crumbs[len(crumbs)-1].IsLast = true
	}
	return crumbs
}

func serveDirectoryListing(w http.ResponseWriter, _ *http.Request, dir, urlPath string, cfg *Config) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		http.Error(w, "Error reading directory", http.StatusInternalServerError)
		return
	}

	var dirs, files []dirEntry
	for _, e := range entries {
		if cfg.NoDotfiles && strings.HasPrefix(e.Name(), ".") {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		isSymlink := e.Type()&os.ModeSymlink != 0
		de := dirEntry{
			Name:      e.Name(),
			IsDir:     e.IsDir(),
			IsSymlink: isSymlink,
			SizeBytes: fi.Size(),
			ModTime:   fi.ModTime().Format("2006-01-02 15:04"),
		}
		if !e.IsDir() {
			de.Size = humanSize(fi.Size())
			if i := strings.LastIndex(e.Name(), "."); i > 0 {
				de.Ext = e.Name()[i+1:]
			}
		}
		if e.IsDir() {
			dirs = append(dirs, de)
		} else {
			files = append(files, de)
		}
	}

	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	data := listingData{
		Path:        urlPath,
		Breadcrumbs: buildBreadcrumbs(urlPath),
		Entries:     append(dirs, files...),
		Version:     version,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := listingTmpl.Execute(w, data); err != nil {
		fmt.Fprintf(os.Stderr, "listing template error: %v\n", err)
	}
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
