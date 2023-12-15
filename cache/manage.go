package cache

// GM管理后台

import (
	"context"

	"gofishing-game/internal/dbo"
	"gofishing-game/internal/env"
	"gofishing-game/internal/pb"
)

var mpool = dbo.NewPool()

func init() {
	t := env.Config().ManageDataSource
	mpool.SetSource(t.User, t.Password, t.Addr, t.Name)

	db := mpool.Get()
	if n := t.MaxIdleConns; n > 0 {
		db.SetMaxIdleConns(n)
	}
	if n := t.MaxOpenConns; n > 0 {
		db.SetMaxOpenConns(n)
	}
}

// 加载配置表
func (cc *Cache) LoadTable(ctx context.Context, req *pb.Request) (*pb.TableConfig, error) {
	name := req.Name
	db := mpool.Get()

	data := &pb.TableConfig{Name: name}
	err := db.QueryRow("select version,content from gm_table where name=?", name).Scan(&data.Version, &data.Content)
	return data, err
}

// 加载全部配置表
func (cc *Cache) LoadAllTable(req *pb.Request, stream pb.Cache_LoadAllTableServer) error {
	db := mpool.Get()

	rs, err := db.Query("select version,name,content from gm_table")
	if err != nil {
		return err
	}
	defer rs.Close()

	for rs.Next() {
		t := &pb.TableConfig{}
		rs.Scan(&t.Version, &t.Name, &t.Content)
		if err := stream.Send(t); err != nil {
			return err
		}
	}
	return nil
}

// 加载脚本
func (cc *Cache) LoadScript(ctx context.Context, req *pb.Request) (*pb.ScriptFile, error) {
	name := req.Name
	db := mpool.Get()

	data := &pb.ScriptFile{Name: name}
	err := db.QueryRow("select body from gm_script where name=?", name).Scan(&data.Body)
	return data, err
}

// 加载全部配置表
func (cc *Cache) LoadAllScript(req *pb.Request, stream pb.Cache_LoadAllScriptServer) error {
	db := mpool.Get()

	rs, err := db.Query("select name,body from gm_script")
	if err != nil {
		return err
	}
	defer rs.Close()

	for rs.Next() {
		t := &pb.ScriptFile{}
		rs.Scan(&t.Name, &t.Body)
		if err := stream.Send(t); err != nil {
			return err
		}
	}
	return nil
}

// 获取所有的配置信息
func (cc *Cache) GetAllClientVersion(ctx context.Context, req *pb.ClientVersion) (*pb.LastVersion, error) {
	db := mpool.Get()
	rows, _ := db.Query("select chan_id,version,json_value from gm_client_version")
	var versions []*pb.WaitUpdateVersion
	for rows != nil && rows.Next() {
		v := &pb.WaitUpdateVersion{}
		rows.Scan(&v.ChanId, &v.Version, dbo.JSON(v))
		versions = append(versions, v)
	}
	return &pb.LastVersion{WaitUpdateVersion: versions}, nil
}

// 查询字典
func (cc *Cache) QueryDictValue(ctx context.Context, req *pb.DictValue) (*pb.DictValue, error) {
	db := mpool.Get()
	resp := &pb.DictValue{Key: req.Key}
	db.QueryRow("select `value` from dict where `key`=?", req.Key).Scan(&resp.Value)
	return resp, nil
}

// 更新字典
func (cc *Cache) UpdateDictValue(ctx context.Context, req *pb.DictValue) (*pb.DictValue, error) {
	db := mpool.Get()
	resp := &pb.DictValue{Key: req.Key}
	db.Exec("insert ignore into dict(`key`,`value`) values(?,?)", req.Key, req.Value)
	db.Exec("update dict set `value`=? where `key`=?", req.Value, req.Key)
	return resp, nil
}
