# github-exporter

Prometheus exporter for GitHub **self-hosted Actions runners**, authenticated as
a **GitHub App** installation. On every Prometheus scrape it queries the GitHub
API for the organization's runners and exposes their state.

## Metrics

| Metric | Type | Labels | Description |
|---|---|---|---|
| `github_runner_status` | gauge | `org`, `runner`, `name`, `os` | `1` if the runner is online, else `0` |
| `github_runner_busy` | gauge | `org`, `runner`, `name`, `os` | `1` if the runner is currently running a job, else `0` |
| `github_runners_total` | gauge | `org` | Total number of registered runners |
| `github_runners_online_total` | gauge | `org` | Number of online runners |
| `github_runners_busy_total` | gauge | `org` | Number of busy runners |
| `github_scrape_success` | gauge | `org` | `1` if the last GitHub API scrape succeeded, else `0` |

> The number of **free** runners (a proxy for spare queue capacity) can be
> derived in PromQL as `github_runners_online_total - github_runners_busy_total`.

Standard Go runtime and process collectors are also exposed.

## Configuration

All settings come from environment variables.

| Variable | Required | Default | Description |
|---|---|---|---|
| `APP_ID` | yes | — | GitHub App ID |
| `INSTALLATION_ID` | yes | — | GitHub App installation ID |
| `PRIVATE_KEY` | yes | — | Path to the App's PEM private key |
| `GITHUB_ORG` | yes | — | Organization whose runners are scraped |
| `LISTEN_ADDRESS` | no | `:9101` | HTTP listen address |
| `METRICS_PATH` | no | `/metrics` | Metrics endpoint path |
| `GITHUB_API_URL` | no | public github.com | API base URL for GitHub Enterprise Server |
| `SCRAPE_TIMEOUT` | no | `30s` | Per-scrape timeout for the GitHub API (Go duration) |

The GitHub App needs the **Self-hosted runners: Read-only** organization
permission.

## Endpoints

- `GET /metrics` — Prometheus metrics
- `GET /healthz` — liveness/readiness probe
- `GET /` — landing page

## Running

```sh
export APP_ID=123456
export INSTALLATION_ID=7891011
export PRIVATE_KEY=/etc/github-exporter/app.pem
export GITHUB_ORG=my-org

go run ./cmd
```

## Docker

```sh
docker build -t github-exporter .
docker run --rm -p 9101:9101 \
  -e APP_ID=123456 \
  -e INSTALLATION_ID=7891011 \
  -e GITHUB_ORG=my-org \
  -e PRIVATE_KEY=/etc/github-exporter/app.pem \
  -v /path/to/app.pem:/etc/github-exporter/app.pem:ro \
  github-exporter
```

The image is a multi-stage build on `distroless/static` (~19 MB), running as a
non-root user with a read-only root filesystem.

## Deploy to Kubernetes (werf + Helm)

The chart lives in [.helm](.helm). Build and deploy with werf:

```sh
werf converge --env production \
  --set config.appId=123456 \
  --set config.installationId=7891011 \
  --set config.org=my-org \
  --set-file privateKey.value=/path/to/app.pem
```

Or render/apply the chart with plain Helm (set the image explicitly):

```sh
helm template .helm \
  --set image.repository=registry.example.com/github-exporter \
  --set image.tag=v0.1.0 \
  --set config.appId=123456 \
  --set config.installationId=7891011 \
  --set config.org=my-org \
  --set-file privateKey.value=/path/to/app.pem | kubectl apply -f -
```

### Private key

Provide the GitHub App PEM key in one of two ways:

- `privateKey.value` — the chart creates a `Secret` from it.
- `privateKey.existingSecret` (+ `privateKey.existingSecretKey`) — reference a
  pre-created `Secret` (recommended for production; keep the key out of values).

### Prometheus scraping

Enable the `ServiceMonitor` (Prometheus Operator) with
`--set serviceMonitor.enabled=true`. Without the operator, add a scrape config
pointing at the Service on port `9101`, path `/metrics`.

## Development

```sh
go build ./...
go vet ./...
go test ./...
```
