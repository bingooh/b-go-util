package db

import (
	"context"
	"fmt"
	"github.com/bingooh/b-go-util/orm"
	"github.com/bingooh/b-go-util/test/db/repo"
	"github.com/bingooh/b-go-util/util"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
)

func mustInsertNTModelForQuery(db *gorm.DB, n int, resetTable bool) (models []*repo.TemplateModel) {
	if resetTable {
		resetTables(db)
	}

	for i := 1; i <= n; i++ {
		m := &repo.TemplateModel{Name: fmt.Sprintf(`user%v`, i), Age: i, IsSuper: i%2 == 0}
		models = append(models, m)
	}

	util.AssertNilErr(db.Create(models).Error)
	return
}

func TestRepoCUD(t *testing.T) {
	r := require.New(t)
	ctx := context.TODO()
	db := mustGetDB(true)
	defer orm.CloseDB(db)

	repo.TemplateRepo = repo.NewTemplateRepository(db)

	mustFindRow := func(id int64) *repo.TemplateModel {
		row, err := repo.TemplateRepo.FindOne(ctx, id)
		r.NoError(err)
		return row
	}

	var rows []*repo.TemplateModel
	for i := 1; i <= 3; i++ {
		m := &repo.TemplateModel{Name: fmt.Sprintf(`user%v`, i), Age: i, IsSuper: true}
		rows = append(rows, m)
	}

	r.NoError(repo.TemplateRepo.Create(ctx, rows[:2]))
	n, err := repo.TemplateRepo.Count(ctx)
	r.NoError(err)
	r.EqualValues(2, n)

	//插入1条记录，忽略age字段
	r.NoError(repo.TemplateRepo.Create(ctx, rows[2], orm.ScopeOmit(`age`)))
	row := mustFindRow(rows[2].ID)
	r.EqualValues(0, row.Age)

	row0 := rows[0]
	row0.Age = 100
	row0.IsSuper = false //不会更新零值字段
	n, err = repo.TemplateRepo.Update(ctx, row0)
	r.NoError(err)
	r.EqualValues(1, n)
	row = mustFindRow(row0.ID)
	r.True(row.IsSuper)

	//更新指定指定字段，包括零值字段
	n, err = repo.TemplateRepo.UpdateSelectColumns(ctx, row0, `name`, `age`, `is_super`)
	r.NoError(err)
	r.EqualValues(1, n)
	row = mustFindRow(row0.ID)
	r.False(row.IsSuper)

	n, err = repo.TemplateRepo.Incr(ctx, row0.ID, `age`, 2)
	r.NoError(err)
	r.EqualValues(1, n)
	row = mustFindRow(row0.ID)
	r.NoError(err)
	r.EqualValues(row0.Age+2, row.Age)

	//以下测试更新无主键值的记录
	//如果row无主键值，则会更新全部符合条件的记录。但必须指定where条件，或者启用全局更新
	noPKRow := &repo.TemplateModel{Name: `no`}
	n, err = repo.TemplateRepo.Update(ctx, noPKRow)
	r.Error(err)
	r.EqualValues(0, n)

	//设置where条件后，全局更新不会报错
	n, err = repo.TemplateRepo.Update(ctx, noPKRow, orm.ScopeWhereAll)
	r.NoError(err)
	r.EqualValues(len(rows), n)

	n, err = repo.TemplateRepo.Incr(ctx, 0, `age`, 2)
	r.Error(err)

	n, err = repo.TemplateRepo.Incr(ctx, 0, `age`, 2, orm.ScopeWhereAll)
	r.NoError(err)

	n, err = repo.TemplateRepo.BatchUpdate(ctx, rows, orm.ScopeSelect(`name`))
	r.NoError(err)
	r.EqualValues(len(rows), n)

	//测试删除，与update相同，记录无主键值则为全局更新，此时必须有where条件
	n, err = repo.TemplateRepo.Delete(ctx, row0.ID) //必须传条件值
	r.NoError(err)
	r.EqualValues(1, n)

	n, err = repo.TemplateRepo.BatchDelete(ctx, rows) //删除多条指定ID记录
	r.NoError(err)
	r.EqualValues(2, n)

	n, err = repo.TemplateRepo.BatchDelete(ctx, row0) //也可只传1条记录
	r.NoError(err)
	r.EqualValues(0, n) //已被前面操作删除

	n, err = repo.TemplateRepo.Delete(ctx, orm.Cond(`age=?`, 1)) //指定内联删除条件
	r.NoError(err)

	n, err = repo.TemplateRepo.Delete(ctx, `age=1`) //指定内联删除条件
	r.NoError(err)

	condRow := &repo.TemplateModel{Name: `a`, Age: 10}
	n, err = repo.TemplateRepo.Delete(ctx, condRow) //指定内联删除条件,会将所有非零值字段作为查询条件
	r.NoError(err)

	n, err = repo.TemplateRepo.Delete(ctx, `1=1`)
	r.NoError(err)

	n, err = repo.TemplateRepo.Delete(ctx, nil, orm.ScopeWhereAll)
	r.NoError(err)
}

func TestRepoQuery(t *testing.T) {
	r := require.New(t)
	ctx := context.TODO()

	db := mustGetDB(true)
	defer orm.CloseDB(db)

	mustInsertNTModelForQuery(db, 5, false)
	repo.TemplateRepo = repo.NewTemplateRepository(db)

	m1, err := repo.TemplateRepo.FindOne(ctx, 1)
	r.NoError(err)
	r.EqualValues(1, m1.ID)
	r.EqualValues(false, m1.IsSuper)

	m2, err := repo.TemplateRepo.FindOne(ctx, 2)
	r.NoError(err)
	r.EqualValues(2, m2.ID)
	r.EqualValues(true, m2.IsSuper)

	m3, err := repo.TemplateRepo.FindOne(ctx,
		orm.Cond(`age=?`, 3),              //内联条件
		orm.ScopeWhere(`name=?`, `user3`)) //外部条件
	r.NoError(err)
	r.EqualValues(3, m3.ID)

	//内联条件,仅包含非零值字段
	m4, err := repo.TemplateRepo.FindOne(ctx, &repo.TemplateModel{Name: `user4`, Age: 4})
	r.NoError(err)
	r.EqualValues(4, m4.ID)

	count, err := repo.TemplateRepo.Count(ctx)
	r.NoError(err)
	r.EqualValues(5, count)

	count, err = repo.TemplateRepo.Count(ctx, orm.ScopeWhere(`is_super=?`, 0))
	r.NoError(err)
	r.EqualValues(3, count)

	exist, err := repo.TemplateRepo.Exist(ctx, orm.ScopeWhere(`age>100`))
	r.NoError(err)
	r.False(exist)

	rows, err := repo.TemplateRepo.Find(ctx, nil)
	r.NoError(err)
	r.EqualValues(5, len(rows))

	rows, err = repo.TemplateRepo.Find(ctx, orm.Cond(1, 2))
	r.NoError(err)
	r.EqualValues(2, len(rows))

	condRow := &repo.TemplateModel{Name: `user1`}
	rows, err = repo.TemplateRepo.Find(ctx, condRow) //仅包含非零值字段
	r.NoError(err)
	r.EqualValues(1, len(rows))

	//使用where可指定条件字段，也可使用map作为查询条件
	rows, err = repo.TemplateRepo.Find(ctx, nil, orm.ScopeWhere(condRow, `name`, `is_super`))
	r.NoError(err)
	r.EqualValues(1, len(rows))

	condMap := orm.Map(`name`, `user1`, `is_super`, false)
	rows, err = repo.TemplateRepo.Find(ctx, condMap)
	r.NoError(err)
	r.EqualValues(1, len(rows))

	//todo 添加map示例

	rows, err = repo.TemplateRepo.Find(ctx, orm.Cond(`1=2`))
	r.NoError(err)
	r.EqualValues(0, len(rows))

	rows, err = repo.TemplateRepo.Find(ctx, nil, orm.ScopeLimit(2, 1))
	r.NoError(err)
	r.EqualValues(2, len(rows))

	rows, err = repo.TemplateRepo.Find(ctx, nil, orm.NewPager(2, 2))
	r.NoError(err)
	r.EqualValues(2, len(rows))

	//不建议使用以下方式拼接复杂SQL
	rows, err = repo.TemplateRepo.Find(ctx, nil,
		orm.ScopeSelect(`is_super,sum(age) as age`),
		orm.ScopeGroup(`is_super`),
		orm.ScopeOrder(`age desc`))
	r.NoError(err)
	r.EqualValues(2, len(rows))
	r.True(rows[0].Age >= rows[1].Age)

	//建议使用以下方式拼接复杂SQL
	scope := orm.Scope(func(db *gorm.DB) *gorm.DB {
		return db.Select(`is_super,sum(age) as age`).Group(`is_super`).Order(`age desc`)
	})
	rows, err = repo.TemplateRepo.Find(ctx, nil, scope)
	r.NoError(err)
	r.EqualValues(2, len(rows))

	rows, count, err = repo.TemplateRepo.FindWithCount(ctx, nil) //分页查询
	r.NoError(err)
	r.EqualValues(5, count)
	r.EqualValues(5, len(rows))

	rows, count, err = repo.TemplateRepo.FindWithCount(ctx, orm.NewPager(1, 2))
	r.NoError(err)
	r.EqualValues(5, count)
	r.EqualValues(2, len(rows))

	rows, count, err = repo.TemplateRepo.FindWithCount(ctx, orm.NewPager(10, 2))
	r.NoError(err)
	r.EqualValues(5, count)
	r.EqualValues(0, len(rows))

	rows, count, err = repo.TemplateRepo.FindWithCount(ctx, orm.NewPager(1, 2), orm.ScopeWhere(`1=2`))
	r.NoError(err)
	r.EqualValues(0, count)
	r.EqualValues(0, len(rows))

	//查询字段值
	ids, err := repo.TemplateRepo.FindID(ctx)
	r.NoError(err)
	r.EqualValues(5, len(ids))

	var name string
	err = repo.TemplateRepo.FindField(ctx, `name`, &name, orm.ScopeWhere(2))
	r.NoError(err)
	r.EqualValues(`user2`, name)

	var names []string
	err = repo.TemplateRepo.FindField(ctx, `name`, &names, orm.ScopeWhere(`id>2`))
	r.NoError(err)
	r.EqualValues(3, len(names))

	mp1 := make(map[int64]string)
	err = repo.TemplateRepo.FindFieldMap(ctx, `id`, `name`, mp1)
	r.NoError(err)
	r.EqualValues(5, len(mp1))
	r.EqualValues(`user1`, mp1[1])

}
