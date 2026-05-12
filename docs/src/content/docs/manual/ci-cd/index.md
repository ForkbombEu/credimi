---
title: Execute Pipelines via CI/CD 
description: Connect your CI/CD to credimi.io to run pipelines automatically and collect results.
---

# CI/CD

Setup your Github Actions (or other CI/CD), upon opening a PR (or similar) to automatically: 
- Upload an app installer, or an issuer/verifier temp url to credimi.io
- Start a Pipeline automatically
- Get results from a POST/GET 

## Integration via Github Actions

Integrate Github Actions to to automaticall run Credimi Pipelines from a pull request.

See docs: [credimi-action](https://github.com/forkbombeu/credimi-test-action)