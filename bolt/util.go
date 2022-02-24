package bolt

import (
	"github.com/bingooh/b-go-util/_string"
	"github.com/bingooh/b-go-util/util"
	bolt "go.etcd.io/bbolt"
)

func MustCreateBucketIfNotExists(db *bolt.DB, bucketName string) {
	util.AssertOk(db != nil, "db is nil")
	util.AssertOk(!_string.Empty(bucketName), "bucketName is nil")

	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err
	})

	if err != nil {
		panic(util.NewDBError(err, "create bucket '%s' error", bucketName))
	}
}

func GetBucketKeySize(db *bolt.DB, bucketName string) (int, error) {
	size := 0

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		size = b.Stats().KeyN
		return nil
	})

	if err != nil {
		return 0, util.NewDBError(err, "get bucket '%v' key size err", bucketName)
	}

	return size, nil
}
