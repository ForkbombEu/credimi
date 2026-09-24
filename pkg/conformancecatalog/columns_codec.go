// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"encoding/json"
	"fmt"
)

// stringArray is a JSON string[] column for visible_in (columnSpec Kind).
type stringArray []string

func (s *stringArray) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*s = stringArray{}
		return nil
	case []byte:
		if len(v) == 0 {
			*s = stringArray{}
			return nil
		}
		return json.Unmarshal(v, (*[]string)(s))
	case string:
		if v == "" {
			*s = stringArray{}
			return nil
		}
		return json.Unmarshal([]byte(v), (*[]string)(s))
	default:
		return fmt.Errorf("stringArray: unsupported Scan type %T", value)
	}
}

func (s stringArray) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]string(s))
}

// memberArray is a JSON []SuiteMember column for suite members (columnSpec Kind).
type memberArray []SuiteMember

func (m *memberArray) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*m = memberArray{}
		return nil
	case []byte:
		if len(v) == 0 {
			*m = memberArray{}
			return nil
		}
		return json.Unmarshal(v, (*[]SuiteMember)(m))
	case string:
		if v == "" {
			*m = memberArray{}
			return nil
		}
		return json.Unmarshal([]byte(v), (*[]SuiteMember)(m))
	default:
		return fmt.Errorf("memberArray: unsupported Scan type %T", value)
	}
}

func (m memberArray) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]SuiteMember(m))
}
