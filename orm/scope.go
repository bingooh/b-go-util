package orm

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ScopeWhereAll = ScopeWhere(`1=1`)

// gorm未提供scope定义，这里定义为接口方便自定义查询条件
type IScope interface {
	Scope(db *gorm.DB) *gorm.DB
}

type Scope func(db *gorm.DB) *gorm.DB

func (f Scope) Scope(db *gorm.DB) *gorm.DB {
	return f(db)
}

type IScopes []IScope

func (s IScopes) Scope(db *gorm.DB) *gorm.DB {
	for _, sp := range s {
		db = db.Scopes(sp.Scope)
	}

	return db
}

func ScopeWhere(query interface{}, args ...interface{}) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(query, args...)
	}
}

func ScopeLimit(limit, offset int) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Limit(limit).Offset(offset)
	}
}

func ScopeOrder(value interface{}) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Order(value)
	}
}

func ScopeGroup(name string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Group(name)
	}
}

func ScopeHaving(query interface{}, args ...interface{}) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Having(query, args...)
	}
}

func ScopeExpr(expresses ...clause.Expression) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Clauses(expresses...)
	}
}

func ScopeExprForUpdate() Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Clauses(ExprForUpdate())
	}
}

func ScopeExprForShare() Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Clauses(ExprForShare())
	}
}

func ScopeDistinct(args ...interface{}) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Distinct(args...)
	}
}

func ScopeSelect(query interface{}, args ...interface{}) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Select(query, args...)
	}
}

func ScopeOmit(columns ...string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Omit(columns...)
	}
}
