# FCAF missing source tests design

## Goal

Close the `Unimplemented source tests` section of the relying-party FCAF
backlog without claiming assertions that beta Capture Wallet cannot prove.

## Evidence design

Every implemented test binds to the scenario that creates its own Capture
session. Assertions inspect the signed request, raw encrypted response, or
decoded presentation produced by that same session. Screenshots remain visual
evidence only and never prove a protocol response.

The supported cases reuse five evidence families: custom-scheme engagement,
PID SD-JWT and mdoc DCQL presentations, request-URI delivery, status-list
issuance plus presentation, and encrypted direct-post JWT responses. Shared
scenarios may serve multiple source tests only when their requested property
and response artifact are identical.

## Boundaries

The beta service cannot create invalid signed Request Objects, a non-POST
reference Wallet profile, a Digital Credentials API exchange, or Wallet RP
registration-certificate variants. Those source tests move to the backlog's
`Blocked` section with one concrete missing capability each. Its live issuer
metadata advertises both SD-JWT and mdoc PID configurations, so the positive
mdoc Token Status List assertions remain in scope.

The four request-URI transport cases are implemented using Capture's raw
Wallet retrieval method and header capture, bound to its session-specific
request endpoint. A generic retrieval event does not prove those source
assertions.

## Validation

New Go validators receive focused table-driven unit tests before their test
definition consumes them. Regenerate the aggregate with `make fcaf-generate`,
run `go test ./cmd/fcaf-pipeline-gen ./pkg/fcaf/... ./pkg/internal/pipeline`,
and run `git diff --check`.
