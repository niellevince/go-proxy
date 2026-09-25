# go-proxy

A small reverse proxy. It matches the incoming `Host` against `proxies.json`, checks `X-PROXY-KEY`, and forwards the request to an HTTPS upstream. Cloudflare Tunnel publishes the hostnames and forwards them to this process. The proxy does not terminate public TLS.

## Run

From the repo root:

```powershell
go run ./cmd/server
```

The listen port comes from `PORT` in `.env`. The default is `8000`. A `PORT` already set in the environment is left as-is. An empty or invalid value falls back to `8000`.

`proxies.json` must exist in the working directory. It is gitignored. Copy `proxies.example.json` to start, or add a route with the CLI, which creates the file when it is missing.

The server reloads `proxies.json` when the file changes. A restart is not required after adding a route.

## Add a route

```powershell
go run cli.go --from hello.world.com --to hi.world.com
```

`--to` can be a hostname or a full `https://` origin. A bare hostname is stored as `https://hi.world.com`. The CLI generates a 32-byte hex key, saves the route, and prints `from`, `to`, and `key`. Copy the key. A duplicate `--from` is rejected and the existing key is left unchanged.

```powershell
go run cli.go --file proxies.json --from hello.world.com --to https://hi.world.com/base
```

`--file` defaults to `proxies.json`.

## Config

```json
{
  "healthHost": "health.example.com",
  "proxies": [
    {
      "from": "hello.world.com",
      "to": "https://hi.world.com",
      "key": "CHANGE_ME"
    }
  ]
}
```

Hosts are matched without case sensitivity. A port on the incoming `Host` header is ignored.

| Request | Result |
|---|---|
| `Host` is `healthHost` | `200` `ok`, no key check, not proxied |
| `Host` is not listed | `404` |
| Missing or wrong `X-PROXY-KEY` | `403` |
| Key matches | Proxied to `to` |

The path and query are forwarded. `Host` and TLS SNI are set to the upstream host. `X-PROXY-KEY` is removed before the request leaves. An upstream failure returns `502`.

Send the key on each proxied request:

```powershell
curl -H "Host: hello.world.com" -H "X-PROXY-KEY: the-printed-key" http://127.0.0.1:8000/path
```

## Cloudflare Tunnel

`cloudflared` runs separately and forwards ingress to `http://127.0.0.1:8000`. `cloudflared.example.yml` is a starting ingress list. Replace the hostnames with the ones in `proxies.json`. Turn on Cloudflare “Always Use HTTPS” so port 80 is not needed here.
