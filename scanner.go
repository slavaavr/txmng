package txmng

import (
	"fmt"
	"reflect"
)

type Scanner interface {
	Scan(args ...any) error
}

type values struct {
	vals []any
}

func Values(args ...any) Scanner {
	return values{vals: args}
}

func (v values) Scan(args ...any) error {
	valsLen := len(v.vals)
	if valsLen != len(args) {
		return fmt.Errorf("vals has length=%d, but args has %d", valsLen, len(args))
	}

	for i := 0; i < valsLen; i++ {
		tArg := reflect.TypeOf(args[i])

		if tArg == nil { // Scan(.., nil, ...)
			continue
		}

		if tArg.Kind() != reflect.Pointer {
			return fmt.Errorf("arg %d is not a pointer", i)
		}

		tArgVal := tArg.Elem()
		tVal := reflect.TypeOf(v.vals[i])

		if tVal == nil { // Values(.., nil, ...)
			continue
		}

		if tArgVal != tVal {
			return fmt.Errorf("types are different %s, %s at position %d", tArgVal.String(), tVal.String(), i)
		}

		if !v.isValidArgKind(tArgVal.Kind()) {
			return fmt.Errorf("invalid arg kind %s at position %d", tArgVal.Kind().String(), i)
		}

		argVal := reflect.ValueOf(args[i]).Elem()
		val := reflect.ValueOf(v.vals[i])

		if !argVal.CanSet() {
			return fmt.Errorf(
				"can't set the value %s, %s at position %d",
				argVal.Kind().String(),
				argVal.String(),
				i,
			)
		}

		argVal.Set(val)
	}

	return nil
}

var invalidArgKinds = map[reflect.Kind]struct{}{
	reflect.Chan:          {},
	reflect.Func:          {},
	reflect.Interface:     {},
	reflect.UnsafePointer: {},
	reflect.Invalid:       {},
}

func (v values) isValidArgKind(arg reflect.Kind) bool {
	if _, ok := invalidArgKinds[arg]; ok {
		return false
	}

	return true
}
