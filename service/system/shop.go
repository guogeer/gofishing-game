// 商城
package system

import (
	"gofishing-game/internal/gameutils"
	"gofishing-game/service"

	"github.com/guogeer/quasar/v2/cmd"
	"github.com/guogeer/quasar/v2/config"
	"github.com/guogeer/quasar/v2/log"
)

const (
	actionKeyShop         = "shop"
	shopGroupFirstPay     = "firstPay"
	shopGroupSubscription = "subscribe"
)

type shopObj struct {
	player *service.Player
}

type payArgs struct {
	OrderId    string  `json:"orderId,omitempty"`
	Uid        int     `json:"uid,omitempty"`
	Price      float64 `json:"price,omitempty"`
	GoodsId    int     `json:"goodsId,omitempty"`
	IsTest     bool    `json:"isTest,omitempty"`
	PaySDK     string  `json:"paySDK,omitempty"`
	Group      string  `json:"group,omitempty"`
	ExpireTs   int64   `json:"expireTs,omitempty"`
	IsFirstPay bool    `json:"isFirstPay,omitempty"`
}

func init() {
	service.AddAction(actionKeyShop, newShopObj)

	cmd.Bind("func_pay", funcPay, (*payArgs)(nil), cmd.WithPrivate())
}

func newShopObj(p *service.Player) service.EnterAction {
	obj := &shopObj{player: p}
	return obj
}

func (obj *shopObj) BeforeEnter() {
}

func (obj *shopObj) onPayOk(paySDK, orderId string, goodsId int) {
	var group string
	var price float64
	config.Scan("shop", goodsId, "group,price", &group, &price)

	obj.player.WriteJSON("payOk", cmd.M{
		"shopId":  goodsId,
		"orderId": orderId,
		"paySDK":  paySDK,
	})
}

func GetShopObj(player *service.Player) *shopObj {
	action := player.GetAction(actionKeyShop)
	return action.(*shopObj)
}

// 支付成功
func funcPay(ctx *cmd.Context, data any) {
	args := data.(*payArgs)

	var price float64
	var reward string
	config.Scan("shop", args.GoodsId, "price,reward", &price, &reward)
	log.Infof("player %d pay id %d rmb %v test %v", args.Uid, args.GoodsId, args.Price, args.IsTest)

	items := gameutils.ParseNumbericItems(reward)
	//log.Debugf("ply %v buy items %v", uid, reward)
	service.AddItems(args.Uid, items, "pay")

	if p := service.GetPlayer(args.Uid); p != nil {
		GetShopObj(p).onPayOk(args.PaySDK, args.OrderId, args.GoodsId)
	}
}
