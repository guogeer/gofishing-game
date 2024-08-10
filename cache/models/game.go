package models

import (
	"database/sql"
	"time"
)

type UserInfo struct {
	Id             int    `gorm:"primaryKey;autoIncrement"`
	Nickname       string `gorm:"type:varchar(50)"`
	Sex            int
	Icon           string `gorm:"type:varchar(255)"`
	PlateIcon      string `gorm:"type:varchar(255)"`
	TimeZone       float32
	Email          string    `gorm:"type:varchar(64)"`
	Ip             string    `gorm:"type:varchar(32)"`
	ClientVersion  string    `gorm:"type:varchar(32)"`
	Mac            string    `gorm:"type:varchar(24)"`
	Imei           string    `gorm:"type:varchar(24)"`
	Imsi           string    `gorm:"type:varchar(24)"`
	ChanId         string    `gorm:"type:varchar(32)"`
	ServerLocation string    `gorm:"type:varchar(32)"`
	CreateTime     time.Time `gorm:"autoCreateTime"`
}

type ItemLog struct {
	Id         int `gorm:"primaryKey;autoIncrement"`
	Uid        int
	ItemId     int
	Way        string `gorm:"type:varchar(64)"`
	Num        int
	Balance    int
	Uuid       string    `gorm:"type:varchar(64)"`
	CreateTime time.Time `gorm:"autoCreateTime"`
}

type OnlineLog struct {
	Id            int    `gorm:"primaryKey;autoIncrement"`
	Uid           int    `gorm:"index:uid_date_idx,unique"`
	CurDate       string `gorm:"index:uid_date_idx,unique"`
	Ip            string `gorm:"type:varchar(32)"`
	ClientVersion string `gorm:"type:varchar(32)"`
	Mac           string `gorm:"type:varchar(24)"`
	Imei          string `gorm:"type:varchar(24)"`
	Imsi          string `gorm:"type:varchar(24)"`
	ChanId        string `gorm:"type:varchar(32)"`
	LoginTime     time.Time
	OfflineTime   sql.NullTime
}

type UserPlate struct {
	Id         int `gorm:"primaryKey;autoIncrement"`
	Uid        int
	Plate      string    `gorm:"type:varchar(16)"`
	OpenId     string    `gorm:"type:varchar(48);uniqueIndex"`
	CreateTime time.Time `gorm:"autoCreateTime"`
}

type UserBin struct {
	Id         int       `gorm:"primaryKey;autoIncrement"`
	Uid        int       `gorm:"index:user_bin_idx,unique"`
	Class      string    `gorm:"type:varchar(16);index:user_bin_idx,unique"`
	Bin        []byte    `gorm:"type:blob"`
	UpdateTime time.Time `gorm:"autoUpdateTime"`
}

type Mail struct {
	Id       int `gorm:"primaryKey;autoIncrement"`
	Type     int
	SendUid  int
	RecvUid  int `gorm:"index"`
	Status   int
	Title    string    `gorm:"type:varchar(64)"`
	Body     string    `gorm:"type:text"`
	SendTime time.Time `gorm:"autoCreateTime"`
}

type Dict struct {
	Id         int       `gorm:"primaryKey;autoIncrement"`
	Key        string    `gorm:"type:varchar(32);index"`
	Value      string    `gorm:"type:json"`
	UpdateTime time.Time `gorm:"autoUpdateTime"`
}

type Table struct {
	Id         int    `gorm:"primaryKey;autoIncrement"`
	Name       string `gorm:"type:varchar(32);index"`
	Version    int
	Content    string    `gorm:"type:text"`
	UpdateTime time.Time `gorm:"onUpdateTime"`
}

type Script struct {
	Id         int       `gorm:"primaryKey;autoIncrement"`
	Name       string    `gorm:"type:varchar(32);index"`
	Body       string    `gorm:"type:text"`
	UpdateTime time.Time `gorm:"onUpdateTime"`
}

type ClientVersion struct {
	Id         int       `gorm:"primaryKey;autoIncrement"`
	ChanId     string    `gorm:"type:varchar(32)"`
	Version    string    `gorm:"type:varchar(16)"`
	AllowIP    string    `gorm:"type:varchar(64)"`
	AllowUid   string    `gorm:"type:varchar(64)"`
	ChangeLog  string    `gorm:"type:text"`
	Reward     string    `gorm:"type:varchar(64)"`
	UpdateTime time.Time `gorm:"onUpdateTime"`
}
