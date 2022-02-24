package orm

import (
	"b-go-util/_interface"
	"b-go-util/_string"
	"b-go-util/slog"
	"context"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"reflect"
)

type ctxKey int

var (
	ctxKeyDB      ctxKey = 1
	ctxKeyClauses ctxKey = 2
)

func newLogger(tag string) *zap.Logger {
	return slog.NewLogger(`db`, tag)
}

func IsRecordNotFoundErr(err error) bool {
	return err == gorm.ErrRecordNotFound
}

func DBIntoContext(parent context.Context, db *gorm.DB) context.Context {
	return context.WithValue(parent, ctxKeyDB, db)
}

func DBFromContext(ctx context.Context) (db *gorm.DB, ok bool) {
	if ctx != nil {
		db, ok = ctx.Value(ctxKeyDB).(*gorm.DB)
	}

	return
}

func DBFromContextOrDefault(ctx context.Context, defaultDB *gorm.DB) *gorm.DB {
	if db, ok := DBFromContext(ctx); ok {
		return db
	}

	return defaultDB
}

func WithLockingForUpdate(parent context.Context) context.Context {
	v := parent.Value(ctxKeyClauses)
	var expressions []clause.Expression
	if v != nil {
		expressions = v.([]clause.Expression)
	}

	return context.WithValue(parent, ctxKeyClauses, append(expressions, ExprForUpdate()))
}

func ClausesFromContext(ctx context.Context) []clause.Expression {
	v := ctx.Value(ctxKeyClauses)
	if v == nil {
		return nil
	}

	return v.([]clause.Expression)
}

//是否为基本数据类型
func isPrimitiveKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Array, reflect.Struct, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Ptr, reflect.UnsafePointer:
		return false
	default:
		return true
	}
}

//获取字段值
func getFieldValue(fieldName string, ignoreZeroValue bool, record interface{}) (interface{}, bool) {
	var rv reflect.Value
	if m, ok := record.(map[string]interface{}); ok {
		if len(m) == 0 {
			return nil, false
		}

		rv = reflect.ValueOf(m[fieldName])
	} else {
		//说明是struct
		rv = reflect.ValueOf(record)
		if !rv.IsValid() {
			return nil, false
		}

		if rv.Kind() == reflect.Ptr {
			if rv.IsNil() {
				return nil, false
			}

			rv = rv.Elem()
		}

		if rv.Kind() != reflect.Struct {
			return nil, false
		}

		rv = reflect.Indirect(rv.FieldByName(fieldName))
	}

	if !rv.IsValid() ||
		!isPrimitiveKind(rv.Kind()) ||
		rv.IsZero() && ignoreZeroValue {
		return nil, false
	}

	return rv.Interface(), true
}

//获取记录的指定字段值
//注意：此方法使用反射获取字段值，处理大量数据需注意性能
//records数据类型可以为struct/map[string]interface{}，或对应的slice
//参数fieldName区分大小写，如果是struct，仅支持公共字段
//返回值仅包括基本数据类型，不包括重复值。如果ignoreZeroValue==true，则不包括零值
func GetFieldValues(fieldName string, ignoreZeroValue bool, records interface{}) (values []interface{}) {
	if records == nil || _string.Empty(fieldName) {
		return
	}

	rv := reflect.ValueOf(records)
	if !rv.IsValid() {
		return
	}

	if rv.Kind() != reflect.Slice {
		if val, ok := getFieldValue(fieldName, ignoreZeroValue, records); ok {
			values = append(values, val)
		}

		return
	}

	//说明是slice
	m := make(map[interface{}]uint8) //用于去重
	for i := 0; i < rv.Len(); i++ {
		if val, ok := getFieldValue(fieldName, ignoreZeroValue, rv.Index(i).Interface()); ok {
			if _, exist := m[val]; !exist {
				m[val] = 0
				values = append(values, val)
			}
		}
	}

	return
}

func ExprForUpdate() clause.Expression {
	return clause.Locking{Strength: "UPDATE"}
}

func ExprOnConflictDoNothing(conflictColumn string) clause.Expression {
	return clause.OnConflict{
		Columns:   []clause.Column{{Name: conflictColumn}},
		DoNothing: true,
	}
}

//如果参数updateColumns为空，则更新全部字段
func ExprOnConflictDoUpdate(conflictColumn string, updateColumns ...string) clause.Expression {
	c := clause.OnConflict{
		Columns: []clause.Column{{Name: conflictColumn}},
	}

	if len(updateColumns) == 0 {
		c.UpdateAll = true
	} else {
		c.DoUpdates = clause.AssignmentColumns(updateColumns)
	}

	return c
}

func ExprIncr(column string, n int) clause.Expr {
	return gorm.Expr(fmt.Sprintf(`%s%+d`, column, n))
}

//批量更新记录帮助类
type BatchUpdateHelper struct {
	db            *gorm.DB
	columns       []string //更新的字段
	isOmitColumns bool     //是否为忽略更新的字段
}

func NewBatchUpdateHelper(db *gorm.DB, columns ...string) *BatchUpdateHelper {
	return &BatchUpdateHelper{db: db, columns: columns}
}

func (h *BatchUpdateHelper) getDB(ctx context.Context) *gorm.DB {
	return DBFromContextOrDefault(ctx, h.db).WithContext(ctx)
}

func (h *BatchUpdateHelper) WithDB(db *gorm.DB) *BatchUpdateHelper {
	h.db = db
	return h
}

func (h *BatchUpdateHelper) WithColumns(columns ...string) *BatchUpdateHelper {
	h.columns = columns
	h.isOmitColumns = false
	return h
}

func (h *BatchUpdateHelper) WithOmitColumns(columns ...string) *BatchUpdateHelper {
	h.columns = columns
	h.isOmitColumns = true
	return h
}

//参数records可直接传入记录数组，如：[]*User
func (h *BatchUpdateHelper) Updates(ctx context.Context, records ...interface{}) error {
	switch len(records) {
	case 0:
		return nil
	case 1:
		records = _interface.Flat(records[0]) //records[0]可能为数组
	}

	for _, record := range records {
		stmt := h.getDB(ctx).Model(record)

		if len(h.columns) > 0 {
			if h.isOmitColumns {
				stmt.Omit(h.columns...)
			} else {
				stmt.Select(h.columns)
			}
		}

		if err := stmt.Updates(record).Error; err != nil {
			return err
		}
	}

	return nil
}
