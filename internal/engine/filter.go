package engine

import (
	"math"
	"strconv"
	"strings"
)

type Filter struct {
	Field    string
	Operator string
	Value    string
}

func ParseFilters(raw string) []Filter {
	if raw == "" {
		return nil
	}

	var filters []Filter
	var currentPart strings.Builder
	parenCount := 0

	for _, ch := range raw {
		switch ch {
		case '(':
			parenCount++
		case ')':
			parenCount--
		}

		if ch == ',' && parenCount == 0 {
			part := strings.TrimSpace(currentPart.String())
			if part != "" {
				f := parseFilter(part)
				if f.Field != "" {
					filters = append(filters, f)
				}
			}
			currentPart.Reset()
			continue
		}

		currentPart.WriteRune(ch)
	}

	part := strings.TrimSpace(currentPart.String())
	if part != "" {
		f := parseFilter(part)
		if f.Field != "" {
			filters = append(filters, f)
		}
	}

	return filters
}

func parseFilter(s string) Filter {
	for _, op := range []string{">=", "<=", "!=", ">", "<", "=", ":"} {
		idx := strings.Index(s, op)
		if idx > 0 {
			operator := op
			if operator == ":" {
				operator = "="
			}
			return Filter{
				Field:    strings.TrimSpace(s[:idx]),
				Operator: operator,
				Value:    strings.TrimSpace(s[idx+len(op):]),
			}
		}
	}
	return Filter{}
}

func ApplyFilters(doc map[string]interface{}, filters []Filter) bool {
	for _, f := range filters {
		if strings.HasPrefix(f.Field, "distance(") && strings.HasSuffix(f.Field, ")") {
			argsStr := f.Field[9 : len(f.Field)-1]
			args := strings.Split(argsStr, ",")
			if len(args) == 3 {
				geoField := strings.TrimSpace(args[0])
				lat, _ := strconv.ParseFloat(strings.TrimSpace(args[1]), 64)
				lng, _ := strconv.ParseFloat(strings.TrimSpace(args[2]), 64)

				geoVal, ok := doc[geoField]
				if !ok {
					return false
				}
				
				if geoMap, ok := geoVal.(map[string]interface{}); ok {
					docLat, lOk := toFloat64(geoMap["lat"])
					docLng, gOk := toFloat64(geoMap["lng"])
					if lOk && gOk {
						dist := haversineDistance(docLat, docLng, lat, lng)
						filterDist, _ := strconv.ParseFloat(f.Value, 64)
						if !compareNumeric(dist, f.Operator, filterDist) {
							return false
						}
						continue
					}
				}
				return false
			}
		}

		val, ok := doc[f.Field]
		if !ok {
			return false
		}
		if !matchFilter(val, f) {
			return false
		}
	}
	return true
}

func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371e3 // Earth radius in meters
	phi1 := lat1 * math.Pi / 180
	phi2 := lat2 * math.Pi / 180
	deltaPhi := (lat2 - lat1) * math.Pi / 180
	deltaLambda := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaPhi/2)*math.Sin(deltaPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*
			math.Sin(deltaLambda/2)*math.Sin(deltaLambda/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func matchFilter(val interface{}, f Filter) bool {
	docNum, docIsNum := toFloat64(val)
	filterNum, filterErr := strconv.ParseFloat(f.Value, 64)

	if docIsNum && filterErr == nil {
		return compareNumeric(docNum, f.Operator, filterNum)
	}

	docStr := toString(val)
	switch f.Operator {
	case "=":
		return strings.EqualFold(docStr, f.Value)
	case "!=":
		return !strings.EqualFold(docStr, f.Value)
	default:
		return docStr == f.Value
	}
}

func compareNumeric(a float64, op string, b float64) bool {
	switch op {
	case "=":
		return a == b
	case "!=":
		return a != b
	case ">":
		return a > b
	case "<":
		return a < b
	case ">=":
		return a >= b
	case "<=":
		return a <= b
	}
	return false
}

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case string:
		if f, err := strconv.ParseFloat(n, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

func toString(v interface{}) string {
	switch s := v.(type) {
	case string:
		return s
	default:
		return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(toFloat64OrZero(v), 'f', -1, 64), "0"), ".")
	}
}

func toFloat64OrZero(v interface{}) float64 {
	f, _ := toFloat64(v)
	return f
}
