package orm

import "gorm.io/gorm"

var (
	DefaultPageSize    = 30  //默认每页记录数
	DefaultMaxPageSize = 100 //默认每页最大记录数
)

// 用于分页
type Paging interface {
	IScope
	Limit() int
	Offset() int
}

type Pager struct {
	No          int `json:"page_no"`   //当前页码，从1开始
	Size        int `json:"page_size"` //每页记录数
	DefaultSize int `json:"-"`         //每页默认记录数
	MaxSize     int `json:"-"`         //每页最大记录数
}

func NewPager(no, size int) *Pager {
	return &Pager{
		No:          no,
		Size:        size,
		DefaultSize: DefaultPageSize,
		MaxSize:     DefaultMaxPageSize,
	}
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
	return db.Limit(p.Limit()).Offset(p.Offset())
}
