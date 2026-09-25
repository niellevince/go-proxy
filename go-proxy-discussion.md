# Go proxy discussion handoff

Summary of the nginx review and the Go + Cloudflare Tunnel idea. Continue from this file on another machine. No code was written for the Go proxy. The only doc created here besides this note is `NGINX_PROXY_SETUP.md`.

## Current host

- Workspace: `/home/ubuntu/nginx-agent` (empty aside from these markdown files).
- Nginx: `1.28.3` (Ubuntu), config path `/etc/nginx/nginx.conf`.
- `nginx -t` as a normal user fails because it cannot read `/etc/ssl/private/cloudflare.key` (`Permission denied`). Run the test as root. The `user` directive warning in that non-root test is expected.
- `conf.d/` and `modules-enabled/` are empty.
- Enabled site: `/etc/nginx/sites-enabled/default` → `/etc/nginx/sites-available/default`.
- Stock Ubuntu site is saved as `/etc/nginx/sites-available/default.bk` and is not enabled.
- `/etc/nginx/nginx.conf` is the package default: `www-data`, `worker_processes auto`, 768 connections, TLS 1.2/1.3, gzip on for the default types only, `server_tokens build` globally, and `include /etc/nginx/sites-enabled/*` inside `http`.

## What the live site does

It is an origin reverse proxy in front of Cloudflare. HTTPS is terminated with a Cloudflare origin certificate. Only listed hostnames are forwarded. One shared `X-PROXY-KEY` covers every proxied host.

| Incoming host | Result |
|---|---|
| `qtplatform.cockxing.bet` | Proxied to `https://product.qtplatform.com` |
| `qtplatform-int.cockxing.bet` | Proxied to `https://product-int.qtplatform.com` |
| `ifconfig.cockxing.bet` | Proxied to `https://ifconfig.me` |
| Any other `*.cockxing.bet` | Connection closed (`444`), no body |
| Wrong or missing `X-PROXY-KEY` | `403 Forbidden` plain text |
| `health.cockxing.bet` | `200 OK` plain text, no key check, not proxied |
| Port 80 for `*.cockxing.bet` | `301` to HTTPS |

Request path on 443:

1. A `map` turns `$host` into an upstream hostname. Unlisted hosts stay empty.
2. An empty upstream returns `444`.
3. `X-PROXY-KEY` must match the secret in the site file. A mismatch returns `403`.
4. `proxy_pass https://$mapped_target` with `Host` and TLS SNI set to the upstream. The key header is stripped before the request leaves. HTTP/1.1 with an empty `Connection` header.
5. Upstream DNS is resolved per request via `1.1.1.1` and `8.8.8.8`, cached 30 seconds. The resolver is required because `proxy_pass` uses a variable.
6. On 443, the exact name `health.cockxing.bet` wins over `*.cockxing.bet`.

Certificate paths: `/etc/ssl/certs/cloudflare.crt` and `/etc/ssl/private/cloudflare.key`.

The live key is only in `/etc/nginx/sites-available/default`. Do not copy it into docs or into the other project. Generate a new key there.

`NGINX_PROXY_SETUP.md` in this folder is a recreation guide for that nginx layout, with placeholders (`example.com`, `CHANGE_ME`).

## Decisions already made (questions only, no code)

**Different keys per hostname.** Yes. The live config uses one shared secret. Nginx can map each hostname to its own expected key and compare the header to that value. The health host can stay open.

**Reimplement in Go with `proxies.json` and `cloudflared`.** Yes.

- A local Go HTTP server reads `proxies.json` (hostname → upstream, plus the key or per-host keys), checks `X-PROXY-KEY`, and reverse-proxies over HTTPS. Rewrite `Host`, set upstream SNI, strip the key, and pass the path and query through.
- `cloudflared` publishes those hostnames through a Cloudflare Tunnel and forwards them to the Go process on localhost.
- No inbound 443 and no origin certificate. Cloudflare terminates public TLS. The tunnel carries traffic to the process.
- Unlisted hosts cannot use nginx `444` through the tunnel. Return a normal status such as 404.
- Port 80 can stay off. Use Cloudflare “Always Use HTTPS” for the redirect.
- Health can be another route in Go, still without a key check.
- The Go process must stay running. The tunnel only forwards to it.
- `proxies.json` is tiny. A handful of host entries is a few kilobytes.

**Memory.** Idle expectation: nginx on this host about 15–40 MB. A Go proxy about 15–40 MB. `cloudflared` about 30–50 MB. Go plus tunnel is about 50–100 MB total, versus nginx alone. The extra cost is the second Go process (`cloudflared`), not the JSON file. Memory grows with concurrent in-flight requests (about a 32 KB copy buffer plus a goroutine each). Both sides stream large bodies. These are typical ranges, not measurements from this machine.

## Suggested next step on the local agent

Build the Go proxy and tunnel config for the other project, not by copying this server’s hostnames or key. Match the behavior above: hostname map from `proxies.json`, header key check, upstream HTTPS with `Host`/SNI rewrite, key stripped, health route without a key, 404 (or similar) for unknown hosts, `cloudflared` ingress to the local port.