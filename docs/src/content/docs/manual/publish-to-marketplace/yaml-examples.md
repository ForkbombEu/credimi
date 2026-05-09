---
title: StepCI and Maestro scripting examples
description: Exampes of StepCI and Maestro recipes so credential, verification and integration with other services.
---

[StepCI](https://stepci.com) is the integration layer used by Credimi to interact with external Issuers and Verifiers.


# YAML: Dynamic generation QR codes and Wallet Actions

The Credimi Marketplace dynamically generates QR codes for both Credential Offers (OpenID4VCI) and Presentation Requests (OpenID4VP). 

These QR codes are powered by StepCI recipes and exposed in the Marketplace (Credentials and Use Case Verifications pages), so end-users (and developers) can try manual interoperability flows. The same StepCI code is used to setup end-to-end automatice Wallet-to-Issuer/Verifier automated checks.

### Use cases

* **OpenID4VCI**: some issuers don’t publish a static `.well-known`. They generate a unique session ID for each credential offer, or show the offer only as a **QR PNG**. These recipes show how to handle both cases.

* **OpenID4VP**: verifier flows always use a session ID to generate a **presentation request**. The recipes show how to capture and reuse those.


### What is StepCI?

[StepCI](https://stepci.com/) is an **open-source API testing tool**. Tests are written in **YAML** and describe step‑by‑step HTTP calls, captures, and assertions. Credimi embeds StepCI directly into the **web interface**, so you can run these flows without installing anything. For more details, see the [StepCI docs](https://stepci.com/docs).

In Credimi we use StepCI to:

* Generate and capture **Credential Offers** and **Presentation Requests**
* Build **deeplinks** from those responses
* Pass the deeplinks to **Maestro** for mobile wallet automation
* Publish the same flows as **QR codes in the Marketplace** for **manual interoperability testing**



---
## OpenID4VCI examples


Use this [StepCI] (https://stepci.com/) when the Issuer does **not** expose a static `.well-known` AND generates a session ID for each credential offer.



### OpenID4VCI — Get Credential Offer (POST) - simple



This example integrates with the [https://labs-openid-interop.vididentity.net/](https://labs-openid-interop.vididentity.net/) issuer.

```yaml
version: "1.0"
name: "VID Identity – issuance-pre-auth"

env:
  base_url: "https://labs-openid-interop.vididentity.net"

tests:
  VID-Identity: 
    steps:
      - name: "Create pre-auth issuance"
        http:
          method: POST
          url: ${{ env.base_url }}/api/issuance-pre-auth
          headers:
            accept: "application/json, text/plain, */*"
          json:
            credentialTypeId: "33095f2f-6f80-4168-8301-abc815848aef"
            issuerDid: "did:ebsi:zpD3Qp8h4psvdgnTGMX6hfE"
            credentialSubject:
              name: "Bianca"
              age: 30
              surname: "Castafiori"
            oid4vciVersion: "Draft13"
            userPin: 6831
          captures:
            deeplink:
              jsonpath: $.rawCredentialOffer
            qr:
              jsonpath: $.qrBase64
            sessionId:
              jsonpath: $.sessionId
```

---

### OpenID4VCI — Get Credential Offer (POST) - extract deepling from html

This integrates the [EUDIW Issuer](https://issuer.eudiw.dev/) which returns the deeplink inside an HTML page

```yaml
version: "1.1"
name: Captures
env:
  host: https://issuer-backend.eudiw.dev/issuer/credentialsOffer/generate
  body: "credentialIds=eu.europa.ec.eudi.pid_vc_sd_jwt&credentialsOfferUri=haip-vci%3A%2F%2F"
tests:
  example:
    steps:
      - name: Post the post
        http:
          url: ${{env.host}}
          method: POST
          headers:
            Content-Type: application/x-www-form-urlencoded
          body: ${{env.body}}
          captures:
            deeplink:
              xpath: /html/body/main/div/div[5]/div[2]/div/code
```




----

### OpenID4VCI — Get Credential Offer (GET)

Use this when the Issuer provides a direct `request_uri` to fetch the offer. 

This example integrates with the [https://issuer.procivis.pensiondemo.findy.fi/](https://issuer.procivis.pensiondemo.findy.fi/) issuer.


```yaml
version: "1.0"
name: "Findynet/Procivis"
env:
  base_url: https://issuer.procivis.pensiondemo.findy.fi
tests:
  procivis-get-credential:
    steps:
      - name: "Get rehabilitation pension credential"
        http:
          method: GET
          url: ${{ env.base_url }}/pensioncredential-rehabilitation.json
          captures:
           deeplink:
             jsonpath: $
```

---

### OpenID4VCI — extract deeplink from QR.png

Some issuers show a **QR** with the offer. Use StepCI with a QR decoder service to extract the deeplink. This is the most tricky case as (currently) we **need to use a 3rd party REST API to read the content of the QR**.

This example read the QR from: [https://ewc.pre.vc-dts.sicpa.com/demo/fakephotoid](https://ewc.pre.vc-dts.sicpa.com/demo/fakephotoid)

```yaml
version: "1.1"
name: "Sicpa Test Issuer"
tests:
  get-deeplink:
    steps:
      - name: get deeplink code
        http:
          url: https://ewc.pre.vc-dts.sicpa.com/api/fetchIssuanceQrCode?attributes[surname]=Matkalainen&attributes[given_name]=Hannah
          method: GET
          captures:
            deeplink:
              jsonpath: $.qr
      - name: parse
        http:
          url: https://aisenseapi.com/services/v1/qrcode_decode
          method: POST
          headers:
            Accept-Encoding: identity
          json:
            payload: "${{captures.deeplink | slice: 22}}"
          captures:
            deeplink:
              jsonpath: $.qrcode_content
```

---

## OpenID4VP examples

### OpenID4VP — Get Presentation Request (POST)

Use this to integrate [EUDIW Verifier](https://verifier.eudiw.dev/home) 


```yaml
version: "1.0"
name: Eudiw web verifier
env:
  host: https://verifier-backend.eudiw.dev/ui/presentations
  vct_values: urn:eudi:pid:1
tests:
  get-deeplink:
    steps:
      - name: Presentations
        http:
          method: POST
          url: ${{ env.host }}
          headers:
            Content-Type: application/json
          json:
            dcql_query:
              credentials:
                - id: query_0
                  format: dc+sd-jwt
                  meta:
                    vct_values:
                      - ${{ env.vct_values }}
                  claims:
                    - path:
                      - family_name
                    - path:
                      - given_name
                    - path:
                      - birthdate
                    - path:
                      - birth_family_name
                    - path:
                      - birth_given_name
                    - path:
                      - place_of_birth
                      - locality
                    - path:
                      - address
                      - formatted
                    - path:
                      - address
                      - country
                    - path:
                      - address
                      - region
                    - path:
                      - address
                      - locality
                    - path:
                      - address
                      - postal_code
                    - path:
                      - address
                      - street_address
                    - path:
                      - address
                      - house_number
                    - path:
                      - sex
                    - path:
                      - nationalities
                      - null
                    - path:
                      - date_of_issuance
                    - path:
                      - date_of_expiry
                    - path:
                      - issuing_authority
                    - path:
                      - document_number
                    - path:
                      - personal_administrative_number
                    - path:
                      - issuing_country
                    - path:
                      - issuing_jurisdiction
                    - path:
                      - picture
                    - path:
                      - email
                    - path:
                      - phone_number
                    - path:
                      - trust_anchor
            nonce: ${{ string.uuid | fake }}
            request_uri_method: post
          captures:
            trasaction_id:
              jsonpath: $.transaction_id
            client_id:
              jsonpath: $.client_id
            request_uri:
              jsonpath: $.request_uri
            request_uri_method:
              jsonpath: $.request_uri_method
      - name: Transform capture
        plugin:
          id: capture-plugin
          params:
            values:
              deeplink: haip-vp://?client_id=${{captures.client_id | url_encode }}&request_uri=${{captures.request_uri | url_encode }}&request_uri_method=${{captures.request_uri_method | url_encode }}
```



### OpenID4VP — Get Presentation Request (POST)

Use this to create a **presentation request** (often session‑based) and capture the `openid4vp://...` deeplink.

The example below integrates with the verification on [https://labs-openid-interop.vididentity.net/](https://labs-openid-interop.vididentity.net/).

```yaml
version: "1.0"
name: "VID Identity – issuance-pre-auth"

env:
  base_url: "https://labs-openid-interop.vididentity.net"

tests:
  VID-Identity–issuance-pre-auth:
    steps:
      - name: "Create pre-auth issuance"
        http:
          method: POST
          url: ${{ env.base_url }}/api/presentations
          headers:
            accept: "application/json, text/plain, */*"
          json:
            scope: "SDJWTCredential"
          captures:
            deeplink:
              jsonpath: $.rawOpenid4vp
            qr:
              jsonpath: $.qrBase64
            sessionId:
              jsonpath: $.sessionId
```

### 5) OpenID4VP — Get Presentation Request (POST)

Use this to create a **presentation request** (often session‑based) and capture the `openid4vp://...` deeplink.

The example below integrates with the "Rent a car" verification on [https://funke.animo.id](https://funke.animo.id).

```yaml
version: "1.0"

name: "Funke Animo - Government ID verification"

env:
  base_url: "https://funke.animo.id"

tests:
  Animo-rent-a-car:
    steps:
      - name: "Create pre-auth issuance"
        http:
          method: POST
          url: ${{ env.base_url }}/api/requests/create
          headers:
            accept: "application/json, text/plain, */*"
          json:
            presentationDefinitionId: 019368ed-3787-7669-b7f4-8c012238e90d__3
            requestScheme: 'openid4vp://'
            responseMode: direct_post.jwt
            requestSignerType: x5c
            transactionAuthorizationType: none
            version: v1.draft24
            queryLanguage: dcql          
          captures:
            deeplink:
              jsonpath: $.authorizationRequestUri
```



---

# Wallet actions

These YAML snippets describe **[Maestro](https://maestro.dev/) flows** that run on a mobile wallet. They consume the deeplinks captured by StepCI and automate user interactions (open app, accept, verify). 

The  snippets can be created with **[Maestro Studio](https://maestro.dev/#maestro-studio)** which you can download and install on Windows/Linux/Mac, you'll also need Android Studio and/or Xcode to run it.



## 📱 Wallet Actions (Maestro)

All the Maestro actions are visible in the Marketplace under the **Actions** section of each **Wallet**, see: https://credimi.io/marketplace 




### Install from Marketplace 

Use this if you don't have the app installer

- Android

```yaml
appId: com.android.vending
---
- openLink: "market://details?id=id.paradym.wallet"
- assertVisible: Install
- tapOn: Install
- tapOn: Open
```

- iOS: 

```yaml
coming soon
```

### onBoarding 

Here typically you will entere a pin code. If the app requires biometrics, that can be managed too (ask us).  

```yaml
appId: eu.europa.ec.euidi
---
- launchApp:
    clearState: true
- extendedWaitUntil:
    visible: "1" 
- tapOn: 1
- tapOn: 2
- tapOn: 3
- tapOn: 4
- tapOn: 5
- tapOn: 6
# - runFlow:
#     when:
#       visible: 'NEXT'
#     commands:
#         - tapOn: 'NEXT'
- extendedWaitUntil:
    visible: "NEXT"
    timeout: 15000
- tapOn:
    text: "NEXT"
    retryTapIfNoChange: true
- extendedWaitUntil:
    visible: "1" 
- tapOn: 1
- tapOn: 2
- tapOn: 3
- tapOn: 4
- tapOn: 5
- tapOn: 6
# - runFlow:
#     when:
#       visible: 'CONFIRM'
#     commands:
#         - tapOn: 'CONFIRM'
- extendedWaitUntil:
    visible: "CONFIRM"
    timeout: 10000

- tapOn:
    text: "CONFIRM"
    retryTapIfNoChange: true
```



### getCredential

Here you can setup 2 types of actions: 

- Generic getCredential: works with issuance flows which bypass authentication (e.g. scan the qr and the get the credential), useful for interop and generic testing 
- getCredential (specifi): these actions include an authentication flow, may require opening a web page or passing other data. 

:::caution
If your isuser has a static .well-known (and the credential_offers have no session IDs), you can import the .well-known straight from this page. A list of Credentials will be auto-populated (you can later edit/delete each of them separately)
:::

See example of a specific getCredential action: 

```yaml

appId: eu.europa.ec.euidi
--- 
- stopApp
- launchApp
# the "openLink: ${deeplink}" opens a deeplink produced in the previous automation step.
- openLink: ${deeplink}
# use a link hardcoded below, for debugging as well as if your credential_offer contains a session ID 
https://credimi.io/api/credential/deeplink?id=forkbomb-bv-andrea/misc-issuer-integration-demo/eudiw-pid-pid-vc-sd-jwt-haip-vci&redirect=true
# - extendedWaitUntil:
#    timeout: 10000
#    visible: "Welcome back" 
# - inputText: "123456"
- tapOn: 1
- tapOn: 2
- tapOn: 3
- tapOn: 4
- tapOn: 5
- tapOn: 6
- extendedWaitUntil:
   visible: "Add" 
   timeout: 30000   
- tapOn: Add
- tapOn:
    point: 6%,61%
- extendedWaitUntil:
    visible: "FormEU" 
    timeout: 5000   
- tapOn: FormEU
- scrollUntilVisible:
    element: "Submit"
- tapOn: Submit
- tapOn:
    point: 50%,50% 
- extendedWaitUntil:
    visible: 
      id: selectCountryForm
    timeout: 5000   
- tapOn:
    below: "Mandatory Information"
    above: "Family Name"
    childOf:                                  
      id: selectCountryForm
- tapOn:
    id: date_picker_header_year
# WORKING bug sometimes it doesn't find the 2005
- scrollUntilVisible:
    element: "2005|2004|2003|2002|2001"
    direction: UP
    speed: 10
- tapOn: "2005|2004|2003|2002|2001"
- tapOn: Set
- scrollUntilVisible:
    element: "Family Name"
    speed: 5
    centerElement: true
- tapOn:
    below: "Family Name"
    childOf:                                  
      id: selectCountryForm
    traits: text 
- inputText: "F"
- hideKeyboard
- scrollUntilVisible:
    element: "Given Name"
    speed: 5
    centerElement: true
- tapOn:
    below: "Given Name"
    childOf:                                  
      id: selectCountryForm
- inputText: "N"
# - hideKeyboard
- scrollUntilVisible:
    element: "Country Code"
    speed: 10
    centerElement: true
- tapOn:
    below: "Country Code"
    traits: text
    childOf:                                   
      id: selectCountryForm
- inputText: "IT"
- scrollUntilVisible:
    element: "Country"
    speed: 10
    centerElement: true
- tapOn:
    below: "Country"
    childOf:                                   
      id: selectCountryForm
- inputText: "IT"
# - hideKeyboard
- scrollUntilVisible:
    element: "Confirm"
- tapOn: "Confirm"
- assertNotVisible: Oups! Something went wrong
- extendedWaitUntil:
    visible: "Review & Send" 
    timeout: 10000   
- scrollUntilVisible:
    element: "Authorize"
- tapOn: "Authorize"
- assertNotVisible: Oups! Something went wrong
- extendedWaitUntil:
    visible: "Close" 
    timeout: 10000   
- assertNotVisible: Oups! Something went wrong
- tapOn: Close

``` 
 
<!--
StepCI captures the deeplink. Maestro drives the wallet app, this works with the [DIDroom Wallet](https://didroom.com/apps/)

```yaml
appId: com.didroom.wallet
---
- launchApp:
    clearState: true
- tapOn: SKIP
- tapOn: LOGIN
- tapOn:
    below: Email
- inputText: tess@tes.com
- hideKeyboard
- tapOn:
    below: password
- inputText: testtest
- hideKeyboard
- tapOn: NEXT next
- scroll
- scroll
- tapOn:
    below: insert your passphrase
- inputText: chronic property inject opera glow client horse notable grape build engine damage
- tapOn: LOGIN
- tapOn: GET CREDENTIALS
- tapOn: "dc+sd-jwt Voucher Credential dc+sd-jwt Voucher Credential test ci"
- tapOn: CONTINUE
- tapOn:
    text: Voucher
    index: 1
- inputText: ten
- hideKeyboard
- tapOn: AUTHENTICATE
- tapOn: Wallet

```
-->


---

👉 Use these as templates. Replace placeholders with your issuer/verifier endpoints and JSONPaths.
