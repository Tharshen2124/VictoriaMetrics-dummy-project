# VictoriaMetrics + VictoriaLogs Dummy Project

This is a **learning project** to practice setting up and using:

- VictoriaMetrics
- VictoriaLogs
- OpenTelemetry
- Grafana
- Golang (for instrumented demo app)

The goal is to understand how metrics and logs flow through a modern observability stack and how to visualize/query them in Grafana.

## What this project is for

- Learn how to run a local observability stack with Docker Compose.
- Learn how OpenTelemetry can collect/forward telemetry.
- Learn how VictoriaMetrics stores and serves time-series metrics.
- Learn how VictoriaLogs stores and serves logs.
- Learn how Grafana connects to both backends for dashboards and exploration.

## Tech Stack

- **VictoriaMetrics** (`victoriametrics/victoria-metrics`)
- **VictoriaLogs** (`victoriametrics/victoria-logs`)
- **OpenTelemetry Collector** (`otel/opentelemetry-collector-contrib`)
- **Grafana** (`grafana/grafana`)
- **Golang** (to produce sample app telemetry)

## Architecture (high level)

1. A Golang app emits metrics/logs/traces (typically via OpenTelemetry SDK).
2. OpenTelemetry Collector receives telemetry on OTLP HTTP (`4318`).
3. Telemetry is exported to VictoriaMetrics and VictoriaLogs.
4. Grafana reads from VictoriaMetrics and VictoriaLogs for dashboards and queries.

## Services and Ports

From `compose.yml`:

- **VictoriaMetrics**: `http://localhost:8428`
- **VictoriaLogs**: `http://localhost:9201`
- **Grafana**: `http://localhost:3000`
- **OpenTelemetry Collector (OTLP HTTP)**: `http://localhost:4318`

> Current retention in this setup is `2d` for VictoriaMetrics and VictoriaLogs.

## Prerequisites

- Docker + Docker Compose
- (Optional) Go installed locally if you want to run a sample app outside containers

## Getting Started

1. Start the stack:

	```bash
	docker compose up -d
	```

2. Check running services:

	```bash
	docker compose ps
	```

3. Open Grafana:

	- URL: `http://localhost:3000`
	- Default credentials (unless changed): `admin` / `admin`

4. Stop the stack when done:

	```bash
	docker compose down
	```

## Useful Commands

View all logs:

```bash
docker compose logs -f
```

View a single service logs:

```bash
docker compose logs -f victoria-metrics
docker compose logs -f victoria-logs
docker compose logs -f opentelemetry
docker compose logs -f grafana
```

Reset all persisted data (destructive):

```bash
docker compose down -v
```

## Suggested Learning Path

1. Bring up the stack and verify all services are healthy.
2. Send sample telemetry from a tiny Go app using OpenTelemetry.
3. Confirm data ingestion in VictoriaMetrics and VictoriaLogs.
4. Explore data in Grafana (dashboards + query editors).
5. Tune retention and collector pipelines to understand trade-offs.

## Project Structure

```text
.
├── compose.yml
├── README.md
├── grafana-datasources.yaml
└── otel-collector-config.yaml
```

## Notes

- This is intentionally a **dummy/non-production** setup.
- Defaults are kept simple for learning and quick iteration.
- For production, add auth, TLS, backup strategy, and hardened configs.
