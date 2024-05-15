package shuiguoji

import (
	"gofishing-game/internal/cardutils"
	"gofishing-game/service"
	"gofishing-game/service/roomutils"
	"third/pb"
	"third/rpc"
	"time"

	"github.com/guogeer/quasar/util"
	"golang.org/x/net/context"
)

func init() {
	service.CreateWorld(&shuiguojiWorld{})

	var cards []int
	for c := 0; c < AllItemNum; c++ {
		for k := 0; k < 16; k++ {
			cards = append(cards, c)
		}
	}
	cardutils.GetCardSystem().Init(cards)

	uid := service.ServiceConfig().Int("sgj_pp_top_user")
	gold := service.ServiceConfig().Int("sgj_pp_top_gold")

	req := &pb.Request{Uid: int32(uid)}
	resp, err := rpc.CacheClient().GetUserInfo(context.Background(), req)
	if err == nil {
		user := &PrizePoolUser{
			SimpleUserInfo: service.SimpleUserInfo{},
			Prize:          gold,
		}
		util.DeepCopy(&user.SimpleUserInfo, resp)
		prizePoolRank.update(user)
	}
}

type shuiguojiWorld struct{}

func (w *shuiguojiWorld) NewRoom(subId int) *roomutils.Room {
	room := &shuiguojiRoom{}
	room.Room = roomutils.NewRoom(subId, room)
	util.NewPeriodTimer(room.Sync, "2010-01-02 00:00:00", time.Second)
	return room.Room
}

func (w *shuiguojiWorld) GetName() string {
	return "sgj"
}

func (w *shuiguojiWorld) NewPlayer() *service.Player {
	p := &shuiguojiPlayer{}

	p.Player = service.NewPlayer(p)
	return p.Player
}

func GetPlayer(id int) *shuiguojiPlayer {
	if p := service.GetPlayer(id); p != nil {
		return p.GameAction.(*shuiguojiPlayer)
	}
	return nil
}
