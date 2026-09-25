package embedder

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// TestoDocumento is the text the model sees for a document: the values of the
// fields, in order, one per line; the items of a list on one line, separated
// by commas. Without fields, every text field but the id, sorted by name.
func TestoDocumento(doc map[string]interface{}, campi []string) string {
	if len(campi) == 0 {
		for k, v := range doc {
			if k == "id" || k == "_id" || k == "_vector" {
				continue
			}
			switch v.(type) {
			case string, []interface{}, []string:
				campi = append(campi, k)
			}
		}
		sort.Strings(campi)
	}
	var righe []string
	for _, c := range campi {
		if t := strings.TrimSpace(valore(doc[c])); t != "" {
			righe = append(righe, t)
		}
	}
	return strings.Join(righe, "\n")
}

func valore(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case []string:
		return strings.Join(x, ", ")
	case []interface{}:
		parti := make([]string, 0, len(x))
		for _, e := range x {
			if s := strings.TrimSpace(valore(e)); s != "" {
				parti = append(parti, s)
			}
		}
		return strings.Join(parti, ", ")
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	}
	return fmt.Sprint(v)
}
