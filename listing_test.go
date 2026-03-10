package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildBreadcrumbs

func TestBuildBreadcrumbs_Root(t *testing.T) {
	crumbs := buildBreadcrumbs("/")
	if len(crumbs) != 0 {
		t.Errorf("expected 0 breadcrumbs for root path, got %d", len(crumbs))
	}
}

func TestBuildBreadcrumbs_SingleSegment(t *testing.T) {
	crumbs := buildBreadcrumbs("/docs")
	if len(crumbs) != 1 {
		t.Fatalf("expected 1 breadcrumb, got %d", len(crumbs))
	}
	if crumbs[0].Name != "docs" {
		t.Errorf("expected Name 'docs', got %q", crumbs[0].Name)
	}
	if crumbs[0].Href != "/docs/" {
		t.Errorf("expected Href '/docs/', got %q", crumbs[0].Href)
	}
	if !crumbs[0].IsLast {
		t.Error("expected single crumb to have IsLast=true")
	}
}

func TestBuildBreadcrumbs_MultipleSegments(t *testing.T) {
	crumbs := buildBreadcrumbs("/a/b/c")
	if len(crumbs) != 3 {
		t.Fatalf("expected 3 breadcrumbs, got %d", len(crumbs))
	}
	names := []string{"a", "b", "c"}
	hrefs := []string{"/a/", "/a/b/", "/a/b/c/"}
	for i, c := range crumbs {
		if c.Name != names[i] {
			t.Errorf("crumb[%d]: expected Name %q, got %q", i, names[i], c.Name)
		}
		if c.Href != hrefs[i] {
			t.Errorf("crumb[%d]: expected Href %q, got %q", i, hrefs[i], c.Href)
		}
	}
	if crumbs[2].IsLast != true {
		t.Error("expected last crumb to have IsLast=true")
	}
	if crumbs[0].IsLast || crumbs[1].IsLast {
		t.Error("only the last crumb should have IsLast=true")
	}
}

func TestBuildBreadcrumbs_TrailingSlash(t *testing.T) {
	// /docs/ and /docs should produce the same breadcrumbs
	crumbs := buildBreadcrumbs("/docs/")
	if len(crumbs) != 1 {
		t.Fatalf("expected 1 breadcrumb for '/docs/', got %d", len(crumbs))
	}
	if crumbs[0].Name != "docs" {
		t.Errorf("expected Name 'docs', got %q", crumbs[0].Name)
	}
}

// humanSize

func TestHumanSize_Bytes(t *testing.T) {
	cases := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{500, "500 B"},
		{1023, "1023 B"},
	}
	for _, tc := range cases {
		if got := humanSize(tc.input); got != tc.expected {
			t.Errorf("humanSize(%d) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestHumanSize_Kilobytes(t *testing.T) {
	if got := humanSize(1024); got != "1.0 KB" {
		t.Errorf("expected '1.0 KB', got %q", got)
	}
	if got := humanSize(1536); got != "1.5 KB" {
		t.Errorf("expected '1.5 KB', got %q", got)
	}
}

func TestHumanSize_Megabytes(t *testing.T) {
	if got := humanSize(1024 * 1024); got != "1.0 MB" {
		t.Errorf("expected '1.0 MB', got %q", got)
	}
}

func TestHumanSize_Gigabytes(t *testing.T) {
	if got := humanSize(1024 * 1024 * 1024); got != "1.0 GB" {
		t.Errorf("expected '1.0 GB', got %q", got)
	}
}

func TestHumanSize_Terabytes(t *testing.T) {
	if got := humanSize(1024 * 1024 * 1024 * 1024); got != "1.0 TB" {
		t.Errorf("expected '1.0 TB', got %q", got)
	}
}

// serveDirectoryListing

func newListingDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "goserve-listing-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func TestServeDirectoryListing_ContentType(t *testing.T) {
	dir := newListingDir(t)
	rr := httptest.NewRecorder()
	serveDirectoryListing(rr, httptest.NewRequest("GET", "/", nil), dir, "/", &Config{})

	ct := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Errorf("expected text/html content-type, got %q", ct)
	}
}

func TestServeDirectoryListing_ShowsFiles(t *testing.T) {
	dir := newListingDir(t)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("data"), 0644)

	rr := httptest.NewRecorder()
	serveDirectoryListing(rr, httptest.NewRequest("GET", "/", nil), dir, "/", &Config{})

	if !strings.Contains(rr.Body.String(), "readme.txt") {
		t.Error("listing should contain 'readme.txt'")
	}
}

func TestServeDirectoryListing_ShowsSubdirectories(t *testing.T) {
	dir := newListingDir(t)
	os.Mkdir(filepath.Join(dir, "subdir"), 0755)

	rr := httptest.NewRecorder()
	serveDirectoryListing(rr, httptest.NewRequest("GET", "/", nil), dir, "/", &Config{})

	if !strings.Contains(rr.Body.String(), "subdir") {
		t.Error("listing should contain 'subdir'")
	}
}

func TestServeDirectoryListing_NoDotfilesHidesDotEntries(t *testing.T) {
	dir := newListingDir(t)
	os.WriteFile(filepath.Join(dir, ".hidden"), []byte("secret"), 0644)
	os.WriteFile(filepath.Join(dir, "visible.txt"), []byte("public"), 0644)

	rr := httptest.NewRecorder()
	serveDirectoryListing(rr, httptest.NewRequest("GET", "/", nil), dir, "/", &Config{NoDotfiles: true})

	body := rr.Body.String()
	if strings.Contains(body, ".hidden") {
		t.Error("listing with NoDotfiles should not contain '.hidden'")
	}
	if !strings.Contains(body, "visible.txt") {
		t.Error("listing with NoDotfiles should still contain 'visible.txt'")
	}
}

func TestServeDirectoryListing_NoDotfilesWithoutFlag(t *testing.T) {
	dir := newListingDir(t)
	os.WriteFile(filepath.Join(dir, ".env"), []byte("secret"), 0644)

	rr := httptest.NewRecorder()
	serveDirectoryListing(rr, httptest.NewRequest("GET", "/", nil), dir, "/", &Config{NoDotfiles: false})

	if !strings.Contains(rr.Body.String(), ".env") {
		t.Error("listing without NoDotfiles should show '.env'")
	}
}

func TestServeDirectoryListing_ShowsBreadcrumbs(t *testing.T) {
	dir := newListingDir(t)
	rr := httptest.NewRecorder()
	serveDirectoryListing(rr, httptest.NewRequest("GET", "/docs/api/", nil), dir, "/docs/api", &Config{})

	body := rr.Body.String()
	// Both path segments should appear in the rendered breadcrumbs
	if !strings.Contains(body, "docs") {
		t.Error("expected 'docs' in breadcrumbs")
	}
	if !strings.Contains(body, "api") {
		t.Error("expected 'api' in breadcrumbs")
	}
}

func TestServeDirectoryListing_EntriesSortedDirsFirst(t *testing.T) {
	dir := newListingDir(t)
	os.WriteFile(filepath.Join(dir, "aaa.txt"), []byte("a"), 0644)
	os.Mkdir(filepath.Join(dir, "zzz"), 0755)

	rr := httptest.NewRecorder()
	serveDirectoryListing(rr, httptest.NewRequest("GET", "/", nil), dir, "/", &Config{})

	body := rr.Body.String()
	zzzPos := strings.Index(body, "zzz")
	aaaPos := strings.Index(body, "aaa.txt")
	if zzzPos == -1 || aaaPos == -1 {
		t.Fatal("expected both 'zzz' and 'aaa.txt' in listing")
	}
	if zzzPos > aaaPos {
		t.Error("directories should appear before files in listing")
	}
}

// symlink listing

func TestServeDirectoryListing_SymlinkHasIndicator(t *testing.T) {
	dir := newListingDir(t)
	target := filepath.Join(dir, "real.txt")
	os.WriteFile(target, []byte("data"), 0644)
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	rr := httptest.NewRecorder()
	serveDirectoryListing(rr, httptest.NewRequest("GET", "/", nil), dir, "/", &Config{})

	body := rr.Body.String()
	if !strings.Contains(body, "link.txt") {
		t.Error("symlink should appear in listing")
	}
	if !strings.Contains(body, `data-symlink="1"`) {
		t.Error("symlink entry should have data-symlink=1 attribute")
	}
}

func TestServeDirectoryListing_RegularFileNotSymlink(t *testing.T) {
	dir := newListingDir(t)
	os.WriteFile(filepath.Join(dir, "plain.txt"), []byte("data"), 0644)

	rr := httptest.NewRecorder()
	serveDirectoryListing(rr, httptest.NewRequest("GET", "/", nil), dir, "/", &Config{})

	body := rr.Body.String()
	if !strings.Contains(body, `data-symlink="0"`) {
		t.Error("regular file should have data-symlink=0")
	}
}
