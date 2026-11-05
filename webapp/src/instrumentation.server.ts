// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Loaded before the server starts (kit.experimental.instrumentation.server).
// SvelteKit then emits spans for handle, load and actions, and records
// unexpected errors and 5xx as exceptions. The exporter reads the same OTEL_*
// variables as the Go server, which runs beside the webapp.

import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { envDetector, detectResources, resourceFromAttributes } from '@opentelemetry/resources';
import { BatchSpanProcessor, NodeTracerProvider } from '@opentelemetry/sdk-trace-node';
import { ATTR_SERVICE_NAME } from '@opentelemetry/semantic-conventions';

if (process.env.OTEL_EXPORTER_OTLP_ENDPOINT) {
	const provider = new NodeTracerProvider({
		resource: resourceFromAttributes({ [ATTR_SERVICE_NAME]: 'credimi-webapp' }).merge(
			detectResources({ detectors: [envDetector] })
		),
		spanProcessors: [new BatchSpanProcessor(new OTLPTraceExporter())]
	});
	provider.register();
	process.once('SIGTERM', () => void provider.shutdown());
}
