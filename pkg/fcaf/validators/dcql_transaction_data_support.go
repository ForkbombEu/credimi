// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

// transactionDataErrorMode reports whether mode is one of the modes that assert
// the Wallet's invalid_transaction_data rejection, which are the only modes
// that may consult NotApplicableWhenWalletSupportsTransactionData.
func transactionDataErrorMode(mode string) bool {
	return mode == "transaction_data_error" || mode == "transaction_data_error_required"
}

// transactionDataSupportVerdict reports the not_applicable verdict a definition
// asks for when its source applies only to a Wallet that does not support the
// transaction_data parameter.
//
// WS_RP_MS_ProtocolMessages__017 states that applicability in its own profile
// section ("Wallet does not support transaction_data") while prescribing an
// invalid_transaction_data rejection. Which side holds is a fact about the
// Wallet under test, so it is read from the Wallet's own captured evidence
// instead of being assumed: a Wallet that processed the transaction_data it
// was sent refutes the source's premise, and the rejection the source
// prescribes then has no subject. decided is false when the definition did not
// ask for this branch, or when the evidence does not prove the support, and
// the mode then decides with the rejection requirement it states.
func transactionDataSupportVerdict(params dcqlModeParams, root map[string]any) (Result, bool) {
	if !params.NotApplicableWhenWalletSupportsTransactionData {
		return Result{}, false
	}
	reason, supports := walletSupportsTransactionData(root)
	if !supports {
		return Result{}, false
	}
	return Result{Status: StatusNotApplicable, Message: reason}, true
}

// walletSupportsTransactionData reports whether the Wallet's own evidence shows
// it supports the transaction_data Authorization Request parameter.
//
// Two captured facts say so. checks.transaction_data_verified is the Capture
// verifier session's judgement, and it is true only for a presentation the
// session accepted whose transaction_data the Wallet bound through
// transaction_data_hashes (OpenID4VP Section 8.4). transaction_data_hashes is
// the member a Wallet returns in its Authorization Response for each
// transaction_data entry it processed, so a capture that recorded it proves
// the same support even before the verifier judged the response; it is looked
// for wherever the capture recorded the Wallet's response, because Capture
// nests it under the session's raw response records.
func walletSupportsTransactionData(root map[string]any) (string, bool) {
	checks, _ := normalizeJSONObject(root["checks"])
	if verified, ok := checks["transaction_data_verified"].(bool); ok && verified {
		return "the Capture verifier session records checks.transaction_data_verified, so the Wallet bound the transaction_data it was sent and the source's applicability (Wallet does not support transaction_data) does not hold", true
	}
	if _, found := findObjectKey(root, "transaction_data_hashes"); found {
		return "the Wallet's Authorization Response carries transaction_data_hashes, so it processed the transaction_data it was sent and the source's applicability (Wallet does not support transaction_data) does not hold", true
	}
	return "", false
}
