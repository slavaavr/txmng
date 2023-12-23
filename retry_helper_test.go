package txmng

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func Test_isPGXSerializationError(t *testing.T) {
	cases := []struct {
		name     string
		body     error
		expected bool
	}{
		{
			name: "pgx serialization error",
			body: &pgconn.PgError{
				Code: "40001",
			},
			expected: true,
		},
		{
			name:     "pgx serialization error, nested",
			body:     fmt.Errorf("some error: %w", &pgconn.PgError{Code: "40001"}),
			expected: true,
		},
		{
			name: "pgx not serialization error",
			body: &pgconn.PgError{
				Code: "test",
			},
			expected: false,
		},
		{
			name:     "pgx not serialization error, nested",
			body:     fmt.Errorf("some error: %w", &pgconn.PgError{Code: "test"}),
			expected: false,
		},
		{
			name:     "not serialization error",
			body:     errors.New("some error"),
			expected: false,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			actual := isPGXSerializationError(c.body)
			assert.Equal(t, c.expected, actual)
		})
	}
}
