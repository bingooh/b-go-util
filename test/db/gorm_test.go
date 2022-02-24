package db

import (
	"database/sql"
	"fmt"
	"github.com/bingooh/b-go-util/orm"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"testing"
	"time"
)

func TestSqlPrintln(t *testing.T) {
	db := mustGetDB(false)
	defer orm.CloseDB(db)

	//打印SQL，需要设置为DryRun，否则Statement执行后不再有SQL
	s := db.Session(&gorm.Session{DryRun: true})
	stmt := s.Save(&TestUser{Name: `bingo`}).Statement

	sql := stmt.SQL.String()
	fmt.Println(sql)
	fmt.Println(stmt.Dialector.Explain(sql, stmt.Vars...))
}

func TestSqlCUD(t *testing.T) {
	r := require.New(t)
	db := mustGetDB(true)
	defer orm.CloseDB(db)

	user1 := &TestUser{Name: `user1`}
	user2 := &TestUser{Name: `user2`}
	user3 := &TestUser{Name: `user3`}

	//create语句基本结构：
	//db.select(指定选择的字段).omit(指定忽略的字段).create(插入记录)
	r.NoError(db.Create(user1).Error)
	r.EqualValues(1, user1.ID)
	r.True(user1.CreatedAt > 0)
	r.True(user1.UpdatedAt > 0)

	rs := db.Create([]*TestUser{user2, user3})
	r.NoError(rs.Error)
	r.EqualValues(2, rs.RowsAffected)
	r.True(user2.ID > 0)
	r.True(user3.ID > 0)
	r.True(user2.CreatedAt > 0)
	r.True(user3.CreatedAt > 0)

	//update语句基本结构：
	//db.model(模型/id过滤条件).where(过滤条件).select(需更新的字段).updates(更新字段和值)
	time.Sleep(1 * time.Second) //暂停1秒，以比较UpdatedAt字段值是否有更新
	user11 := *user1
	user1.Name = `b1`

	//updates()默认只更新非零值字段， 以下仅更新name字段
	//如果需跳过hook方法，可使用UpdateColumn()、UpdateColumns()
	r.NoError(db.Model(user1).Update(`name`, `b1`).Error)
	r.NoError(db.Model(user1).Updates(&TestUser{Name: `b1`}).Error)
	r.NoError(db.Model(user1).Select(`name`).Updates(user1).Error)
	r.True(user11.CreatedAt == user1.CreatedAt)
	r.True(user11.UpdatedAt < user1.UpdatedAt) //UpdatedAt字段有自动设回,实际上gorm利用hook先设置user.UpdatedAt，再拼接SQL并执行

	//更新操作默认会添加传入的user的主键作为where条件，且禁止全局更新
	//以下将报错，因为noPKUser没有主键值，即没有设置where条件，执行全局更新
	//需要启用AllowGlobalUpdate==true，或者传入的user的主键值不为零值
	noPKUser := &TestUser{Age: 10}
	r.Error(db.Model(noPKUser).Updates(noPKUser).Error)
	r.NoError(db.Model(noPKUser).Where(1).Updates(noPKUser).Error)

	//全局更新
	r.NoError(db.Model(noPKUser).Where(`1=1`).Updates(noPKUser).Error)
	r.NoError(db.Session(&gorm.Session{AllowGlobalUpdate: true}).Model(noPKUser).Updates(noPKUser).Error)
	r.NoError(db.Exec(`update t_test_user set age=9, updated_at=?`, time.Now().Unix()).Error)

	//更新多模型对象，只能遍历更新，不支持传入users数组
	//另见：TestBatchUpdates()
	user2.Name = `b2`
	user3.Name = `b3`
	r.NoError(db.Model(user2).Select(`name`).Updates(user2).Error)
	r.NoError(db.Model(user3).Select(`name`).Updates(user3).Error)

	//加载当前记录，获取当前age值
	r.NoError(db.First(user1).Error)
	user11 = *user1

	//使用表达式更新，age+1
	r.NoError(db.Model(user1).Update(`age`, gorm.Expr(`age+1`)).Error)
	r.NoError(db.Model(user1).Update(`age`, orm.ExprIncr(`age`, 1)).Error)
	r.NoError(db.Model(user1).Update(`age`, orm.ExprIncr(`age`, -1)).Error)
	r.NoError(db.First(user1).Error)
	r.EqualValues(user11.Age+1, user1.Age)

	//delete语句基本结构
	//db.where(过滤条件).delete(模型/内联条件)
	rs = db.Delete(user1) //添加id过滤条件
	r.NoError(rs.Error)
	r.EqualValues(1, rs.RowsAffected)

	r.NoError(db.Delete(noPKUser, 1).Error)
	r.NoError(db.Delete(noPKUser, `id=?`, 1).Error)
	r.NoError(db.Where(1).Delete(noPKUser).Error)

	//全局删除，默认必须有where条件。其他全局删除方式同全局更新
	rs = db.Where(`1=1`).Delete(noPKUser)
	r.NoError(rs.Error)
	r.EqualValues(2, rs.RowsAffected)

	//不建议使用软删除，可参考其字段设计
}

//SQL语句：
//1个Statement包含多个Clause，1个Clause包含1个SQL语句(表达式)和对应参数
//db.Model()等会创建1个Statement，后续方法调用会添加/删除此Statement包含的Clause
//db.Create()/Find()等会解析Statement包含的Clause拼接完整SQL并执行获取执行结果
//
//gorm的方法：
// - 链式方法：     除Finisher方法外的方法，但db.Raw()后面不再支持链接方法
// - Finisher方法：如Create()/Find()
//
//调用链式方法会修改Statement的Clause，调用Finisher方法会执行SQL
//首次调用链式方法会创建1个Statement，最近1次调用Finisher方法会销毁当前的Statement，这成为1次会话
//如果直接调用Finisher方法，会先创建后再销毁Statement，仍然为1次会话
//会话期间创建的Statement是线程不安全的
//
//db.Session()可以创建1个新会话，默认带有父会话(Statement)的Clause，见测试TestDBSession()
func TestSqlQuery(t *testing.T) {
	r := require.New(t)
	db := mustGetDB(true)
	defer orm.CloseDB(db)

	//插入测试记录
	users := mustInsertUsersForQuery(db)

	//query语句基本结构:
	//db.First(结果对象,主键值/内联条件)
	//db.Where(外部条件/查询对象).First(结果对象)
	//
	//db.First()/Last()/Take() 如果查询结果为空，返回gorm.ErrRecordNotFound（此操作自动添加查询条件：limit 1）
	//db.Find()                如果查询结果为空，不会返回gorm.ErrRecordNotFound
	//
	//First()/Find()等可设置内联条件
	// - 默认为主键值
	// - 默认添加传入struct的相应名称作为目标表名称
	// - 默认添加传入struct的非零值主键作为过滤条件
	//Where()可设置外部条件：
	// - 默认为主键值，可接收string/map/struct作为过滤条件
	// - 默认忽略struct零值字段值，可指定要使用的过滤条件字段

	//查询单条记录(自动添加查询条件：limit 1)
	user := &TestUser{}
	r.NoError(db.First(user).Error) //查询第1条记录，根据主键升序排序
	r.EqualValues(users[0].ID, user.ID)

	user = &TestUser{}
	r.NoError(db.Last(user).Error) //查询最后1条记录，根据主键升序排序
	r.EqualValues(users[len(users)-1].ID, user.ID)

	user = &TestUser{}
	r.NoError(db.Take(user).Error) //查询第1条记录，无排序
	r.EqualValues(users[0].ID, user.ID)

	//根据主键查询
	user = &TestUser{}
	r.NoError(db.First(user, 2).Error)
	r.EqualValues(2, user.ID)

	user = &TestUser{}
	r.NoError(db.Take(user, 3).Error) //建议优先使用take()，无排序
	r.EqualValues(3, user.ID)

	var rsUsers []*TestUser
	r.NoError(db.Find(&rsUsers, []int{1, 2}).Error)
	r.EqualValues(2, len(rsUsers))

	rsUsers = nil
	r.NoError(db.Where([]int{1, 2}).Find(&rsUsers).Error)
	r.EqualValues(2, len(rsUsers))

	user = &TestUser{}
	r.NoError(db.Find(user, 1).Error) //Find()也可以扫描1条记录
	r.EqualValues(1, user.ID)

	//Take()/Find()等方法默认添加主键作为过滤条件
	user = &TestUser{ID: 4}
	r.NoError(db.Take(user).Error)
	r.EqualValues(4, user.ID)

	user = &TestUser{ID: 4}
	r.ErrorIs(gorm.ErrRecordNotFound, db.Take(user, 5).Error) //where id=4 and id=5

	//传入的users数组里的user包含的主键不会添加为过滤条件
	rsUsers = []*TestUser{
		{ID: 1}, {ID: 2},
	}
	r.NoError(db.Find(&rsUsers).Error) //查询全部记录
	r.EqualValues(len(users), len(rsUsers))

	//查询多条记录
	rsUsers = nil //可以为空，不会引起错误
	r.NoError(db.Find(&rsUsers).Error)
	r.EqualValues(len(users), len(rsUsers))

	rsUsers = nil
	r.NoError(db.Limit(1).Offset(1).Find(&rsUsers).Error)
	r.EqualValues(1, len(rsUsers))
	r.EqualValues(users[1].ID, rsUsers[0].ID)

	rsUsers = nil
	stmt := db.Select(`id`).Where(`id>?`, 1).Order(`age desc,id`) //根据年龄倒序排序
	r.NoError(stmt.Find(&rsUsers).Error)
	r.EqualValues(4, len(rsUsers))
	r.EqualValues(0, rsUsers[0].Age) //未查询年龄字段，所以为0
	r.EqualValues(users[len(users)-1].ID, rsUsers[0].ID)

	rsUsers = nil
	r.NoError(db.Select(`sex,sum(age) as age`).Group(`sex`).Find(&rsUsers).Error) //根据性别统计年龄
	r.EqualValues(2, len(rsUsers))                                                //将结果扫描进rsUsers，字段名称必须对应
	r.True(rsUsers[0].Age > 0 && rsUsers[1].Age > 0)

	//使用struct/map指定查询条件,默认忽略struct的零值字段
	rsUsers = nil
	condMap := map[string]interface{}{`sex`: 1, `age`: 0}
	r.NoError(db.Where(condMap).Find(&rsUsers).Error) //包含age==0条件，查询结果为空，但不会返回gorm.ErrRecordNotFound
	r.True(len(rsUsers) == 0)

	rsUsers = nil
	condUser := &TestUser{Sex: 1}
	r.NoError(db.Where(condUser).Find(&rsUsers).Error) //不包含age条件，忽略零值字段
	r.True(len(rsUsers) > 0)

	rsUsers = nil
	r.NoError(db.Where(condUser, `sex`, `age`).Find(&rsUsers).Error) //指定查询条件字段
	r.True(len(rsUsers) == 0)
}

func TestSqlQueryScan(t *testing.T) {
	r := require.New(t)
	db := mustGetDB(true)
	defer orm.CloseDB(db)

	//插入测试记录
	users := mustInsertUsersForQuery(db)

	//将结果扫描进自定义struct/map
	//以下测试调用db.Model()/Table()等指定目标表名称，否则报错
	type Result struct {
		Sex      uint8
		TotalAge uint
	}

	rs := &Result{}
	r.NoError(db.Model(&TestUser{}).Select(`sex,sum(age) as TotalAge`).Group(`sex`).Take(rs).Error)
	r.True(rs.TotalAge > 0)

	//查询结果被转换为字符串，不推荐使用map存储查询结果
	var rsMap = make(map[string]interface{})
	r.NoError(db.Table(`t_test_user`).Select(`sex,sum(age) as TotalAge`).Group(`sex`).Take(&rsMap).Error)
	totalAge, ok := rsMap[`TotalAge`].(string)
	r.True(ok && totalAge != `0`)

	//利用别名避免使用自定义struct，适合小部分情况
	user := &TestUser{}
	r.NoError(db.Select(`sex,sum(age) as age`).Group(`sex`).Take(user).Error)
	r.True(user.Age > 0)

	//scan()扫描结果到目标对象
	user = &TestUser{}
	r.NoError(db.Model(&TestUser{}).Scan(user).Error) //查询多条记录，但只扫描第1条。建议使用take()
	r.True(user.Age > 0)

	var rsUsers []*TestUser
	r.NoError(db.Model(&TestUser{}).Scan(&rsUsers).Error) //扫描多条记录。建议使用Find()
	r.EqualValues(len(users), len(rsUsers))

	sumAge := 0
	r.NoError(db.Model(&TestUser{}).Select(`sum(age)`).Scan(&sumAge).Error) //扫描单个字段值，建议使用take()
	r.True(sumAge > 0)

	//逐行扫描
	//注意：必须关闭返回的rows(遍历完最后1条记录会自动关闭)，否则会一直占有1个数据库连接，可能导致数据库连接耗光
	rows, err := db.Model(&TestUser{}).Limit(2).Rows()
	defer rows.Close()

	r.NoError(err)
	for rows.Next() {
		user = &TestUser{}
		r.NoError(db.ScanRows(rows, user))
		r.True(user.ID > 0)
	}
}

func TestSqlQueryRaw(t *testing.T) {
	r := require.New(t)
	db := mustGetDB(true)
	defer orm.CloseDB(db)

	//插入测试记录
	users := mustInsertUsersForQuery(db)

	//db.Exec(): 执行CUD
	//db.Raw() : 执行Query
	//注：gorm提供的方法拼接sql时会根据不同的数据库驱动生成符合目标数据库的sql语句
	stmt := db.Exec(`update t_test_user set age=10 where id=1`)
	r.NoError(stmt.Error)
	r.EqualValues(1, stmt.RowsAffected)

	var rsUsers []*TestUser
	r.Error(db.Exec(`select * from t_test_user`).Scan(&rsUsers).Error) //exec()不支持执行select，将报错

	r.NoError(db.Raw(`select * from t_test_user`).Scan(&rsUsers).Error)
	r.EqualValues(len(users), len(rsUsers))

	//使用命名参数，也可传入map/struct
	//注意：默认命名参数名称区分大小写
	rsUsers = nil
	r.NoError(db.Raw(`select * from t_test_user where age>@min and age<@max`,
		sql.Named(`min`, 0), sql.Named(`max`, 10)).Scan(&rsUsers).Error)
	r.True(len(rsUsers) > 0)

	rsUsers = nil
	ageCond := struct {
		Min uint16 //不支持私有字段
		Max uint16
	}{1, 10}
	r.NoError(db.Raw(`select * from t_test_user where age>@Min and age<@Max`, ageCond).Scan(&rsUsers).Error)
	r.True(len(rsUsers) > 0)
}

func TestSqlQueryAdv(t *testing.T) {
	r := require.New(t)
	db := mustGetDB(true)
	defer orm.CloseDB(db)

	//插入测试记录
	_ = mustInsertUsersForQuery(db)

	type MiniUser struct {
		ID  int64
		Age uint16
	}
	//智能选择查询字段
	r.NoError(db.Find(&TestUser{}).Error)                                           //select *
	r.NoError(db.Model(&TestUser{}).Find(&MiniUser{}).Error)                        //select MiniUser全部字段
	r.NoError(db.Session(&gorm.Session{QueryFields: true}).Find(&TestUser{}).Error) //select TestUser全部字段

	//for update
	r.NoError(db.Clauses(clause.Locking{Strength: "UPDATE"}).Find(&TestUser{ID: 1}).Error)
	r.NoError(db.Clauses(orm.ExprForUpdate()).Find(&TestUser{ID: 1}).Error)

	//子查询
	//SELECT * FROM (SELECT id,name FROM `t_test_user` WHERE age>10) as t_user WHERE t_user.name like 'user%'
	subQuery := db.Model(&TestUser{}).Select(`id,name`).Where(`age>10`)
	r.NoError(db.Table(`(?) as t_user`, subQuery).Where(`t_user.name like ?`, `user%`).Find(&TestUser{}).Error)

	//复杂查询条件，建议使用db.Raw()
	//SELECT * FROM `t_test_user` WHERE `t_test_user`.`id` IN (1,2) AND (age<5 OR age>10)
	stmt := db.Where([]int{1, 2}).Where(db.Where(`age<5`).Or(`age>10`))
	r.NoError(stmt.Find(&TestUser{}).Error)

	//FirstOrInit()/FirstOrCreate()请参考官方文档，建议谨慎使用
	//后者可实现upsert/insert ignore等操作，但它是至少执行2条sql实现的，首先会执行1条查询sql进行判断
	//所以建议在事务里调用此方法，同时根据情况使用tx.Clause()添加for update查询条件
	//文档连接：https://gorm.io/zh_CN/docs/advanced_query.html#FirstOrInit
	r.NoError(db.Clauses(orm.ExprForUpdate()).FirstOrCreate(&TestUser{ID: 10}).Error)

	//批量操作
	mustInsertNUsersForQuery(db, 10, true)

	var rsUsers []*TestUser
	stmt = db.FindInBatches(&rsUsers, 2, func(tx *gorm.DB, batch int) error {
		//每次处理2条，共循环5次，batch值从1开始, tx.RowsAffected表示本次包含记录数
		//返回错误将终止下次处理
		if u := rsUsers[0]; u.ID == 1 {
			//使用tx执行sql自动添加本次全部记录的主键过滤条件
			u.Name = `bingo`

			//以下会自动添加过滤条件：WHERE (`id` = 1 OR `id` = 2)
			//如果直接使用save()不使用select()会报错，且添加id=1过滤条件避免本次所有记录
			return tx.Select(`name`).Where(u.ID).Save(u).Error
		}
		return nil
	})
	r.NoError(stmt.Error)

	user := &TestUser{ID: 1}
	r.NoError(db.First(user).Error)
	r.EqualValues(`bingo`, user.Name)

	///查询单例值
	var names []string
	r.NoError(db.Model(&TestUser{}).Pluck(`name`, &names).Error)
	r.True(len(names) > 0)

	//Scopes，自定义常用过滤条件,如翻页
	//软删除可调用db.Unscoped()取消过滤条件，但自定义Scope不能
	sexCond := func(db *gorm.DB) *gorm.DB {
		return db.Where(`sex=0`)
	}

	var rsUser1 []*TestUser
	var rsUser2 []*TestUser
	var rsUser3 []*TestUser
	r.NoError(db.Find(&rsUser1).Error)
	r.NoError(db.Scopes(sexCond).Find(&rsUser2).Error)
	r.NoError(db.Scopes(sexCond).Unscoped().Find(&rsUser3).Error) //并未取消sexCond过滤条件
	r.True(len(rsUser1) > 0)
	r.True(len(rsUser1) > len(rsUser2))
	r.True(len(rsUser2) == len(rsUser3))

	//count
	var count int64
	r.NoError(db.Model(&TestUser{}).Where(`sex=0`).Count(&count).Error)
	r.True(count > 0)

	count = 0
	r.NoError(db.Model(&TestUser{}).Distinct(`sex`).Count(&count).Error)
	r.True(count > 0)

	//可以先执行count，再执行分页查询(不能相反)
	stmt = db.Model(&TestUser{}).Select(`id`, `name`).Where(`id`, []int{1, 2, 3}).Order(`age desc`)

	count = 0
	r.NoError(stmt.Count(&count).Error)
	r.True(count > 0)

	r.NoError(stmt.Limit(2).Offset(1).Find(&rsUsers).Error)
	r.True(len(rsUsers) == 2)
}

func TestSqlOnConflict(t *testing.T) {
	r := require.New(t)
	db := mustGetDB(true)
	defer orm.CloseDB(db)

	mustInsertUsersForQuery(db)
	var rsUsers []*TestUser

	//save()根据传入的参数是否包含主键值判断执行更新或插入操作
	//save()执行更新操作会包括零值字段，建议使用select(),否则可能导致created_at等字段被更新为0
	r.NoError(db.Save(&TestUser{Name: `bingo`}).Error)                       //插入操作
	r.NoError(db.Select(`name`).Save(&TestUser{ID: 1, Name: `bingo`}).Error) //更新操作，使用select()避免更新其他字段
	r.NoError(db.Find(&rsUsers, `name=?`, `bingo`).Error)
	r.EqualValues(2, len(rsUsers))

	assertNameEqual := func(id uint64, name string) {
		user := &TestUser{ID: id}
		r.NoError(db.First(user).Error)
		r.EqualValues(name, user.Name)
	}

	//OnConflict()依赖Columns指定的列是否为主键、唯一键等判断是否有冲突
	//发生冲突不做任何操作
	db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: `id`}},
		DoNothing: true,
	}).Create(&TestUser{ID: 2})
	assertNameEqual(2, `user2`)

	db.Clauses(orm.ExprOnConflictDoNothing(`id`)).Create(&TestUser{ID: 2})
	assertNameEqual(2, `user2`)

	//发生冲突更新全部字段，包含零值字段
	db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: `id`}},
		UpdateAll: true,
	}).Create(&TestUser{ID: 2, Name: `u2`})
	assertNameEqual(2, `u2`)

	db.Clauses(orm.ExprOnConflictDoUpdate(`id`)).Create(&TestUser{ID: 2, Name: `u2`})
	assertNameEqual(2, `u2`)

	//发生冲突更新指定字段
	db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: `id`}},
		DoUpdates: clause.AssignmentColumns([]string{`name`}),
	}).Create(&TestUser{ID: 3, Name: `u3`})
	assertNameEqual(3, `u3`)

	db.Clauses(orm.ExprOnConflictDoUpdate(`id`, `name`)).Create(&TestUser{ID: 3, Name: `u3`})
	assertNameEqual(3, `u3`)

	//发生冲突更新指定字段
	db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: `id`}},
		DoUpdates: clause.Assignments(map[string]interface{}{`name`: `u33`}),
	}).Create(&TestUser{ID: 3, Name: `u3`})
	assertNameEqual(3, `u33`)

	//未发生冲突，name字段不是唯一键
	//mysql:执行更新操作，mysql会使用表的主键和所有唯一键判断冲突，指定的冲突判断列无效
	//pgsql:执行插入操作，但id冲突导致报错(待验证)
	db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: `name`}},
		DoUpdates: clause.AssignmentColumns([]string{`name`}),
	}).Create(&TestUser{ID: 4, Name: `u4`})
	assertNameEqual(4, `u4`)

	//OnConflict其他字段配置mysql不支持，需在pgsql上测试
	db.Clauses(clause.OnConflict{
		OnConstraint: `uk_name`, //指定判断冲突的唯一键(索引)
		DoNothing:    true,
	}).Create(&TestUser{ID: 4, Name: `u4`})

	//以下sql语句表示：
	// - 判断新插入记录是否与age>10的记录的name列发生冲突
	// - 如果是则更新发生冲突的且sex=0的记录的name为新插入记录的name值
	db.Clauses(clause.OnConflict{
		Columns:     []clause.Column{{Name: `name`}},
		Where:       clause.Where{Exprs: []clause.Expression{gorm.Expr(`age>10`)}}, //指定冲突记录过滤条件
		TargetWhere: clause.Where{Exprs: []clause.Expression{gorm.Expr(`sex=0`)}},  //指定要更新的冲突记录过滤条件
		DoUpdates:   clause.AssignmentColumns([]string{`name`}),
	}).Create(&TestUser{ID: 4, Name: `u4`})
}

func TestDBSession(t *testing.T) {
	r := require.New(t)
	db := mustGetDB(true)
	defer orm.CloseDB(db)

	mustInsertUsersForQuery(db)
	var rsUsers []*TestUser

	//首次调用链式方法会创建1个Statement，即开启1个会话
	stmt1 := db.Where(1)
	stmt2 := db.Where(1)
	r.True(stmt1 != stmt2)

	stmt3 := stmt1.Or(2) //继续使用前面创建Statement，即仍然处于当前会话
	r.True(stmt1 == stmt3)

	//执行Finisher方法，结束当前会话
	r.NoError(stmt1.Find(&rsUsers).Error)
	r.True(len(rsUsers) == 2) //会带上stmt3的clause，完整过滤条件：where id=1 or id=2

	//默认新建session包含父会话的clause
	r.NoError(db.Where(1).Session(&gorm.Session{}).Find(&rsUsers).Error)
	r.True(len(rsUsers) == 1 && rsUsers[0].ID == 1) //过滤条件：where id=1

	//设置NewDB==true可去掉父会话的clause
	r.NoError(db.Where(1).Session(&gorm.Session{NewDB: true}).Find(&rsUsers).Error)
	r.True(len(rsUsers) > 1) //过滤条件：无
}
