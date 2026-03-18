package engine

import (
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
	parts := strings.Split(raw, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
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
