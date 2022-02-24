package bolt

import (
	"bytes"
	"fmt"
	ubolt "github.com/bingooh/b-go-util/bolt"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
	"os"
	"sync"
	"testing"
	"time"
)

func mustNewDb() (db *bolt.DB, clear func()) {
	option := &ubolt.Option{
		DbFilePath: "./test.db",
		Timeout:    1 * time.Minute,
	}

	db = ubolt.MustNewDb(option)
	clear = func() {
		if err := db.Close(); err != nil {
			fmt.Printf("remove db err: %v\n", err)
		}

		if err := os.Remove(option.DbFilePath); err != nil {
			fmt.Printf("remove db file err: %v\n", err)
		}
	}

	return
}

func TestRepo(t *testing.T) {
	r := require.New(t)
	db, clear := mustNewDb()
	defer clear()

	repo := ubolt.NewRepository(db, "t_test")

	assertKeySizeEqual := func(expect interface{}) {
		size, err := repo.KeySize()
		r.NoError(err, "get key size should no err")
		r.Equal(expect, size, "key size should equal")
	}

	err := repo.RecreateBucket()
	r.NoError(err, "create bucket should no err")
	assertKeySizeEqual(0)

	r.NoError(repo.PutString("kstr", "v1"), "put string should no err")
	r.NoError(repo.PutBool("kbool", true), "put bool should no err")
	r.NoError(repo.PutInt("kint", 1), "put int should no err")
	r.NoError(repo.PutInt64("kint64", int64(10)), "put int64 should no err")
	assertKeySizeEqual(4)

	exist, err := repo.Has("kstr")
	r.NoError(err, "has should no err")
	r.True(exist, "has should be true")

	exist, err = repo.Has("knokey")
	r.NoError(err, "has should no err")
	r.False(exist, "has should be false")

	vstr, err := repo.GetString("kstr")
	r.NoError(err, "get string should no err")
	r.Equal("v1", vstr, "get string should equal")

	vbool, err := repo.GetBool("kbool")
	r.NoError(err, "get bool should no err")
	r.Equal(true, vbool, "get bool should equal")

	vint, err := repo.GetInt("kint")
	r.NoError(err, "get int should no err")
	r.Equal(1, vint, "get int should equal")

	vint64, err := repo.GetInt64("kint64")
	r.NoError(err, "get int64 should no err")
	r.Equal(int64(10), vint64, "get int64 should equal")

	n, err := repo.Incr("incr", 1)
	r.NoError(err, "incr should no err")
	r.EqualValues(1, n, "incr should equal")

	n, err = repo.Incr("incr", 10)
	r.NoError(err, "incr should no err")
	r.EqualValues(11, n, "incr should equal")

	n, err = repo.Incr("incr", -2)
	r.NoError(err, "incr should no err")
	r.EqualValues(9, n, "incr should equal")

	err = repo.RecreateBucket()
	r.NoError(err, "create bucket should no err")
	assertKeySizeEqual(0)

	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				err := repo.BatchPutInt(fmt.Sprintf("k-%v-%v", i, j), j)
				r.NoError(err, "batch put int should no err")
			}
		}(i)
	}
	wg.Wait()
	assertKeySizeEqual(100)

	//删除k-0-xx的记录，共10条
	err = repo.DelEach(func(k, v []byte) (bool, error) {
		if bytes.HasPrefix(k, []byte("k-0")) {
			return true, nil
		}

		return false, nil
	})
	r.NoError(err, "del each should no err")
	assertKeySizeEqual(90)
}
