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
func (cc *Cache) LoadTable(ctx context.Context, req *pb.LoadTableReq) (*pb.LoadTableResp, error) {
	name := req.Name
	db := mpool.Get()

	data := &pb.TableConfig{Name: name}
	err := db.QueryRow("select version,content from gm_table where name=?", name).Scan(&data.Version, &data.Content)
	return &pb.LoadTableResp{File: data}, err
}

// 加载全部配置表
func (cc *Cache) LoadAllTable(req *pb.EmptyReq, stream pb.Cache_LoadAllTableServer) error {
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
func (cc *Cache) LoadScript(ctx context.Context, req *pb.LoadScriptReq) (*pb.LoadScriptResp, error) {
	name := req.Name
	db := mpool.Get()

	file := &pb.ScriptFile{Name: name}
	err := db.QueryRow("select body from gm_script where name=?", name).Scan(&file.Body)
	return &pb.LoadScriptResp{File: file}, err
}

// 加载全部配置表
func (cc *Cache) LoadAllScript(req *pb.EmptyReq, stream pb.Cache_LoadAllScriptServer) error {
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

// 查询字典
func (cc *Cache) QueryDict(ctx context.Context, req *pb.QueryDictReq) (*pb.QueryDictResp, error) {
	db := mpool.Get()
	resp := &pb.QueryDictResp{}
	db.QueryRow("select `value` from dict where `key`=?", req.Key).Scan(&resp.Value)
	return resp, nil
}

// 更新字典
func (cc *Cache) UpdateDict(ctx context.Context, req *pb.UpdateDictReq) (*pb.EmptyResp, error) {
	db := mpool.Get()
	db.Exec("insert ignore into dict(`key`,`value`) values(?,?)", req.Key, req.Value)
	db.Exec("update dict set `value`=? where `key`=?", req.Value, req.Key)
	return &pb.EmptyResp{}, nil
}
