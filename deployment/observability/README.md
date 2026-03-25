# Observability

Shared OTel backend for all `credimi-runner` instances.

The services are wired into the main `credimi-2` Docker Compose flow through `docker-compose.override.yml`.

Start from the `credimi-2` repo root:

```bash
docker compose up -d --build
```

Main endpoints:

- OpenSearch Dashboards: `http://localhost:5601`
- Credimi OTel dashboard: `http://localhost:5601/app/dashboards#/view/credimi-otel-overview`
- OpenSearch API: `http://localhost:9200`
- OTLP HTTP ingest: `http://localhost:4318`
- OTLP gRPC ingest: `http://localhost:4317`
- Collector health: `http://localhost:13133`
