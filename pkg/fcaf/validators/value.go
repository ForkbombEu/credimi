// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"encoding/base64"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	fullDatePattern           = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
	internationalPhonePattern = regexp.MustCompile(`^\+[0-9]+$`)
	countrySubdivisionPattern = regexp.MustCompile(`^([A-Z]{2})-([A-Z0-9]{1,3})$`)
)

// ISO 3166-1 alpha-2 officially assigned codes. User-assigned codes are
// handled separately because the PID rulebook explicitly permits them.
var iso3166Alpha2 = codeSet(strings.Fields(`
AD AE AF AG AI AL AM AO AQ AR AS AT AU AW AX AZ
BA BB BD BE BF BG BH BI BJ BL BM BN BO BQ BR BS BT BV BW BY BZ
CA CC CD CF CG CH CI CK CL CM CN CO CR CU CV CW CX CY CZ
DE DJ DK DM DO DZ
EC EE EG EH ER ES ET
FI FJ FK FM FO FR
GA GB GD GE GF GG GH GI GL GM GN GP GQ GR GS GT GU GW GY
HK HM HN HR HT HU
ID IE IL IM IN IO IQ IR IS IT
JE JM JO JP
KE KG KH KI KM KN KP KR KW KY KZ
LA LB LC LI LK LR LS LT LU LV LY
MA MC MD ME MF MG MH MK ML MM MN MO MP MQ MR MS MT MU MV MW MX MY MZ
NA NC NE NF NG NI NL NO NP NR NU NZ
OM
PA PE PF PG PH PK PL PM PN PR PS PT PW PY
QA
RE RO RS RU RW
SA SB SC SD SE SG SH SI SJ SK SL SM SN SO SR SS ST SV SX SY SZ
TC TD TF TG TH TJ TK TL TM TN TO TR TT TV TW TZ
UA UG UM US UY UZ
VA VC VE VG VI VN VU
WF WS
YE YT
ZA ZM ZW
`))

func codeSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

// e164CountryCallingCodes holds the country calling codes assigned by ITU-T
// Recommendation E.164 (list of assigned country codes, Annex to ITU
// Operational Bulletin No. 1114 of 15 December 2016), including the shared and
// global-service codes of that list and +970 (Palestine), which later editions
// assign. Spare, reserved and unassigned codes are absent.
var e164CountryCallingCodes = codeSet(strings.Fields(`
1 7
20 27
30 31 32 33 34 36 39 40 41 43 44 45 46 47 48 49
60 61 62 63 64 65 66
81 82 84 86
90 91 92 93 94 95 98
211 212 213 216 218 220 221 222 223 224 225 226 227 228 229
230 231 232 233 234 235 236 237 238 239 240 241 242 243 244 245
246 247 248 249 250 251 252 253 254 255 256 257 258 260 261 262
263 264 265 266 267 268 269 290 291 297 298 299
350 351 352 353 354 355 356 357 358 359
370 371 372 373 374 375 376 377 378 379 380 381 382 383 385 386 387 388 389
420 421 423
500 501 502 503 504 505 506 507 508 509 51 52 53 54 55 56 57 58
590 591 592 593 594 595 596 597 598 599
670 672 673 674 675 676 677 678 679 680 681 682 683 685 686 687 688 689 690 691 692
800 808 850 852 853 855 856 870 878 880 881 882 883 886 888
960 961 962 963 964 965 966 967 968 970 971 972 973 974 975 976 977 979
991 992 993 994 995 996 998
`))

// hasAssignedCountryCallingCode reports whether digits starts with a country
// calling code of the ITU-T E.164 list and continues with at least one digit.
func hasAssignedCountryCallingCode(digits string) bool {
	for length := 1; length <= 3 && length < len(digits); length++ {
		if _, ok := e164CountryCallingCodes[digits[:length]]; ok {
			return true
		}
	}
	return false
}

// validateInternationalPhone applies the PID rules for an international phone
// number: the value starts with "+" followed by digits only, and those digits
// start with a country calling code assigned by ITU-T E.164.
func validateInternationalPhone(text string) error {
	if !internationalPhonePattern.MatchString(text) {
		return fmt.Errorf("value must start with + and contain digits only")
	}
	if !hasAssignedCountryCallingCode(strings.TrimPrefix(text, "+")) {
		return fmt.Errorf("value must start with an assigned ITU-T E.164 country calling code")
	}
	return nil
}

func resolveObjectPath(root any, path string) (any, bool) {
	if path == "" {
		return nil, false
	}
	current := root
	for _, segment := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[segment]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func requireUTF8String(value any) (string, error) {
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("value is %T, expected string", value)
	}
	if !utf8.ValidString(text) {
		return "", fmt.Errorf("value is not valid UTF-8")
	}
	return text, nil
}

func validateFullDate(text string) error {
	if !fullDatePattern.MatchString(text) {
		return fmt.Errorf("value must use YYYY-MM-DD format")
	}
	parsed, err := time.Parse("2006-01-02", text)
	if err != nil || parsed.Format("2006-01-02") != text {
		return fmt.Errorf("value is not a valid calendar date")
	}
	return nil
}

// rfc3339UTCDateTimePattern is the RFC 3339 §5.6 "date-time" ABNF restricted to
// the "Z" time-offset, with the "02" digit width of §5.6 for every field.
var rfc3339UTCDateTimePattern = regexp.MustCompile(
	`^([0-9]{4})-([0-9]{2})-([0-9]{2})T([0-9]{2}):([0-9]{2}):([0-9]{2})Z$`,
)

func validateRFC3339UTCDateTime(text string) error {
	fields := rfc3339UTCDateTimePattern.FindStringSubmatch(text)
	if fields == nil {
		return fmt.Errorf("value must use YYYY-MM-DDThh:mm:ssZ format")
	}
	date := fmt.Sprintf("%s-%s-%s", fields[1], fields[2], fields[3])
	if err := validateFullDate(date); err != nil {
		return fmt.Errorf("value is not a valid UTC date-time")
	}
	// The pattern guarantees decimal fields, so the conversions cannot fail.
	hour, _ := strconv.Atoi(fields[4])
	minute, _ := strconv.Atoi(fields[5])
	second, _ := strconv.Atoi(fields[6])
	// RFC 3339 §5.7 allows "60" for a leap second.
	if hour > 23 || minute > 59 || second > 60 {
		return fmt.Errorf("value is not a valid UTC date-time")
	}
	return nil
}

func isPIDCountryCode(value string) bool {
	if _, ok := iso3166Alpha2[value]; ok {
		return true
	}
	// The PID rulebook only permits the two-character user-assigned codes
	// AA, QM-QZ, XA-XZ and ZZ, so a longer value must not match by range.
	if len(value) != 2 {
		return false
	}
	if value == "AA" || value == "ZZ" {
		return true
	}
	return value >= "QM" && value <= "QZ" || value >= "XA" && value <= "XZ"
}

func validateCountrySubdivision(value string, country string) error {
	matches := countrySubdivisionPattern.FindStringSubmatch(value)
	if matches == nil {
		return fmt.Errorf("value must use an ISO 3166-2 country-subdivision shape")
	}
	if !isPIDCountryCode(matches[1]) {
		return fmt.Errorf(
			"subdivision country prefix %q is not an accepted country code",
			matches[1],
		)
	}
	if country != "" && matches[1] != country {
		return fmt.Errorf(
			"subdivision country prefix %q does not match issuing country %q",
			matches[1],
			country,
		)
	}
	subdivisions, listed := iso3166Subdivisions[matches[1]]
	if !listed {
		return fmt.Errorf("country %q lists no ISO 3166-2 subdivisions", matches[1])
	}
	if _, ok := subdivisions[matches[2]]; !ok {
		return fmt.Errorf(
			"subdivision code %q is not listed in ISO 3166-2 for %q",
			matches[2],
			matches[1],
		)
	}
	return nil
}

func integralNumber(value any) (int64, bool) {
	switch number := value.(type) {
	case int:
		return int64(number), true
	case int8:
		return int64(number), true
	case int16:
		return int64(number), true
	case int32:
		return int64(number), true
	case int64:
		return number, true
	case uint:
		if uint64(number) > math.MaxInt64 {
			return 0, false
		}
		return int64(number), true
	case uint8:
		return int64(number), true
	case uint16:
		return int64(number), true
	case uint32:
		return int64(number), true
	case uint64:
		if number > math.MaxInt64 {
			return 0, false
		}
		return int64(number), true
	case float32:
		float := float64(number)
		if math.Trunc(float) != float || float < math.MinInt64 || float > math.MaxInt64 {
			return 0, false
		}
		return int64(float), true
	case float64:
		if math.Trunc(number) != number || number < math.MinInt64 || number > math.MaxInt64 {
			return 0, false
		}
		return int64(number), true
	default:
		return 0, false
	}
}

func decodeJPEGDataURL(value string) ([]byte, error) {
	const prefix = "data:image/jpeg;base64,"
	if !strings.HasPrefix(value, prefix) {
		return nil, fmt.Errorf("value is not a JPEG base64 data URL")
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, prefix))
	if err != nil {
		return nil, fmt.Errorf("decode JPEG data URL: %w", err)
	}
	if !hasJPEGStart(decoded) {
		return nil, fmt.Errorf("decoded image does not start with JPEG marker FF D8")
	}
	return decoded, nil
}

func hasJPEGStart(value []byte) bool {
	return len(value) >= 2 && value[0] == 0xff && value[1] == 0xd8
}
