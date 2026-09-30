# columnSpec generates Go grain records

`columnSpec` remains the sole authored SoT for each catalog grain’s wire/SQL
fields (ADR-0005). `Check` and `SuiteRecord` are generated from those slices
(Kind→Go type, snake→exported field, `db`/`json` tags), including
`Client: false` timestamps. We rejected keeping hand-authored structs policed
by tag-match tests: that dual authoring failed the depth deletion test once
Client emit already treated `columnSpec` as SoT. We also rejected dropping
named structs for maps only: `LoadFromDir` / `projectSuites` / HTTP embeds need
typed field literals inside the package. `bindParams` reflection stays; it
already joins by json name and does not need a second generated bind path.

## Considered Options

- **Keep hand structs + tag-match tests** — lowest churn; rejected; does not
  unify the dual SoT.
- **Maps / `dbx.Params` only** — rejected; rewrites in-package construction and
  scan embeds for little leverage.
- **Generate grain records from columnSpec (chosen)** — one authored table;
  FE emit and Go records share the same generate entrypoint family.
