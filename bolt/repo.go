package bolt

import (
	"b-go-util/_string"
	"b-go-util/util"
	"bytes"
	bolt "go.etcd.io/bbolt"
	"strconv"
)

type Repository struct {
	db         *bolt.DB
	bucketName string
}

func NewRepository(db *bolt.DB, bucketName string) *Repository {
	util.AssertOk(db != nil, "db is nil")
	util.AssertOk(!_string.Empty(bucketName), "bucketName is empty")

	r := &Repository{
		db:         db,
		bucketName: bucketName,
	}

	if err := r.CreateBucketIfNotExists(); err != nil {
		panic(util.NewDBError(err, "create bucket '%s' error", bucketName))
	}

	return r
}

func (r *Repository) DB() *bolt.DB {
	return r.db
}

func (r *Repository) BucketName() string {
	return r.bucketName
}

func (r *Repository) Bucket(tx *bolt.Tx) *bolt.Bucket {
	return tx.Bucket([]byte(r.bucketName))
}

func (r *Repository) Update(fn func(bucket *bolt.Bucket) error) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return fn(r.Bucket(tx))
	})
}

func (r *Repository) View(fn func(bucket *bolt.Bucket) error) error {
	return r.db.View(func(tx *bolt.Tx) error {
		return fn(r.Bucket(tx))
	})
}

func (r *Repository) Batch(fn func(bucket *bolt.Bucket) error) error {
	return r.db.Batch(func(tx *bolt.Tx) error {
		return fn(r.Bucket(tx))
	})
}

func (r *Repository) Del(key string) error {
	return r.Update(func(bucket *bolt.Bucket) error {
		return bucket.Delete([]byte(key))
	})
}

func (r *Repository) Put(key string, val []byte) error {
	return r.Update(func(bucket *bolt.Bucket) error {
		return bucket.Put([]byte(key), val)
	})
}

func (r *Repository) PutString(key, val string) error {
	return r.Put(key, []byte(val))
}

func (r *Repository) PutBool(key string, val bool) error {
	return r.PutString(key, strconv.FormatBool(val))
}

func (r *Repository) PutInt(key string, val int) error {
	return r.PutString(key, strconv.Itoa(val))
}

func (r *Repository) PutInt64(key string, val int64) error {
	return r.PutString(key, strconv.FormatInt(val, 10))
}

//key对应的值必须为整数，否则将报错
func (r *Repository) Incr(key string, n int64) (int64, error) {
	err := r.Update(func(bucket *bolt.Bucket) error {
		if val := bucket.Get([]byte(key)); val != nil {
			v, err := strconv.ParseInt(string(val), 10, 64)

			if err != nil {
				return err
			}

			n += v
		}

		return bucket.Put([]byte(key), []byte(strconv.FormatInt(n, 10)))
	})

	if err != nil {
		return 0, err
	}

	return n, nil
}

func (r *Repository) BatchPut(key string, val []byte) error {
	return r.Batch(func(bucket *bolt.Bucket) error {
		return bucket.Put([]byte(key), val)
	})
}

func (r *Repository) BatchPutString(key, val string) error {
	return r.BatchPut(key, []byte(val))
}

func (r *Repository) BatchPutBool(key string, val bool) error {
	return r.BatchPutString(key, strconv.FormatBool(val))
}

func (r *Repository) BatchPutInt(key string, val int) error {
	return r.BatchPutString(key, strconv.Itoa(val))
}

func (r *Repository) BatchPutInt64(key string, val int64) error {
	return r.BatchPutString(key, strconv.FormatInt(val, 10))
}

func (r *Repository) Has(key string) (bool, error) {
	var exist bool

	err := r.View(func(bucket *bolt.Bucket) error {
		if bucket.Get([]byte(key)) != nil {
			exist = true
		}

		return nil
	})

	if err != nil {
		return false, err
	}

	return exist, err
}

func (r *Repository) HasOrDefault(key string, df bool) (bool, error) {
	if exist, err := r.Has(key); err == nil {
		return exist, nil
	}

	return df, nil
}

func (r *Repository) Get(key string) ([]byte, error) {
	var data []byte

	err := r.View(func(bucket *bolt.Bucket) error {
		if val := bucket.Get([]byte(key)); val != nil {
			data = make([]byte, len(val))
			copy(data, val)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return data, err
}

//GetXX() 如果key不存在，返回默认零值，而不是返回错误
func (r *Repository) GetString(key string) (string, error) {
	if val, err := r.Get(key); err != nil {
		return "", err
	} else {
		return string(val), nil
	}
}

func (r *Repository) GetBool(key string) (bool, error) {
	if val, err := r.GetString(key); err != nil {
		return false, err
	} else {
		return strconv.ParseBool(val)
	}
}

func (r *Repository) GetInt(key string) (int, error) {
	if val, err := r.GetString(key); err != nil {
		return 0, err
	} else {
		return strconv.Atoi(val)
	}
}

func (r *Repository) GetInt64(key string) (int64, error) {
	if val, err := r.GetString(key); err != nil {
		return 0, err
	} else {
		return strconv.ParseInt(val, 10, 64)
	}
}

func (r *Repository) GetStringOrDefault(key, df string) string {
	if val, err := r.GetString(key); err == nil {
		return val
	}

	return df
}

func (r *Repository) GetBoolOrDefault(key string, df bool) bool {
	if val, err := r.GetBool(key); err == nil {
		return val
	}

	return df
}

func (r *Repository) GetIntOrDefault(key string, df int) int {
	if val, err := r.GetInt(key); err == nil {
		return val
	}

	return df
}

func (r *Repository) GetInt64OrDefault(key string, df int64) int64 {
	if val, err := r.GetInt64(key); err == nil {
		return val
	}

	return df
}

func (r *Repository) DelBucket() error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte(r.bucketName))
	})
}

func (r *Repository) CreateBucketIfNotExists() error {
	return r.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(r.bucketName))
		return err
	})
}

func (r *Repository) RecreateBucket() error {
	return r.db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte(r.bucketName))

		if err == nil {
			_, err = tx.CreateBucket([]byte(r.bucketName))
		}

		return err
	})
}

func (r *Repository) KeySize() (int, error) {
	return GetBucketKeySize(r.db, r.bucketName)
}

//获取此bucket对应的下1个序列值
func (r *Repository) NextId() (uint64, error) {
	var id uint64
	err := r.Update(func(bucket *bolt.Bucket) error {
		//根据官方文档，在Update()里调用NextSequence()不会返回错误，因此忽略
		id, _ = bucket.NextSequence()
		return nil
	})

	if err != nil {
		return 0, err
	}

	return id, nil
}

func (r *Repository) NextIdInt64() (int64, error) {
	if id, err := r.NextId(); err != nil {
		return 0, err
	} else {
		return int64(id), nil
	}
}

func (r *Repository) NextIdString() (string, error) {
	if id, err := r.NextIdInt64(); err != nil {
		return "", err
	} else {
		return strconv.FormatInt(id, 10), nil
	}
}

//迭代查询每条记录，直到fn返回false或err不为空
//如果fn要保存k，v到外部变量，则需要copy()，具体请阅读bucket.ForEach()注释
func (r *Repository) ForEach(fn func(k, v []byte) (bool, error), prefix ...string) error {
	return r.View(func(bucket *bolt.Bucket) error {
		return r.iterateCursor(fn, bucket.Cursor(), prefix...)
	})
}

//删除每条记录，直到fn返回false或err不为空
func (r *Repository) DelEach(fn func(k, v []byte) (bool, error), prefix ...string) error {
	return r.Update(func(bucket *bolt.Bucket) error {
		c := bucket.Cursor()
		return r.iterateCursor(func(k, v []byte) (bool, error) {
			if ok, err := fn(k, v); ok && err == nil {
				return true, c.Delete()
			} else {
				return ok, err
			}
		}, c, prefix...)
	})
}

func (r *Repository) iterateCursor(fn func(k, v []byte) (bool, error), c *bolt.Cursor, prefix ...string) error {
	if len(prefix) == 0 || _string.Empty(prefix[0]) {
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if ok, err := fn(k, v); !ok || err != nil {
				return err
			}
		}
	} else {
		pfx := []byte(prefix[0])
		for k, v := c.Seek(pfx); k != nil && bytes.HasPrefix(k, pfx); k, v = c.Next() {
			if ok, err := fn(k, v); !ok || err != nil {
				return err
			}
		}
	}

	return nil
}
