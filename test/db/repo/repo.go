package repo

import (
	"context"
	"github.com/bingooh/b-go-util/_reflect"
	"github.com/bingooh/b-go-util/orm"
	"github.com/bingooh/b-go-util/util"
	"gorm.io/gorm"
)

var TemplateRepo *TemplateRepository

type TemplateRepository struct {
	db *gorm.DB
}

func NewTemplateRepository(db *gorm.DB) *TemplateRepository {
	util.AssertOk(db != nil, `db为空`)
	return &TemplateRepository{db: db}
}

func (r *TemplateRepository) handleRecordNotFoundErr(row *TemplateModel, cause error) (*TemplateModel, error) {
	if orm.IsRecordNotFoundErr(cause) {
		return nil, nil
	}

	return row, cause
}

func (r *TemplateRepository) DB(ctx context.Context) *gorm.DB {
	db := orm.DBFromContextOrDefault(ctx, r.db)
	return db.WithContext(ctx)
}

func (r *TemplateRepository) Create(ctx context.Context, rows interface{}, scopes ...orm.IScope) error {
	return r.DB(ctx).Scopes(orm.IScopes(scopes).Scope).Create(rows).Error
}

func (r *TemplateRepository) Update(ctx context.Context, row *TemplateModel, scopes ...orm.IScope) (affected int64, err error) {
	rs := r.DB(ctx).Model(row).Scopes(orm.IScopes(scopes).Scope).Updates(row)
	return rs.RowsAffected, rs.Error
}

func (r *TemplateRepository) UpdateSelectColumns(ctx context.Context, row *TemplateModel, columns ...string) (affected int64, err error) {
	rs := r.DB(ctx).Model(row).Select(columns).Updates(row)
	return rs.RowsAffected, rs.Error
}

func (r *TemplateRepository) Incr(ctx context.Context, id int64, column string, value int, scopes ...orm.IScope) (affected int64, err error) {
	rs := r.DB(ctx).Model(&TemplateModel{ID: id}).Scopes(orm.IScopes(scopes).Scope).Update(column, orm.ExprIncr(column, value))
	return rs.RowsAffected, rs.Error
}

func (r *TemplateRepository) BatchUpdate(ctx context.Context, rows interface{}, scopes ...orm.IScope) (affected int64, err error) {
	return orm.BatchUpdate(r.DB(ctx), rows, scopes...)
}

func (r *TemplateRepository) Delete(ctx context.Context, cond interface{}, scopes ...orm.IScope) (affected int64, err error) {
	rs := r.DB(ctx).Scopes(orm.IScopes(scopes).Scope).Delete(&TemplateModel{}, orm.Cond(cond)...)
	return rs.RowsAffected, rs.Error
}

func (r *TemplateRepository) BatchDelete(ctx context.Context, rows interface{}, scopes ...orm.IScope) (affected int64, err error) {
	ids := _reflect.PluckStructFieldValues(`id`, true, true, rows)
	return r.Delete(ctx, ids, scopes...)
}

func (r *TemplateRepository) FindOne(ctx context.Context, cond interface{}, scopes ...orm.IScope) (*TemplateModel, error) {
	row := &TemplateModel{}
	err := r.DB(ctx).Scopes(orm.IScopes(scopes).Scope).Take(row, orm.Cond(cond)...).Error
	return r.handleRecordNotFoundErr(row, err)
}

func (r *TemplateRepository) Count(ctx context.Context, scopes ...orm.IScope) (count int64, err error) {
	err = r.DB(ctx).Scopes(orm.IScopes(scopes).Scope).Model(&TemplateModel{}).Count(&count).Error
	return
}

func (r *TemplateRepository) Exist(ctx context.Context, scopes ...orm.IScope) (exist bool, err error) {
	count, err := r.Count(ctx, scopes...)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *TemplateRepository) Find(ctx context.Context, cond interface{}, scopes ...orm.IScope) ([]*TemplateModel, error) {
	var rows []*TemplateModel
	err := r.DB(ctx).Scopes(orm.IScopes(scopes).Scope).Find(&rows, orm.Cond(cond)...).Error
	return rows, err
}

// FindWithCount 分页查询，建议在事务里调用此方法，否则查询条数可能不一致
func (r *TemplateRepository) FindWithCount(ctx context.Context, pager orm.Paging, scopes ...orm.IScope) (rows []*TemplateModel, count int64, err error) {
	stmt := r.DB(ctx).Model(&TemplateModel{}).Scopes(orm.IScopes(scopes).Scope)

	if err = stmt.Count(&count).Error; err != nil {
		return nil, 0, err
	}

	if count == 0 {
		return nil, 0, nil
	}

	if pager != nil {
		stmt = stmt.Limit(pager.Limit()).Offset(pager.Offset())
	}

	if err = stmt.Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	return rows, count, nil
}

// FindField 查询指定列名称的字段值并扫描到result，参数result必须为指针类型
func (r *TemplateRepository) FindField(ctx context.Context, column string, result interface{}, scopes ...orm.IScope) error {
	return r.DB(ctx).Model(&TemplateModel{}).Scopes(orm.IScopes(scopes).Scope).Pluck(column, result).Error
}

func (r *TemplateRepository) FindID(ctx context.Context, scopes ...orm.IScope) (ids []int64, err error) {
	err = r.FindField(ctx, `id`, &ids, scopes...)
	return
}

// FindFieldMap 查询指定列名称的字段值并扫描到map，参数result必须为非空map[keyColumn]valColumn
func (r *TemplateRepository) FindFieldMap(ctx context.Context, keyColumn, valColumn string, result interface{}, scopes ...orm.IScope) error {
	scopes = append(scopes, orm.ScopeSelect(keyColumn, valColumn)) //覆盖传入的select语句

	rows, err := r.Find(ctx, nil, scopes...)
	if err != nil {
		return err
	}

	return _reflect.ConvertStructListToMap(keyColumn, valColumn, result, rows)
}
