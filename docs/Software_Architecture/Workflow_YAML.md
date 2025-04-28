<!--- encapsulate weird stuff  -->
<!--- <code v-pre>{{.}}</code>  -->

# Manage a workflow in Step-CI

## Intro

The following worklow is written in a YAML file, processed by step.ci (which allows to easy check the REST outputs against values) and executed by temporal.io.

In order to generate an url containing an intent (openid4vci://something), needed to produce a QR code to test a wallet according to the OpenID Foundation conformance checks, you need to execute the following REST API CALL to the site https://www.certification.openid.net

- POST to https://www.certification.openid.net/api/plan
- POST to https://www.certification.openid.net/api/runner
- GET to <code v-pre>https://www.certification.openid.net/api/runner/${{captures.id}}</code>

The flow of these calls is managed and tested via StepCI [https://stepci.com], so you need to create a YAML file like the example based on the StepCI documentation.
The first key "components" is needed to pass the bearer token as input from CLI since all API calls to https://www.certification.openid.net must be authenticated.
Then you need to describe the tests you want to test in the tests object. In our case we have only one flow to test which is the one that executes the three calls above with the right parameters to be able to get the QR code URL.
The flow in the example file we called example and inside it we list the steps to which each step corresponds to a call. For each step:

name can be anything
http must contain all the parameters necessary to execute the call
captures are the values contained in the response that are necessary to pass to the next step or as output of the entire flow
check contains the tests that are performed on the response in our case we test the response status and compare the body of the response against a jsonschema specified in the schema.

## STEPS:

1. https://www.certification.openid.net/api/plan

The http key contains the following keys (POST):

url
params: are the query parameters. Requires planName and variant as plain json (see explanation Matteo)
auth: takes the bearer token passed from CLI
header: specifies the type of body
json: must contain the form (see explanation Matteo) structured correctly as jso
from the response obtained in captures the id associated with the plan must be saved.

2. https://www.certification.openid.net/api/runner

The http key contains as keys (method POST)

url
params: Requires test, the name of the test, and plan which is the id from the captures from the previous response
auth: takes the bearer token passed from CLI
header: specifies the type of body
json: must contain the form (see explanation Matteo) structured correctly as json
from the response obtained in captures the id associated with the test module must be saved.
This POST returns, among other objects, the "testModulesID" to be used in the third query.

3. Step.ci extracts "testModulesID" from the output of the previous POST and stores it in "captures.id", which is passed as parameter in the third query:

<code v-pre>https://www.certification.openid.net/api/runner/${{captures.id}}</code>

From the body of the response obtained, "browser.urls[0]" must be stored, as it contains the "intent".

## Run the flow

To run this script in credimi we use a custom runner of StepCI [https://github.com/ForkbombEu/stepci-captured-runner] that allows you to print as output all the captures of the steps if the flow ends correctly, otherwise an error is returned.

In credimi this YAML is generated inside an "activity" in a "Temporal workflow" [https://github.com/ForkbombEu/credimi/blob/main/pkg/OpenID4VP/workflow/workflow.go] and executed by the runner in a subsequent "activity" [https://github.com/ForkbombEu/credimi/blob/main/pkg/OpenID4VP/workflow/activities.go], this code contains some of the YAML configuration as described above. The choice of "test Plan" and "test module" (currently only "oid4vp-id2-wallet-test-plan" and "oid4vp-id2-wallet-happy-flow-no-state" are supported) is passed as a parameter via GUI. Other configurable variable inputs are the "variant" (containing the internal parameters used by tests) and the "form" that are passed in a single JSON.

The temporal workflow can be started in two ways:

- from the frontend by clicking on start new check
- Via CLI: `credimi openid4vp-test -i input.json -u user@example.org`

