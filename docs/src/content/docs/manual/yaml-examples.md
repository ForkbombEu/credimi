---
title: "YAML: Dynamic generation QR codes and Wallet Actions"
description: ""
---

The Credimi Hub dynamically generates QR codes for both Credential Offers (OpenID4VCI) and Presentation Requests (OpenID4VP). 

These QR codes are powered by StepCI recipes and exposed in the Hub under Credentials and Use Case Verifications, so end users and developers can try manual interoperability flows. The same StepCI code is used to set up end-to-end Wallet-to-Issuer/Verifier automated checks.

### Use cases

* **OpenID4VCI**: some issuers don’t publish a static `.well-known`. They generate a unique session ID for each credential offer, or show the offer only as a **QR PNG**. These recipes show how to handle both cases.

* **OpenID4VP**: verifier flows always use a session ID to generate a **presentation request**. The recipes show how to capture and reuse those.


### What is StepCI?

[StepCI](https://stepci.com/) is an **open-source API testing tool**. Tests are written in **YAML** and describe step‑by‑step HTTP calls, captures, and assertions. Credimi embeds StepCI directly into the **web interface**, so you can run these flows without installing anything. For more details, see the [StepCI docs](https://docs.stepci.com).

In Credimi we use StepCI to:

* Generate and capture **Credential Offers** and **Presentation Requests**
* Build **deeplinks** from those responses
* Pass the deeplinks to **Maestro** for mobile wallet automation
* Publish the same flows as **QR codes in the Hub** for **manual interoperability testing**

:::note
Always use:

	env:  
		host:  (url | mandatory )
		body:  (request body | optional )
 
as the examples below in the StepCI configuration, to setup the base url (and the body, if needed) of the service you're calling. This will help further automation at a later point.
:::


---
## OpenID4VCI examples


### OpenID4VCI — Get Credential Offer (POST)

Use this when the Issuer does **not** expose a static `.well-known` and generates a new session for each credential offer.

This example integrates with the [https://labs-openid-interop.vididentity.net/](https://labs-openid-interop.vididentity.net/) issuer.

```yaml
version: "1.0"
name: "VID Identity – issuance-pre-auth"

env:
  host: https://labs-openid-interop.vididentity.net/api/issuance-pre-auth

tests:
  VID-Identity: 
    steps:
      - name: "Create pre-auth issuance"
        http:
          method: POST
          url: ${{ env.host }}
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
     
```

---

### OpenID4VCI — Get Credential Offer (GET)

Use this when the Issuer provides a direct `request_uri` to fetch the offer. 

This example integrates with the [https://issuer.procivis.pensiondemo.findy.fi/](https://issuer.procivis.pensiondemo.findy.fi/) issuer.


```yaml
version: "1.0"
name: "Findynet/Procivis"
env:
  host: https://issuer.procivis.pensiondemo.findy.fi
tests:
  procivis-get-credential:
    steps:
      - name: "Get rehabilitation pension credential"
        http:
          method: GET
          url: ${{ env.host }}/pensioncredential-rehabilitation.json
          captures:
           deeplink:
             body: true
```

---

### OpenID4VCI — Decode QR (PNG)

Some issuers show a **QR** with the offer. Use StepCI with a QR decoder service to extract the deeplink. This is the most tricky case as (currently) we **need to use a 3rd party REST API to read the content of the QR**.

This example read the QR from: [https://ewc.pre.vc-dts.sicpa.com/demo/fakephotoid](https://ewc.pre.vc-dts.sicpa.com/demo/fakephotoid)

```yaml
version: "1.1"
name: Sicpa Test Issuer

env:
  host: https://ewc.pre.vc-dts.sicpa.com/api/fetchIssuanceQrCode?attributes[surname]=Matkalainen&attributes[given_name]=Hannah

tests:
  get-deeplink:
    steps:
      - name: get deeplink code
        http:
          url: ${{ env.host }}
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

### OpenID4VCI — read deeplink from html body

Some issuers show the in the body of an HTML page. You can navigate the DOM using *deeplink.xpath* to find the element you're looking for.


```yaml
version: "1.1"
name: Captures
env:
  host: https://issuer-backend.eudiw.dev/issuer/credentialsOffer/generate
  body: "credentialIds=eu.europa.ec.eudi.pid_vc_sd_jwt&credentialsOfferUri=openid-credential-offer%3A%2F%2F"
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
          check:
            status: /^20/
          captures:
            deeplink:
              xpath: /html/body/main/div/div[5]/div[2]/div/code
```
---

## OpenID4VP examples

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

### OpenID4VP — Get Presentation Request (POST)

Use this to create a **presentation request** (often session‑based) and capture the `openid4vp://...` deeplink.

The example below integrates with the "Rent a car" verification on [https://funke.animo.id](https://funke.animo.id).

```yaml
version: "1.0"
name: "VID Identity – issuance-pre-auth"

env:
  host: "https://labs-openid-interop.vididentity.net"

tests:
  VID-Identity–issuance-pre-auth:
    steps:
      - name: "Create pre-auth issuance"
        http:
          method: POST
          url: ${{ env.host }}/api/presentations
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



---

# Wallet actions

These YAML snippets describe **[Maestro](https://maestro.dev/) flows** that run on a mobile wallet. They consume the deeplinks captured by StepCI and automate user interactions (open app, accept, verify). 

The  snippets can be created with **[Maestro Studio](https://maestro.dev/#maestro-studio)** which you can download and install on Windows/Linux/Mac, you'll also need Android Studio and/or Xcode to run it.



## 📱 Wallet Actions (Maestro)

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

---

👉 Use these as templates. Replace placeholders with your issuer/verifier endpoints and JSONPaths.
