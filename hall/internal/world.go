package internal

import (
	"context"
	"encoding/json"
	"time"

	"gofishing-game/internal/pb"
	"gofishing-game/internal/rpc"
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"
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

	onlines   map[int]service.ClientOnline
	buildRank []buildRankItem
	massMails []*Mail // 系统群发的邮件

	maintain struct {
		StartTime, EndTime time.Time
		notifyTimer        *util.Timer
	}
	segments []service.GameOnlineSegment
}

func GetWorld() *hallWorld {
	return service.GetWorld().(*hallWorld)
}

func init() {
	w := &hallWorld{
		onlines: make(map[int]service.ClientOnline),
	}

	service.CreateWorld(w)

	startTime, _ := config.ParseTime("2010-01-01")
	util.NewPeriodTimer(w.UpdateOnline, startTime, 5*time.Minute)
	//util.NewPeriodTimer(hall.tick1m, startTime, 1*time.Minute)
	//util.NewPeriodTimer(hall.tick1d, startTime, 24*time.Hour)

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
		util.DeepCopy(mail, pbMail)
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

func (w *hallWorld) GetCurrentOnline() []*pb.SubGame {
	data := make([]*pb.SubGame, 0, 16)
	for subId, g := range w.onlines {
		data = append(data, &pb.SubGame{ServerName: g.ServerName, Id: int32(subId), Num: int32(g.Online)})
	}
	return data
}

func (w *hallWorld) UpdateOnline() {
	data := w.GetCurrentOnline()
	if len(data) == 0 {
		return
	}
	cmd.Forward("plate", "ReportOnline", cmd.M{"Servers": data})
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
		"Content": "The game will be temporarily closed after {seconds} s.",
		"StartTs": w.maintain.StartTime.Unix(),
		"EndTs":   w.maintain.EndTime.Unix(),
	}
	if secs > 60 {
		msg["Content"] = "The game will be temporarily closed after {clock}."

		util.StopTimer(w.maintain.notifyTimer)
		w.maintain.notifyTimer = util.NewTimer(w.notifyMaintain, time.Until(w.maintain.StartTime)-time.Minute)
	}
	log.Infof("maintain broadcast msg %s", msg["Content"])
	service.Broadcast2Gateway("Maintain", msg)
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
			util.StopTimer(w.maintain.notifyTimer)
			w.maintain.StartTime = startTime
			w.maintain.EndTime = endTime
			log.Infof("maintain game at %s-%s content %s allow list %s", pbData.StartTime, pbData.EndTime, pbData.Content, pbData.AllowList)

			if d := time.Until(startTime); d > 5*time.Minute {
				w.maintain.notifyTimer = util.NewTimer(w.notifyMaintain, d-5*time.Minute)
			} else if d > time.Minute {
				w.maintain.notifyTimer = util.NewTimer(w.notifyMaintain, -time.Minute)
			}
		})
	}()
}
