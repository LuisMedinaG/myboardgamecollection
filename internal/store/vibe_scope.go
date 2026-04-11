package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// ErrForeignOwnership is returned when a mutation references games or vibes
// that are not owned by the authenticated user.
var ErrForeignOwnership = errors.New("one or more ids are not owned by the user")

func ownedIDs(tx *sql.Tx, table string, userID int64, ids []int64) (map[int64]bool, error) {
	if len(ids) == 0 {
		return map[int64]bool{}, nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	query := fmt.Sprintf("SELECT id FROM %s WHERE user_id = ? AND id IN (%s)", table, placeholders)

	args := make([]any, 0, len(ids)+1)
	args = append(args, userID)
	for _, id := range ids {
		args = append(args, id)
	}

	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64]bool, len(ids))
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result[id] = true
	}
	return result, rows.Err()
}

func IsOwnershipError(err error) bool {
	return errors.Is(err, ErrForeignOwnership)
}
