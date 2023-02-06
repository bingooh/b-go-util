package orm

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/bingooh/b-go-util/_interface"
	"github.com/bingooh/b-go-util/_reflect"
	"github.com/bingooh/b-go-util/_string"
	"github.com/bingooh/b-go-util/slog"
	"github.com/bingooh/b-go-util/util"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
)

type ctxKeyDB struct{}

type Transaction func(fc func(tx *gorm.DB) error, opts ...*sql.TxOptions) error

func newLogger(tag string) *zap.Logger {
	return slog.NewLogger(`db`, tag)
}

// Cond 内联查询条件
func Cond(args ...interface{}) []interface{} {
	switch len(args) {
	case 0:
		return nil
	case 1:
		if _reflect.IsNil(args[0]) {
			return nil
		}

		return _interface.Flat(args[0])
	default:
		return args
	}
}

func Map(kv ...interface{}) map[string]interface{} {
	if len(kv) == 0 {
		return nil
	}
	util.AssertOk(len(kv)%2 == 0, `kv参数个数不是偶数`)

	m := make(map[string]interface{})
	var key string
	for i, v := range kv {
		if i%2 == 0 {
			if k, ok := v.(string); ok {
				key = k
			} else {
				key = fmt.Sprintf(`%v`, v)
			}

			continue
		}

		if !_string.Empty(key) {
			m[key] = v
		}
	}

	return m
}

func IsRecordNotFoundErr(err error) bool {
	return err == gorm.ErrRecordNotFound
}

func DBIntoContext(parent context.Context, db *gorm.DB) context.Context {
	return context.WithValue(parent, ctxKeyDB{}, db)
}

func DBFromContext(ctx context.Context) (db *gorm.DB, ok bool) {
	if ctx != nil {
		db, ok = ctx.Value(ctxKeyDB{}).(*gorm.DB)
	}

	return
}

func DBFromContextOrDefault(ctx context.Context, defaultDB *gorm.DB) *gorm.DB {
	if db, ok := DBFromContext(ctx); ok {
		return db
	}

	return defaultDB
}

func ExprIncr(column string, n int) clause.Expr {
	return gorm.Expr(fmt.Sprintf(`%s%+d`, column, n))
}

func ExprForUpdate() clause.Expression {
	return clause.Locking{Strength: "UPDATE"}
}

func ExprForShare() clause.Expression {
	return clause.Locking{Strength: "SHARE"}
}

func ToClauseColumns(columns ...string) (rs []clause.Column) {
	for _, column := range columns {
		if !_string.Empty(column) {
			rs = append(rs, clause.Column{Name: column})
		}
	}

	return
}

func ExprOnConflictDoNothing(conflictColumns string) clause.Expression {
	return clause.OnConflict{
		Columns:   ToClauseColumns(strings.Split(conflictColumns, `,`)...),
		DoNothing: true,
	}
}

// ExprOnConflictDoUpdate 参数conflictColumns多个用逗号分隔。如果参数updateColumns为空，则更新全部字段
func ExprOnConflictDoUpdate(conflictColumns string, updateColumns ...string) clause.Expression {
	c := clause.OnConflict{
		Columns: ToClauseColumns(strings.Split(conflictColumns, `,`)...),
	}

	if len(updateColumns) == 0 {
		c.UpdateAll = true
	} else {
		c.DoUpdates = clause.AssignmentColumns(updateColumns)
	}

	return c
}

func BatchUpdate(db *gorm.DB, records interface{}, scopes ...IScope) (affected int64, err error) {
	rows := _interface.Flat(records) //records可能为切片
	if len(rows) == 0 {
		return 0, nil
	}

	for _, row := range rows {
		rs := db.Model(row).Scopes(IScopes(scopes).Scope).Updates(row)
		if rs.Error != nil {
			return affected, err
		}

		affected += 1
	}

	return affected, nil
}
