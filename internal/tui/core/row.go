package core

import "fmt"

// InterfaceSliceToStrings converts a slice of interface{} to a slice of strings.
func InterfaceSliceToStrings(row []interface{}) []string {
	result := make([]string, len(row))
	for i, v := range row {
		if v == nil {
			result[i] = "NULL"
		} else if b, ok := v.([]byte); ok {
			result[i] = string(b)
		} else {
			result[i] = fmt.Sprintf("%v", v)
		}
	}
	return result
}
