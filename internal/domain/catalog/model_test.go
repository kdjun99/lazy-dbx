package catalog_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdjun99/lazy-dbx/internal/domain/catalog"
)

func TestColumn_FormatType_PK(t *testing.T) {
	c := catalog.Column{
		Name:       "id",
		OrdinalPos: 1,
		DataType:   "int",
		Nullable:   false,
		KeyType:    "PRI",
		Default:    "",
	}
	assert.Equal(t, "int [PK] NOT NULL", c.FormatType())
}

func TestColumn_FormatType_Nullable(t *testing.T) {
	c := catalog.Column{
		Name:       "description",
		OrdinalPos: 2,
		DataType:   "varchar(255)",
		Nullable:   true,
		KeyType:    "",
		Default:    "",
	}
	assert.Equal(t, "varchar(255) NULL", c.FormatType())
}

func TestColumn_FormatType_Regular(t *testing.T) {
	c := catalog.Column{
		Name:       "name",
		OrdinalPos: 3,
		DataType:   "varchar(100)",
		Nullable:   false,
		KeyType:    "",
		Default:    "",
	}
	assert.Equal(t, "varchar(100) NOT NULL", c.FormatType())
}

func TestColumn_FormatType_PKNullable(t *testing.T) {
	c := catalog.Column{
		Name:       "id",
		OrdinalPos: 1,
		DataType:   "bigint",
		Nullable:   true,
		KeyType:    "PRI",
		Default:    "",
	}
	assert.Equal(t, "bigint [PK] NULL", c.FormatType())
}

func TestTable_IsView(t *testing.T) {
	tests := []struct {
		name     string
		table    catalog.Table
		expected bool
	}{
		{
			name:     "view type",
			table:    catalog.Table{Name: "v_users", Schema: "public", Type: "VIEW"},
			expected: true,
		},
		{
			name:     "base table",
			table:    catalog.Table{Name: "users", Schema: "public", Type: "BASE TABLE"},
			expected: false,
		},
		{
			name:     "mysql table",
			table:    catalog.Table{Name: "orders", Schema: "mydb", Type: "BASE TABLE"},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.table.IsView())
		})
	}
}
