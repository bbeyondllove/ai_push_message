package utils

import (
	"database/sql"
	"errors"
)

// IsSQLNoRowsError 检查错误是否为SQL无结果错误
func IsSQLNoRowsError(err error) bool {
	return err != nil && (errors.Is(err, sql.ErrNoRows) || err.Error() == "sql: no rows in result set")
}
