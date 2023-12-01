package internal

import "reflect"

const (
	LongDateFmt  = "2006-01-02 15:04:05"
	ShortDateFmt = "2006-01-02"
)

func IndexArrayFunc(array any, equal func(i int) bool) int {
	arrayValues := reflect.ValueOf(array)
	for i := 0; i < arrayValues.Len(); i++ {
		if equal(i) {
			return i
		}
	}
	return -1
}
