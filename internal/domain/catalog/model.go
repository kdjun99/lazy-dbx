// Package catalog defines domain models and interfaces for database schema inspection.
package catalog

// Database represents a database (schema) in a connection.
type Database struct {
	Name string
}

// Table represents a table or view in a database.
type Table struct {
	Name   string
	Schema string
	Type   string // "BASE TABLE" or "VIEW"
}

// Column represents a column in a table.
type Column struct {
	Name       string
	OrdinalPos int
	DataType   string
	Nullable   bool
	KeyType    string // "PRI", "UNI", "MUL", or ""
	Default    string
}

// FormatType returns a human-readable type string, e.g. "varchar(255) [PK] NULL".
func (c Column) FormatType() string {
	s := c.DataType
	if c.KeyType == "PRI" {
		s += " [PK]"
	}
	if c.Nullable {
		s += " NULL"
	} else {
		s += " NOT NULL"
	}
	return s
}

// IsView reports whether the table is a database view.
func (t Table) IsView() bool {
	return t.Type == "VIEW"
}
