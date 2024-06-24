package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/guogeer/quasar/v2/utils"

	"gofishing-game/internal/pb"
	"gofishing-game/internal/rpc"
	"gofishing-game/service"

	"github.com/guogeer/quasar/v2/cmd"
	"github.com/guogeer/quasar/v2/config"
	"github.com/guogeer/quasar/v2/log"
)

// 版本更新奖励
type ClientVersion struct {
	// 版本 1.12.123.1234_r,1.12.123.1234_d,1.12.123.1234
	Version string `json:"version"`
	// 奖励 [[1102,1000],[1104,2000]]
	Reward string `json:"reward"`
	ChanId string `json:"chan_id"`
	Title  string `json:"title"`
	Body   string `json:"change_log"`
}

type buildRankItem struct {
	Uid        int
	Nickname   string
	Icon       string
	BuildLevel int
	BuildExp   int
}

type hallWorld struct {
	currentBestGateway string

	onlines   map[string]service.ServerOnline
	buildRank []buildRankItem
	massMails []*Mail // 系统群发的邮件

	maintain struct {
		StartTime, EndTime time.Time
		notifyTimer        *utils.Timer
	}
	segments []service.GameOnlineSegment
}

func GetWorld() *hallWorld {
	return service.GetWorld((*hallWorld)(nil).GetName()).(*hallWorld)
}

func init() {
	w := &hallWorld{
		onlines: make(map[string]service.ServerOnline),
	}

	service.AddWorld(w)

	startTime, _ := config.ParseTime("2010-01-01")
	utils.NewPeriodTimer(w.UpdateOnline, startTime, 5*time.Minute)
	//utils.NewPeriodTimer(hall.tick1m, startTime, 1*time.Minute)
	//utils.NewPeriodTimer(hall.tick1d, startTime, 24*time.Hour)

	w.updateBuildRank()
	w.updateMaintain()

	// 加载群发邮件
	resp, _ := rpc.CacheClient().QuerySomeMail(context.Background(),
		&pb.QuerySomeMailReq{Type: MailTypeMass, Num: 999})
	if resp == nil {
		log.Fatal("load mass mail fail")
	}
	for _, pbMail := range resp.Mails {
		mail := &Mail{}
		utils.DeepCopy(mail, pbMail)
	}
}

func (w *hallWorld) GetName() string {
	return "hall"
}

func (w *hallWorld) NewPlayer() *service.Player {
	p := &hallPlayer{}
	p.Player = service.NewPlayer(p)
	p.DataObj().Push(p)

	p.mailObj = newMailObj(p)
	return p.Player
}

func (w *hallWorld) GetCurrentOnline() []service.ServerOnline {
	data := make([]service.ServerOnline, 0, 16)
	for _, g := range w.onlines {
		data = append(data, g)
	}
	return data
}

func (w *hallWorld) UpdateOnline() {
	data := w.GetCurrentOnline()
	if len(data) == 0 {
		return
	}
	cmd.Forward("plate", "reportOnline", cmd.M{"servers": data})
}

func GetPlayer(uid int) *hallPlayer {
	if p := service.GetPlayer(uid); p != nil {
		return p.GameAction.(*hallPlayer)
	}
	return nil
}

func (w *hallWorld) GetBestGateway() string {
	return w.currentBestGateway
}

func (w *hallWorld) updateBuildRank() {
	go func() {
		dict, _ := rpc.CacheClient().QueryDict(context.Background(), &pb.QueryDictReq{Key: "build_rank"})
		rpc.OnResponse(func() {
			json.Unmarshal(dict.Value, &w.buildRank)
		})
	}()
}

func (w *hallWorld) notifyMaintain() {
	secs := time.Until(w.maintain.StartTime).Seconds()
	msg := cmd.M{
		"content": "The game will be temporarily closed after {seconds} s.",
		"startTs": w.maintain.StartTime.Unix(),
		"endTs":   w.maintain.EndTime.Unix(),
	}
	if secs > 60 {
		msg["content"] = "The game will be temporarily closed after {clock}."

		utils.StopTimer(w.maintain.notifyTimer)
		w.maintain.notifyTimer = utils.NewTimer(w.notifyMaintain, time.Until(w.maintain.StartTime)-time.Minute)
	}
	log.Infof("maintain broadcast msg %s", msg["content"])
	service.Broadcast2Gateway("maintain", msg)
}

func (w *hallWorld) updateMaintain() {
	go func() {
		dict, err := rpc.CacheClient().QueryDict(context.Background(), &pb.QueryDictReq{Key: "maintain"})
		if err != nil {
			return
		}

		pbData := &pb.Maintain{}
		json.Unmarshal([]byte(dict.Value), pbData)
		startTime, _ := config.ParseTime(pbData.StartTime)
		endTime, _ := config.ParseTime(pbData.EndTime)
		rpc.OnResponse(func() {
			utils.StopTimer(w.maintain.notifyTimer)
			w.maintain.StartTime = startTime
			w.maintain.EndTime = endTime
			log.Infof("maintain game at %s-%s content %s allow list %s", pbData.StartTime, pbData.EndTime, pbData.Content, pbData.AllowList)

			if d := time.Until(startTime); d > 5*time.Minute {
				w.maintain.notifyTimer = utils.NewTimer(w.notifyMaintain, d-5*time.Minute)
			} else if d > time.Minute {
				w.maintain.notifyTimer = utils.NewTimer(w.notifyMaintain, -time.Minute)
			}
		})
	}()
}
