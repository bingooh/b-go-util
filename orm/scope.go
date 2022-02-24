package orm

import "gorm.io/gorm"

//gorm未提供scope定义，这里定义为接口方便自定义查询条件
type IScope interface {
	Scope(db *gorm.DB) *gorm.DB
}

type Scope func(db *gorm.DB) *gorm.DB

func (f Scope) Scope(db *gorm.DB) *gorm.DB {
	return f(db)
}

type IScopes []IScope

func (s IScopes) Scope(db *gorm.DB) *gorm.DB {
	for _, iScope := range s {
		db = db.Scopes(iScope.Scope)
	}

	return db
}

//用于分页
type Paging interface {
	Limit() int
	Offset() int
}

func PageScope(pager Paging) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Limit(pager.Limit()).Offset(pager.Offset())
	}
}

type Pager struct {
	No          int `json:"page_no"`   //当前页码，从1开始
	Size        int `json:"page_size"` //每页记录数
	DefaultSize int `json:"-"`         //每页默认记录数
	MaxSize     int `json:"-"`         //每页最大记录数
}

func (p Pager) Limit() int {
	if p.Size <= 0 {
		return p.DefaultSize
	}

	if p.MaxSize > 0 && p.Size > p.MaxSize {
		return p.MaxSize
	}

	return p.Size
}

func (p Pager) Offset() int {
	if limit := p.Limit(); limit > 0 && p.No > 1 {
		return limit * (p.No - 1)
	}

	return 0
}

func (p Pager) Scope(db *gorm.DB) *gorm.DB {
	return PageScope(p).Scope(db)
}
