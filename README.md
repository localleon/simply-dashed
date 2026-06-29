# simply-dashed

Static dashboard app in Go with HTMX and YAML config.

## Features

- YAML-driven dashboard index with per-dashboard groups and links
- One global search bar with HTMX partial updates
- Local static asset vendoring for airgapped use
- Startup icon refresh with local cache fallback
- Responsive single-page layout using Milligram plus custom styling
- Docker, Helm, and GitHub Actions scaffolding for OCI delivery
- Hardened Helm defaults for restricted pod security environments

## Run

```bash
cp config.example.yaml config.yaml
go mod vendor
go run -mod=vendor ./cmd/iconfetch -config config.yaml -icon-dir data/icons
go run -mod=vendor ./main.go -config config.yaml -refresh-icons=false
```

Open `http://localhost:8080`.

## Vendor pinned frontend assets

```bash
./hack/vendor-assets.sh
```

This downloads exact versions listed in `hack/vendor-assets.lock` into `internal/server/static/vendor/`.

## Vendor icons for airgapped runtime

```bash
go run -mod=vendor ./cmd/iconfetch -config config.yaml -icon-dir data/icons
```

Runtime can then stay offline:

```bash
go run -mod=vendor ./main.go -config config.yaml -icon-dir data/icons -refresh-icons=false
```

## Release flow

- Open and merge PRs normally.
- Create and push a Git tag like `v1.2.3`.
- That tag alone triggers the release workflow.
- CI then builds and signs the container image, generates SBOM and vulnerability reports, packages the Helm chart with matching version metadata, pushes the OCI chart, and creates the GitHub release.

## Helm chart defaults

Chart defaults target stricter clusters:

- `replicaCount: 2`
- non-root runtime with dropped Linux capabilities
- `readOnlyRootFilesystem: true`
- `seccompProfile: RuntimeDefault`
- service account token automount disabled
- readiness, liveness, and startup probes on `/healthz`
- resource requests and limits set
- PDB and topology spread enabled
- values validated with `values.schema.json`

## Config

```yaml
title: Dashboards
subtitle: Static dashboard index
listen_addr: ":8080"
dashboards:
  - path: /infra
    title: Infrastructure
    subtitle: Fleet and runtime tooling
    groups:
      - name: Infrastructure
        description: Fleet and runtime tooling
        links:
          - name: Grafana
            description: Metrics dashboards and alerts
            url: https://grafana.example.com
            icon: https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons/png/grafana.png
```

The root page lists all configured dashboards, and each dashboard serves its own search and link groups under its configured path.
