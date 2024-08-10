package main

// GM管理后台

import (
	"context"

	"gofishing-game/cache/models"
	"gofishing-game/internal/dbo"
	"gofishing-game/internal/pb"

	"gorm.io/gorm/clause"
)

// 加载配置表
func (cc *Cache) LoadTable(ctx context.Context, req *pb.LoadTableReq) (*pb.LoadTableResp, error) {
	db := dbo.Get()

	var table models.Table
	err := db.Where("name=?", req.Name).Take(&table).Error
	return &pb.LoadTableResp{File: &pb.TableConfig{
		Name:    req.Name,
		Content: table.Content,
		Version: int32(table.Version),
	}}, err
}

// 加载全部配置表
func (cc *Cache) LoadAllTable(req *pb.EmptyReq, stream pb.Cache_LoadAllTableServer) error {
	db := dbo.Get()

	var tables []models.Table
	db.Find(&tables)
	for _, table := range tables {
		if err := stream.Send(&pb.TableConfig{Version: int32(table.Version), Name: table.Name, Content: table.Content}); err != nil {
			return err
		}
	}
	return nil
}

// 加载脚本
func (cc *Cache) LoadScript(ctx context.Context, req *pb.LoadScriptReq) (*pb.LoadScriptResp, error) {
	db := dbo.Get()

	var script models.Script
	err := db.Where("name=?", req.Name).Take(&script).Error
	return &pb.LoadScriptResp{File: &pb.ScriptFile{Name: req.Name, Body: script.Body}}, err
}

// 加载全部配置表
func (cc *Cache) LoadAllScript(req *pb.EmptyReq, stream pb.Cache_LoadAllScriptServer) error {
	db := dbo.Get()

	var scripts []models.Script
	db.Find(&scripts)
	for _, script := range scripts {
		if err := stream.Send(&pb.ScriptFile{Name: script.Name, Body: script.Body}); err != nil {
			return err
		}
	}
	return nil
}

// 查询字典
func (cc *Cache) QueryDict(ctx context.Context, req *pb.QueryDictReq) (*pb.QueryDictResp, error) {
	db := dbo.Get()

	var dict models.Dict
	db.Where("key=?", req.Key).Take(&dict)
	return &pb.QueryDictResp{Value: []byte(dict.Value)}, nil
}

// 更新字典
func (cc *Cache) UpdateDict(ctx context.Context, req *pb.UpdateDictReq) (*pb.EmptyResp, error) {
	db := dbo.Get()

	err := db.Model(models.Dict{}).Clauses(&clause.OnConflict{UpdateAll: true}).Create(&models.Dict{Key: req.Key, Value: string(req.Value)}).Error
	return &pb.EmptyResp{}, err
}
