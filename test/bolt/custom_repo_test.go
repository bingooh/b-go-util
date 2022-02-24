package bolt

import (
	"encoding/json"
	"fmt"
	"github.com/bingooh/b-go-util/_string"
	ubolt "github.com/bingooh/b-go-util/bolt"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
	"strconv"
	"testing"
)

type User struct {
	Id   string
	Name string
}

//存放user对象，使用json序列化
type UserRepository struct {
	*ubolt.Repository
}

func NewUserRepository(db *bolt.DB) *UserRepository {
	r := ubolt.NewRepository(db, "t_user")
	return &UserRepository{r}
}

func (r *UserRepository) PutUser(user *User) error {
	if user == nil {
		return fmt.Errorf("user is nil")
	}

	if _string.Empty(user.Id) {
		if id, err := r.NextIdInt64(); err != nil {
			return err
		} else {
			user.Id = strconv.FormatInt(id, 10)
		}
	}

	if userJson, err := json.Marshal(user); err != nil {
		return err
	} else {
		return r.Put(user.Id, userJson)
	}
}

func (r *UserRepository) GetUser(id string) (*User, error) {
	if _string.Empty(id) {
		return nil, fmt.Errorf("user id is nil")
	}

	if userJson, err := r.Get(id); err != nil || userJson == nil {
		return nil, err
	} else {
		user := new(User)
		if err := json.Unmarshal(userJson, user); err != nil {
			return nil, err
		}

		return user, nil
	}
}

func TestCustomRepo(t *testing.T) {
	r := require.New(t)
	db, clear := mustNewDb()
	defer clear()

	repo := NewUserRepository(db)

	err := repo.RecreateBucket()
	r.NoError(err, "create bucket should no err")

	err = repo.PutUser(&User{Name: "user"})
	r.NoError(err, "put user(no id) should no err")

	err = repo.PutUser(&User{Id: "user1", Name: "user1"})
	r.NoError(err, "put user(with id) should no err")

	err = repo.PutUser(&User{Id: "user1", Name: "user1"})
	r.NoError(err, "put user(with id) should no err")

	size, err := repo.KeySize()
	r.NoError(err, "get size should no err")
	r.EqualValues(2, size, "key size should equal")

	user, err := repo.GetUser("user1")
	r.NoError(err, "get user should no err")
	r.Equal("user1", user.Id, "user.id should equal")
	r.Equal("user1", user.Name, "user.name should equal")
}
