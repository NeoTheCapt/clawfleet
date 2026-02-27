package store

import "database/sql"

// queryRows is a small helper to reduce repeated rows.Next/Close loops.
func queryRows[T any](db *sql.DB, query string, args []interface{}, scan func(*sql.Rows) (*T, error)) ([]*T, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*T
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}
