// database pool
package dbo

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"google.golang.org/protobuf/proto"
)

const defaultIdleConns = 100
const defaultOpenConns = 200
const defaultConnLifeTime = 1800 // MySQL默认8小时

type Pool struct {
	DBs      *sql.DB
	User     string
	Password string
	Addr     string
	DbName   string
}

func (dbPool *Pool) SetSource(user, password, addr, dbname string) {
	dbPool.User = user
	dbPool.Password = password
	dbPool.Addr = addr
	dbPool.DbName = dbname
}

func (p *Pool) Get() *sql.DB {
	if p.DBs != nil {
		return p.DBs
	}
	s := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&loc=Local", p.User, p.Password, p.Addr, p.DbName)
	db, err := sql.Open("mysql", s)
	if err != nil {
		panic(err.Error())
	}
	db.SetMaxIdleConns(defaultIdleConns)
	db.SetMaxOpenConns(defaultOpenConns)
	db.SetConnMaxLifetime(defaultConnLifeTime * time.Second)
	p.DBs = db
	return p.DBs
}

func NewPool() *Pool {
	p := &Pool{}
	return p
}

var dbPool = NewPool()

func SetSource(user, password, addr, dbname string) {
	dbPool.SetSource(user, password, addr, dbname)
}

func Get() *sql.DB {
	return dbPool.Get()
}

type jsonValue struct {
	ptr any
}

func (jv *jsonValue) Scan(value any) error {
	if value != nil {
		if buf, ok := value.([]byte); ok {
			return json.Unmarshal(buf, jv.ptr)
		}
	}
	return nil
}

func (jv *jsonValue) Value() (driver.Value, error) {
	return json.Marshal(jv.ptr)
}

func JSON(ptr any) *jsonValue {
	return &jsonValue{ptr: ptr}
}

type pbValue struct {
	ptr proto.Message
}

func (pv *pbValue) Scan(value any) error {
	if value != nil {
		if buf, ok := value.([]byte); ok {
			return proto.Unmarshal(buf, pv.ptr)
		}
	}
	return nil
}

func (pv *pbValue) Value() (driver.Value, error) {
	return proto.Marshal(pv.ptr)
}

func PB(ptr proto.Message) *pbValue {
	return &pbValue{ptr: ptr}
}
