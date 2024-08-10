// database pool
package dbo

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const (
	defaultIdleConns    = 100
	defaultOpenConns    = 200
	defaultConnLifeTime = 1800 // MySQL默认8小时
)

type Pool struct {
	db         *gorm.DB
	User       string
	Password   string
	Addr       string
	SchemaName string
}

func (dbPool *Pool) SetSource(user, password, addr, dbname string) {
	dbPool.User = user
	dbPool.Password = password
	dbPool.Addr = addr
	dbPool.SchemaName = dbname
}

func (p *Pool) Get() *gorm.DB {
	if p.db != nil {
		return p.db
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", p.User, p.Password, p.Addr, p.SchemaName)
	ormDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{NamingStrategy: schema.NamingStrategy{SingularTable: true}})
	if err != nil {
		panic(err)
	}

	db, _ := ormDB.DB()
	db.SetMaxIdleConns(1)
	if err := ormDB.Exec("create database if not exists " + p.SchemaName).Error; err != nil {
		panic(err)
	}
	if err := ormDB.Exec("use " + p.SchemaName).Error; err != nil {
		panic(err)
	}
	db.SetMaxIdleConns(defaultIdleConns)
	db.SetMaxOpenConns(defaultOpenConns)
	db.SetConnMaxLifetime(defaultConnLifeTime * time.Second)
	p.db = ormDB
	return p.db
}

func NewPool() *Pool {
	p := &Pool{}
	return p
}

var dbPool = NewPool()

func SetSource(user, password, addr, dbname string) {
	dbPool.SetSource(user, password, addr, dbname)
}

func Get() *gorm.DB {
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
