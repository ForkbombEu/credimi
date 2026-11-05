// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { type Attributes, SpanStatusCode, trace } from '@opentelemetry/api';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { registerInstrumentations } from '@opentelemetry/instrumentation';
import { FetchInstrumentation } from '@opentelemetry/instrumentation-fetch';
import { resourceFromAttributes } from '@opentelemetry/resources';
import { BatchSpanProcessor } from '@opentelemetry/sdk-trace-base';
import { WebTracerProvider } from '@opentelemetry/sdk-trace-web';
import { ATTR_SERVICE_NAME, ATTR_SERVICE_VERSION } from '@opentelemetry/semantic-conventions';

// Credimi forwards these batches to the collector with its own credentials
// (observability.BrowserTracesPath) and labels them credimi-browser.
const TRACES_PATH = '/api/otlp/v1/traces';

export function start(version: string) {
	const provider = new WebTracerProvider({
		resource: resourceFromAttributes({
			[ATTR_SERVICE_NAME]: 'credimi-browser',
			[ATTR_SERVICE_VERSION]: version
		}),
		spanProcessors: [new BatchSpanProcessor(new OTLPTraceExporter({ url: TRACES_PATH }))]
	});
	provider.register();
	// Same-origin requests only carry traceparent: issuers and verifiers under
	// test never see Credimi trace headers.
	registerInstrumentations({
		instrumentations: [new FetchInstrumentation({ ignoreUrls: [TRACES_PATH] })]
	});

	const tracer = trace.getTracer('credimi-webapp');
	return (error: unknown, attributes: Attributes) => {
		const exception = error instanceof Error ? error : String(error);
		const span = tracer.startSpan('exception', { attributes });
		span.recordException(exception);
		span.setStatus({
			code: SpanStatusCode.ERROR,
			message: error instanceof Error ? error.message : String(error)
		});
		span.end();
	};
}
