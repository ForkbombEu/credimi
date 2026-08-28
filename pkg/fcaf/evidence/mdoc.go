// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package evidence

import (
	"encoding/base64"
	"fmt"
	"reflect"

	"github.com/fxamacker/cbor/v2"
)

type MDocPresentation struct {
	Raw              []byte                            `json:"raw"`
	Documents        []MDocDocument                    `json:"documents"`
	DocumentErrors   []map[string]int64                `json:"document_errors,omitempty"`
	Status           uint64                            `json:"status"`
	DecodedTopLevel  map[string]any                    `json:"decoded_top_level,omitempty"`
	SelectedDocument int                               `json:"selected_document"`
	Namespaces       map[string]map[string]MDocElement `json:"namespaces,omitempty"`
}

type MDocDocument struct {
	DocType         string                            `json:"doc_type"`
	DigestAlgorithm string                            `json:"digest_algorithm,omitempty"`
	Namespaces      map[string]map[string]MDocElement `json:"namespaces"`
	Errors          map[string]map[string]int64       `json:"errors,omitempty"`
}

type MDocElement struct {
	Identifier       string  `json:"identifier"`
	Value            any     `json:"value"`
	MajorType        uint8   `json:"major_type"`
	ContentMajorType uint8   `json:"content_major_type"`
	Tag              *uint64 `json:"tag,omitempty"`
	Raw              []byte  `json:"raw"`
}

type rawDeviceResponse struct {
	Version        string             `cbor:"version"`
	Documents      []rawDocument      `cbor:"documents"`
	DocumentErrors []map[string]int64 `cbor:"documentErrors"`
	Status         uint64             `cbor:"status"`
}

type rawDocument struct {
	DocType      string                      `cbor:"docType"`
	IssuerSigned rawIssuerSigned             `cbor:"issuerSigned"`
	Errors       map[string]map[string]int64 `cbor:"errors"`
}

type rawIssuerSigned struct {
	NameSpaces map[string][]cbor.RawMessage `cbor:"nameSpaces"`
	IssuerAuth cbor.RawMessage              `cbor:"issuerAuth"`
}

type rawIssuerSignedItem struct {
	ElementIdentifier string          `cbor:"elementIdentifier"`
	ElementValue      cbor.RawMessage `cbor:"elementValue"`
}

var mdocDecMode = mustMDocDecMode()

func mustMDocDecMode() cbor.DecMode {
	mode, err := cbor.DecOptions{
		DupMapKey:            cbor.DupMapKeyEnforcedAPF,
		MaxNestedLevels:      32,
		MaxArrayElements:     4096,
		MaxMapPairs:          4096,
		IndefLength:          cbor.IndefLengthAllowed,
		TagsMd:               cbor.TagsAllowed,
		IntDec:               cbor.IntDecConvertNone,
		DefaultMapType:       reflect.TypeFor[map[string]any](),
		UTF8:                 cbor.UTF8RejectInvalid,
		UnrecognizedTagToAny: cbor.UnrecognizedTagContentToAny,
		TimeTagToAny:         cbor.TimeTagToRFC3339Nano,
	}.DecMode()
	if err != nil {
		panic(err)
	}
	return mode
}

func ParseMDocPresentation(encoded any) (*MDocPresentation, error) {
	raw, err := decodeMDocBytes(encoded)
	if err != nil {
		return nil, err
	}

	var response rawDeviceResponse
	if err := mdocDecMode.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("decode mdoc DeviceResponse: %w", err)
	}
	if len(response.Documents) == 0 {
		return nil, fmt.Errorf("mdoc DeviceResponse contains no documents")
	}

	documents := make([]MDocDocument, 0, len(response.Documents))
	for index, document := range response.Documents {
		if document.DocType == "" {
			return nil, fmt.Errorf("mdoc document %d has no docType", index)
		}
		namespaces, err := parseMDocNamespaces(document.IssuerSigned.NameSpaces)
		if err != nil {
			return nil, fmt.Errorf("decode mdoc document %d: %w", index, err)
		}
		digestAlgorithm, err := parseMDocDigestAlgorithm(document.IssuerSigned.IssuerAuth)
		if err != nil {
			return nil, fmt.Errorf("decode mdoc document %d issuerAuth: %w", index, err)
		}
		documents = append(documents, MDocDocument{
			DocType:         document.DocType,
			DigestAlgorithm: digestAlgorithm,
			Namespaces:      namespaces,
			Errors:          document.Errors,
		})
	}

	var topLevel map[string]any
	_ = mdocDecMode.Unmarshal(raw, &topLevel)
	return &MDocPresentation{
		Raw:              raw,
		Documents:        documents,
		DocumentErrors:   response.DocumentErrors,
		Status:           response.Status,
		DecodedTopLevel:  topLevel,
		SelectedDocument: 0,
		Namespaces:       documents[0].Namespaces,
	}, nil
}

func parseMDocDigestAlgorithm(raw cbor.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	sign1Raw := raw
	var tagged cbor.RawTag
	if err := mdocDecMode.Unmarshal(raw, &tagged); err == nil && tagged.Number == 18 {
		sign1Raw = tagged.Content
	}
	var sign1 []cbor.RawMessage
	if err := mdocDecMode.Unmarshal(sign1Raw, &sign1); err != nil {
		return "", fmt.Errorf("decode COSE_Sign1: %w", err)
	}
	if len(sign1) != 4 {
		return "", fmt.Errorf("COSE_Sign1 has %d entries, expected 4", len(sign1))
	}
	var payload []byte
	if err := mdocDecMode.Unmarshal(sign1[2], &payload); err != nil {
		return "", fmt.Errorf("decode COSE_Sign1 payload: %w", err)
	}
	var mso struct {
		DigestAlgorithm string `cbor:"digestAlgorithm"`
	}
	msoBytes := payload
	var wrapped []byte
	if err := mdocDecMode.Unmarshal(payload, &wrapped); err == nil {
		msoBytes = wrapped
	}
	if err := mdocDecMode.Unmarshal(msoBytes, &mso); err != nil {
		return "", fmt.Errorf("decode Mobile Security Object: %w", err)
	}
	if mso.DigestAlgorithm == "" {
		return "", fmt.Errorf("mobile security object has no digestAlgorithm")
	}
	return mso.DigestAlgorithm, nil
}

func (p *MDocPresentation) Document(docType string) (*MDocDocument, bool) {
	if p == nil {
		return nil, false
	}
	for index := range p.Documents {
		if p.Documents[index].DocType == docType {
			return &p.Documents[index], true
		}
	}
	return nil, false
}

func (p *MDocPresentation) Element(namespace string, identifier string) (MDocElement, bool) {
	if p == nil {
		return MDocElement{}, false
	}
	elements, ok := p.Namespaces[namespace]
	if !ok {
		return MDocElement{}, false
	}
	element, ok := elements[identifier]
	return element, ok
}

func parseMDocNamespaces(
	rawNamespaces map[string][]cbor.RawMessage,
) (map[string]map[string]MDocElement, error) {
	namespaces := make(map[string]map[string]MDocElement, len(rawNamespaces))
	for namespace, rawItems := range rawNamespaces {
		elements := make(map[string]MDocElement, len(rawItems))
		for _, rawItem := range rawItems {
			itemBytes, err := unwrapIssuerSignedItem(rawItem)
			if err != nil {
				return nil, fmt.Errorf("namespace %q: %w", namespace, err)
			}
			var item rawIssuerSignedItem
			if err := mdocDecMode.Unmarshal(itemBytes, &item); err != nil {
				return nil, fmt.Errorf("namespace %q issuer-signed item: %w", namespace, err)
			}
			if item.ElementIdentifier == "" {
				return nil, fmt.Errorf(
					"namespace %q contains item without elementIdentifier",
					namespace,
				)
			}
			if _, duplicate := elements[item.ElementIdentifier]; duplicate {
				return nil, fmt.Errorf(
					"namespace %q contains duplicate element %q",
					namespace,
					item.ElementIdentifier,
				)
			}
			element, err := decodeMDocElement(item.ElementIdentifier, item.ElementValue)
			if err != nil {
				return nil, fmt.Errorf(
					"namespace %q element %q: %w",
					namespace,
					item.ElementIdentifier,
					err,
				)
			}
			elements[item.ElementIdentifier] = element
		}
		namespaces[namespace] = elements
	}
	return namespaces, nil
}

func unwrapIssuerSignedItem(raw cbor.RawMessage) ([]byte, error) {
	var tagged cbor.RawTag
	if err := mdocDecMode.Unmarshal(raw, &tagged); err != nil {
		return nil, fmt.Errorf("issuer-signed item is not tagged CBOR: %w", err)
	}
	if tagged.Number != 24 {
		return nil, fmt.Errorf("issuer-signed item uses tag %d, expected 24", tagged.Number)
	}
	var encoded []byte
	if err := mdocDecMode.Unmarshal(tagged.Content, &encoded); err != nil {
		return nil, fmt.Errorf("issuer-signed item tag 24 content is not a byte string: %w", err)
	}
	return encoded, nil
}

func decodeMDocElement(identifier string, raw cbor.RawMessage) (MDocElement, error) {
	if len(raw) == 0 {
		return MDocElement{}, fmt.Errorf("empty element value")
	}
	element := MDocElement{
		Identifier:       identifier,
		MajorType:        raw[0] >> 5,
		ContentMajorType: raw[0] >> 5,
		Raw:              append([]byte(nil), raw...),
	}
	if element.MajorType == 6 {
		var tagged cbor.RawTag
		if err := mdocDecMode.Unmarshal(raw, &tagged); err != nil {
			return MDocElement{}, fmt.Errorf("decode tagged element: %w", err)
		}
		element.Tag = &tagged.Number
		if len(tagged.Content) == 0 {
			return MDocElement{}, fmt.Errorf("tagged element has empty content")
		}
		element.ContentMajorType = tagged.Content[0] >> 5
		if err := mdocDecMode.Unmarshal(tagged.Content, &element.Value); err != nil {
			return MDocElement{}, fmt.Errorf("decode tagged element content: %w", err)
		}
		return element, nil
	}
	if err := mdocDecMode.Unmarshal(raw, &element.Value); err != nil {
		return MDocElement{}, fmt.Errorf("decode element value: %w", err)
	}
	return element, nil
}

func decodeMDocBytes(encoded any) ([]byte, error) {
	switch value := encoded.(type) {
	case []byte:
		return append([]byte(nil), value...), nil
	case string:
		for _, encoding := range []*base64.Encoding{
			base64.RawURLEncoding,
			base64.URLEncoding,
			base64.RawStdEncoding,
			base64.StdEncoding,
		} {
			decoded, err := encoding.DecodeString(value)
			if err == nil {
				return decoded, nil
			}
		}
		return nil, fmt.Errorf("mdoc value is not valid base64 or base64url")
	default:
		return nil, fmt.Errorf("mdoc value is %T, expected string or bytes", encoded)
	}
}
