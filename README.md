# goserve

A fast, zero-config HTTP file server for local development. Inspired by the [`http-server`](https://github.com/http-party/http-server) npm package, built in Go.

## Features

- **Directory listing** — clean HTML UI with breadcrumb navigation, sorted dirs and files
- **Gzip compression** — automatic content encoding for supported clients
- **Cache control** — configurable `Cache-Control` headers, or disabled entirely
- **Basic authentication** — protect your server with a username and password
- **TLS/HTTPS** — serve over HTTPS with your own certificate and key
- **CORS** — add permissive `Access-Control-Allow-Origin: *` headers
- **SPA mode** — serve `index.html` for all unmatched routes (client-side routing)
- **Default extension** — resolve `/about` → `/about.html` automatically
- **Custom 404 page** — place a `404.html` in the root to use it
- **Dotfile protection** — hide and deny access to dotfiles
- **Symlink support** — optionally follow symbolic links
- **Access logging** — Apache-style logs with IP, method, path, status, and elapsed time
- **Browser auto-open** — launch the browser automatically on start
- **Graceful shutdown** — handles `SIGINT`/`SIGTERM` with a 5-second drain

## Installation

**From source:**

```sh
go install goserve@latest
```

**Or clone and build:**

```sh
git clone https://github.com/yourname/goserve
cd goserve
go build -o goserve .
```

## Usage

```
goserve [path] [flags]
```

Serve the current directory on port 8080:

```sh
goserve
```

Serve a specific directory:

```sh
goserve ./dist
```

Serve on a different port:

```sh
goserve -p 3000
```

## Flags

| Flag            | Short | Default    | Description                                        |
| --------------- | ----- | ---------- | -------------------------------------------------- |
| `--port`        | `-p`  | `8080`     | Port to listen on                                  |
| `--address`     | `-a`  | _(all)_    | Address to bind to                                 |
| `--no-listing`  | `-d`  | `false`    | Disable directory listing                          |
| `--silent`      | `-s`  | `false`    | Suppress all log output                            |
| `--no-gzip`     |       | `false`    | Disable gzip compression                           |
| `--cache`       | `-c`  | `-1`       | Cache `max-age` in seconds (`-1` disables caching) |
| `--username`    |       |            | Username for basic auth (requires `--password`)    |
| `--password`    |       |            | Password for basic auth (requires `--username`)    |
| `--tls`         | `-S`  | `false`    | Enable TLS/HTTPS                                   |
| `--cert`        | `-C`  | `cert.pem` | Path to TLS certificate                            |
| `--key`         | `-K`  | `key.pem`  | Path to TLS private key                            |
| `--cors`        |       | `false`    | Enable CORS (`Access-Control-Allow-Origin: *`)     |
| `--no-dotfiles` |       | `false`    | Hide dotfiles and deny access to them              |
| `--timeout`     | `-t`  | `120`      | Connection timeout in seconds (`0` to disable)     |
| `--ext`         | `-e`  | `html`     | Default extension for extensionless URLs           |
| `--open`        | `-o`  | `false`    | Open browser after starting                        |
| `--utc`         |       | `false`    | Use UTC timestamps in logs                         |
| `--symlinks`    |       | `false`    | Follow symbolic links                              |
| `--spa`         |       | `false`    | SPA mode — serve `index.html` for unmatched paths  |

## Examples

**Serve a React/Vue/Svelte build with SPA routing:**

```sh
goserve ./dist --spa
```

**Password-protect a directory:**

```sh
goserve ./private --username admin --password secret
```

**Serve over HTTPS:**

```sh
goserve --tls --cert cert.pem --key key.pem
```

**Share files on the local network with caching enabled:**

```sh
goserve ./files --cache 3600
```

**Serve a static site quietly (no logs) with browser auto-open:**

```sh
goserve ./site --silent --open
```
