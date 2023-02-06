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

// gorm默认使用字段ID作为表主键，如果是其他名称则需配置: gorm:"primaryKey"
type TestUser struct {
	ID        uint64 `json:"id"   gorm:"primaryKey;autoIncrement"`
	Name      string `json:"name" gorm:"type:varchar(200);not null"`
	Age       uint16 `json:"age"  gorm:"not null"`
	Sex       uint8  `json:"sex"  gorm:"not null"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

//自定义表名称
/*func (u TestUser) TableName() string {
	return `t_test_person`
}*/

func resetTables(db *gorm.DB) {
	var user *TestUser
	var tmm *repo.TemplateModel

	util.AssertNilErr(db.Migrator().DropTable(user, tmm))
	util.AssertNilErr(db.AutoMigrate(user, tmm))
}

func mustInsertNUsersForQuery(db *gorm.DB, n int, resetTable bool) (users []*TestUser) {
	if resetTable {
		resetTables(db)
	}

	for i := 0; i < n; i++ {
		i := uint16(i + 1)
		users = append(users, &TestUser{Name: fmt.Sprintf(`user%v`, i), Age: i, Sex: uint8(i % 2)})
	}

	util.AssertNilErr(db.Create(users).Error)
	return
}

func mustInsertUsersForQuery(db *gorm.DB) (users []*TestUser) {
	return mustInsertNUsersForQuery(db, 5, false)
}

func mustGetDB(resetTable bool) *gorm.DB {
	db := orm.MustNewDefaultDB() //读取默认配置文件db.toml
	if resetTable {
		resetTables(db)
	}

	return db
}

func TestDBWithContext(t *testing.T) {
	r := require.New(t)
	db := mustGetDB(false)
	defer orm.CloseDB(db)

	ctx := orm.DBIntoContext(context.Background(), db)
	r.NotNil(ctx)

	db2, ok := orm.DBFromContext(ctx)
	r.True(ok && db2 != nil && db2 == db)

	db3, ok := orm.DBFromContext(context.Background())
	r.True(!ok && db3 == nil)
}

func TestBatchUpdates(t *testing.T) {
	r := require.New(t)
	db := mustGetDB(true)
	defer orm.CloseDB(db)

	assertUserEqual := func(user *TestUser) {
		user1 := &TestUser{ID: user.ID}
		r.NoError(db.Take(user1).Error)
		r.EqualValues(user1.Name, user.Name)
		r.EqualValues(user1.Age, user.Age)
	}

	//插入测试记录
	users := mustInsertUsersForQuery(db)[0:3]
	user0 := users[0]
	user1 := users[1]
	user2 := users[2]

	//更新全部字段，因未传入需更新的字段
	n, err := orm.BatchUpdate(db, users)
	r.NoError(err)
	r.EqualValues(len(users), n)
	assertUserEqual(user0)

	//更新指定字段，会同时更新updated_at字段
	user0.Name = `bingo`
	user1.Age = 100
	_, err = orm.BatchUpdate(db, users, orm.ScopeSelect(`name`, `age`))
	r.NoError(err)
	assertUserEqual(user0)
	assertUserEqual(user1)
	assertUserEqual(user2)

	//指定忽略更新字段，即更新除忽略字段以外的字段
	user2.Name = `bingo`
	_, err = orm.BatchUpdate(db, users, orm.ScopeOmit(`name`, `age`))
	r.NoError(err)

	user22 := &TestUser{ID: user2.ID}
	r.NoError(db.Take(user22).Error)
	r.NotEqual(user22.Name, user2.Name) //未更新name字段
}

type UserAgeScope struct {
	MinAge uint16
	MaxAge uint16
}

func (s UserAgeScope) Scope(db *gorm.DB) *gorm.DB {
	if s.MinAge >= 0 {
		db = db.Where(`age >= ?`, s.MinAge)
	}

	if s.MaxAge >= 0 {
		db = db.Where(`age <= ?`, s.MaxAge)
	}

	return db
}

type UserNameScope struct {
	Names []string
}

func (s UserNameScope) Scope(db *gorm.DB) *gorm.DB {
	if len(s.Names) > 0 {
		return db.Where(`name in ?`, s.Names)
	}

	return db
}

func TestScope(t *testing.T) {
	r := require.New(t)
	//ctx := context.Background()
	db := mustGetDB(true)
	defer orm.CloseDB(db)

	users := mustInsertUsersForQuery(db)

	ageScope := &UserAgeScope{
		MinAge: 1,
		MaxAge: 3,
	}

	nameScope := &UserNameScope{
		Names: []string{users[1].Name, users[2].Name},
	}

	scopes := orm.IScopes{ageScope, nameScope}

	var rsUsers1 []*TestUser
	var rsUsers2 []*TestUser
	r.NoError(db.Scopes(ageScope.Scope, nameScope.Scope).Find(&rsUsers1).Error)
	r.NoError(db.Scopes(scopes.Scope).Find(&rsUsers2).Error)
	r.Equal(rsUsers1, rsUsers2)

	page := orm.Pager{No: 1, Size: 1}
	scopes = append(scopes, page)

	var rsUsers3 []*TestUser
	r.NoError(db.Scopes(scopes.Scope).Find(&rsUsers3).Error)
	r.EqualValues(1, len(rsUsers3))
}
