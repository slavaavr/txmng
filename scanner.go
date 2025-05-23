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
	l := len(v.vals)
	if l != len(args) {
		return fmt.Errorf("vals has length=%d, but args has %d", l, len(args))
	}

	for i := 0; i < l; i++ {
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

func (v values) isValidArgKind(arg reflect.Kind) bool {
	switch arg {
	case reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128,
		reflect.Array,
		reflect.Map,
		reflect.Pointer,
		reflect.Slice,
		reflect.String,
		reflect.Struct:
		return true

	case reflect.Chan, // Unacceptable
		reflect.Func,
		reflect.Interface,
		reflect.UnsafePointer,
		reflect.Invalid:
	}

	return false
}
