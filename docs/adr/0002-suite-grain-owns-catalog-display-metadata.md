# Suite grain owns catalog display metadata

Suite display fields (`name`, logo, homepage, repository, help, description) live only on the suite projection (`conformance_suites`). Nested browse/pickers build from suite rows grouped on filesystem `fs_standard` / `fs_version` / `suite`, not from denormalized copies on every check. Check rows stay lean (path, title, facets, identity).

This supersedes the v1 HITL #1399 choice to denorm suite meta onto each `conformance_checks` row. That denorm was correct when only flat checks existed and a second meta fetch hurt; once suite rows carry meta plus member `check_paths` / titles / files, fan-out onto checks failed the depth deletion test and forced `ProjectSuites` to re-lift the same strings.

## Considered Options

- **Keep check denorm** — preserves nest-from-checks; rejected as ongoing schema and locality tax.
- **Lean checks + join suite meta on the client** — two fetches; rejected when suite rows already include members.
- **Nest from suite rows only (chosen)** — one suite-grain read for browse; checks remain for path/facet list use.

## Dual browse axes

Product fields (`standard` / `component` / `version`) drive the hub suite **table**.
Filesystem fields (`fs_standard` / `fs_version` / `path_prefix`) drive the nested
detail/picker tree so hub URLs stay FS-path-stable. Hub routes load each
projection only where needed (table vs conformance-checks detail); they do not
dual-fetch on every hub layout.
