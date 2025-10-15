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


### 1) OpenID4VCI — Get Credential Offer (POST)

Use this when the Issuer does **not** expose a static `.well-known` and generates a new session for each credential offer.

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

### 2) OpenID4VCI — Get Credential Offer (GET)

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

### 3) OpenID4VCI — Decode QR (PNG)

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

### 4) OpenID4VP — Get Presentation Request (POST)

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
