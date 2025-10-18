package txmng

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func ref[T any]() *T     { var t T; return &t }
func refv[T any](t T) *T { return &t }

func TestScanner_Scan(t *testing.T) {
	type tmp struct {
		a int
	}

	cases := []struct {
		name        string
		values      Scanner
		actual      []interface{}
		expected    []interface{}
		expectedErr error
	}{
		{
			name:   "valid example",
			values: Values(1, 4.2, "hello", tmp{41}, &tmp{42}),
			actual: []interface{}{
				ref[int](),
				ref[float64](),
				ref[string](),
				ref[tmp](),
				ref[*tmp](),
			},
			expected: []interface{}{
				refv(1),
				refv(4.2),
				refv("hello"),
				refv(tmp{41}),
				refv(&tmp{42}),
			},
			expectedErr: nil,
		},
		{
			name:        "valid example, omitting values using nil",
			values:      Values(1, 2, 3),
			actual:      []interface{}{nil, ref[int](), nil},
			expected:    []interface{}{nil, refv(2), nil},
			expectedErr: nil,
		},
		{
			name:        "valid example, omitting values using nil 2",
			values:      Values(nil, 2, 3),
			actual:      []interface{}{ref[int](), ref[int](), nil},
			expected:    []interface{}{refv(0), refv(2), nil},
			expectedErr: nil,
		},
		{
			name:        "scan error: different lengths",
			values:      Values(1, 2, 3),
			actual:      []interface{}{ref[int](), ref[int]()},
			expected:    nil,
			expectedErr: fmt.Errorf("invalid size: expect %d, got %d", 3, 2),
		},
		{
			name:        "scan error: arg is not a pointer",
			values:      Values(1, 2, 3),
			actual:      []interface{}{nil, 0, nil},
			expected:    nil,
			expectedErr: fmt.Errorf("invalid argument at position %d: must be a pointer", 1),
		},
		{
			name:        "scan error: invalid arg kind",
			values:      Values(func() {}),
			actual:      []interface{}{ref[func()]()},
			expected:    nil,
			expectedErr: fmt.Errorf("invalid argument kind at position %d: '%s'", 0, "func"),
		},
		{
			name:        "scan error: different types",
			values:      Values(1, 2, 3),
			actual:      []interface{}{nil, nil, ref[float64]()},
			expected:    nil,
			expectedErr: fmt.Errorf("invalid argument type at position %d: expect '%s', got '%s'", 2, "int", "float64"),
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			err := c.values.Scan(c.actual...)
			if c.expectedErr != nil {
				assert.Equal(t, c.expectedErr, err)
			} else {
				assert.Equal(t, c.expected, c.actual)
			}
		})
	}
}
