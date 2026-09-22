package onlyoffice

import (
	"fmt"
	"strconv"
	"strings"
)

// addressCategoryCodes maps ASC.CRM.Core.AddressCategory names to their numeric
// codes. The OO API expects the code, the UI/docs use the label.
var addressCategoryCodes = map[string]int{
	"home":    0,
	"postal":  1,
	"office":  2,
	"billing": 3,
	"other":   4,
	"work":    5,
}

// AddressCategoryCode returns the numeric code for an AddressCategory label
// (Home|Postal|Office|Billing|Other|Work) or a numeric string. Unknown/empty
// labels fall back to Billing, the category `oo companies create` used.
func AddressCategoryCode(category string) int {
	s := strings.ToLower(strings.TrimSpace(category))
	if s == "" {
		return addressCategoryCodes["billing"]
	}
	if n, err := strconv.Atoi(s); err == nil {
		if n >= 0 && n <= 5 {
			return n
		}
		return addressCategoryCodes["billing"]
	}
	if n, ok := addressCategoryCodes[s]; ok {
		return n
	}
	return addressCategoryCodes["billing"]
}

// ContactAddresses returns the postal address rows of a contact map.
func ContactAddresses(contact map[string]any) []map[string]any {
	if rows, ok := contact["addresses"].([]any); ok {
		return mapsFromAnySlice(rows)
	}
	if rows, ok := contact["addresses"].([]map[string]any); ok {
		return rows
	}
	return nil
}

// HasContactAddress reports whether a contact already has the given postal
// address. street+city+zip+category identify it; comparison is normalized.
func HasContactAddress(contact map[string]any, street, city, zip, category string) bool {
	wantStreet, wantCity, wantZip := normalizeAddressPart(street), normalizeAddressPart(city), normalizeAddressPart(zip)
	wantCat := AddressCategoryCode(category)
	for _, row := range ContactAddresses(contact) {
		if normalizeAddressPart(fmt.Sprint(row["street"])) != wantStreet {
			continue
		}
		if normalizeAddressPart(fmt.Sprint(row["city"])) != wantCity {
			continue
		}
		if normalizeAddressPart(fmt.Sprint(row["zip"])) != wantZip {
			continue
		}
		if int(anyToFloat(row["category"])) != wantCat {
			continue
		}
		return true
	}
	return false
}

func normalizeAddressPart(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.Join(strings.Fields(s), " ")
}

func anyToFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(n), 64)
		return f
	default:
		return 0
	}
}
