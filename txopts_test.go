package txmng

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsolationLevel_toSQL(t *testing.T) {
	cases := []struct {
		name     string
		level    IsolationLevel
		expected sql.IsolationLevel
	}{
		{
			name:     "LevelDefault",
			level:    LevelDefault,
			expected: sql.LevelDefault,
		},
		{
			name:     "LevelReadUncommitted",
			level:    LevelReadUncommitted,
			expected: sql.LevelReadUncommitted,
		},
		{
			name:     "LevelReadCommitted",
			level:    LevelReadCommitted,
			expected: sql.LevelReadCommitted,
		},
		{
			name:     "LevelWriteCommitted",
			level:    LevelWriteCommitted,
			expected: sql.LevelWriteCommitted,
		},
		{
			name:     "LevelRepeatableRead",
			level:    LevelRepeatableRead,
			expected: sql.LevelRepeatableRead,
		},
		{
			name:     "LevelSnapshot",
			level:    LevelSnapshot,
			expected: sql.LevelSnapshot,
		},
		{
			name:     "LevelSerializable",
			level:    LevelSerializable,
			expected: sql.LevelSerializable,
		},
		{
			name:     "LevelLinearizable",
			level:    LevelLinearizable,
			expected: sql.LevelLinearizable,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			actual := c.level.toSQL()
			assert.Equal(t, c.expected, actual)
		})
	}
}
