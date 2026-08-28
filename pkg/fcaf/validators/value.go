// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"encoding/base64"
	"fmt"
	"math"
	"regexp"
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

func validateRFC3339UTCDateTime(text string) error {
	if len(text) != 20 || !strings.HasSuffix(text, "Z") {
		return fmt.Errorf("value must use YYYY-MM-DDThh:mm:ssZ format")
	}
	if _, err := time.Parse("2006-01-02T15:04:05Z", text); err != nil {
		return fmt.Errorf("value is not a valid UTC date-time")
	}
	return nil
}

func isPIDCountryCode(value string) bool {
	if _, ok := iso3166Alpha2[value]; ok {
		return true
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
