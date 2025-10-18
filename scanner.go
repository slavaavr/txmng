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

func (s values) Scan(args ...any) error {
	if len(s.vals) != len(args) {
		return fmt.Errorf("invalid size: expect %d, got %d", len(s.vals), len(args))
	}

	for i := 0; i < len(s.vals); i++ {
		argType := reflect.TypeOf(args[i])

		if argType == nil { // Scan(.., nil, ...)
			continue
		}

		if argType.Kind() != reflect.Pointer {
			return fmt.Errorf("invalid argument at position %d: must be a pointer", i)
		}

		argType = argType.Elem()
		valType := reflect.TypeOf(s.vals[i])

		if valType == nil { // Values(.., nil, ...)
			continue
		}

		if !s.isValidArgKind(argType.Kind()) {
			return fmt.Errorf("invalid argument kind at position %d: '%s'", i, argType.Kind().String())
		}

		if argType != valType {
			return fmt.Errorf("invalid argument type at position %d: expect '%s', got '%s'",
				i, valType.String(), argType.String())
		}

		argVal := reflect.ValueOf(args[i]).Elem()
		val := reflect.ValueOf(s.vals[i])

		if !argVal.CanSet() {
			return fmt.Errorf("unable to set value for argument at position %d: '%s.%s'",
				i, argVal.Kind().String(), argVal.String())
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

func (s values) isValidArgKind(arg reflect.Kind) bool {
	if _, ok := invalidArgKinds[arg]; ok {
		return false
	}

	return true
}
