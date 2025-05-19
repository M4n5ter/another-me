package option

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
)

// Scan 从数据库驱动程序分配一个值。
// 此方法是 database/sql.Scanner 接口所必需的。
func (o *Option[T]) Scan(src any) error {
	// 通过 sql.Null[T] 的绕行允许我们访问将扫描值分配给内置类型和标准类型（如 *sql.Rows）的标准规则，
	// 这些规则不会直接从标准库导出。
	var v sql.Null[T]
	err := v.Scan(src)
	if err != nil {
		return fmt.Errorf("failed to scan option: %w", err)
	}
	if v.Valid {
		*o = Some(v.V)
	} else {
		*o = None[T]()
	}
	return nil
}

// Value 返回一个驱动程序值。
// 此方法是 database/sql/driver.Valuer 接口所必需的。
func (o Option[T]) Value() (driver.Value, error) {
	if o.IsNone() {
		return nil, nil
	}

	v, err := driver.DefaultParameterConverter.ConvertValue(o.Unwrap())
	if err != nil {
		return nil, fmt.Errorf("failed to convert option value: %w", err)
	}
	return v, nil
}
