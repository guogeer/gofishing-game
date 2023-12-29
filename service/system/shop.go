// 商城
package system

import (
	"strings"
	"time"

	"gofishing-game/internal/gameutils"
	"gofishing-game/service"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
)

const (
	actionKeyShop         = "shop"
	shopGroupFirstPay     = "FirstPay"
	shopGroupSubscription = "Subscribe"
)

type shopObj struct {
	player               *service.Player
	subscriptionExpireTs int64 // 订阅结束时间
	subscriptionLastTs   int64 // 订阅上次发放奖励时间
}

type payArgs struct {
	OrderId    string
	UId        int
	Price      float64
	ItemId     int
	ItemNum    int64
	IsTest     bool
	PaySDK     string
	IsStop     bool
	Group      string
	ExpireTs   int64
	IsFirstPay bool
}

func init() {
	service.AddAction(actionKeyShop, newShopObj)

	cmd.BindWithName("FUNC_Pay", funcPay, (*payArgs)(nil))
	cmd.BindWithName("FUNC_UpdatePurchaseSubcription", funcUpdatePurchaseSubcription, (*payArgs)(nil))
}

func newShopObj(p *service.Player) service.EnterAction {
	obj := &shopObj{player: p}
	return obj
}

func (obj *shopObj) BeforeEnter() {
	p := obj.player
	if obj.subscriptionLastTs > 0 {
		p.SetClientValue("SubscriptionNum", 1)
	}
	if obj.subscriptionExpireTs > time.Now().Unix() {
		p.SetClientValue("SubscriptionExpireTs", obj.subscriptionExpireTs)
	}
}

func (obj *shopObj) OnAddItems(itemLog *gameutils.ItemLog) {
	p := obj.player
	// now := time.Now()
	if itemLog.Way != "pay" {
		return
	}
	// p.shopObj.OnBuy(w.ShopId, w.IsTestPay)
	var group string
	var price, firstPay float64
	config.Scan("Shop", itemLog.ShopId, "Group,ShopsPrice", &group, &price)
	itemLog.ClientWay = strings.Join([]string{"pay", group}, "_")

	data := cmd.M{
		"ShopId":   itemLog.ShopId,
		"OrderId":  itemLog.OrderId,
		"PaySDK":   itemLog.PaySDK,
		"FirstPay": firstPay,
	}

	p.WriteJSON("PayOk", data)
}

func (obj *shopObj) getSubscriptionReward() (string, string) {
	shopRows := config.FilterRows("Shop", "Group", shopGroupSubscription)
	if len(shopRows) == 0 {
		return "", ""
	}

	var firstReward, dayReward string
	config.Scan("Shop", shopRows[0], "FirstReward,DailyReward", &firstReward, &dayReward)
	return firstReward, dayReward
}

// 首次订阅
func (obj *shopObj) IsFirstSubscription() bool {
	return obj.subscriptionLastTs == 0
}

func (obj *shopObj) checkPurchaseSubscriptionReward() bool {
	now := time.Now()
	if obj.subscriptionExpireTs <= now.Unix() {
		return false
	}
	// 今日奖励已发放
	if now.Truncate(24*time.Hour) == time.Unix(obj.subscriptionLastTs, 0).Truncate(24*time.Hour) {
		return false
	}
	firstReward, dayReward := obj.getSubscriptionReward()
	if obj.subscriptionLastTs == 0 {
		obj.player.ItemObj().AddSome(gameutils.ParseItems(firstReward), "subscription_first")
	}
	obj.subscriptionLastTs = now.Unix()
	obj.player.ItemObj().AddSome(gameutils.ParseItems(dayReward), "subscription_day")
	return true
}

func (obj *shopObj) updatePurchaseSubscription(expireTs int64) {
	obj.subscriptionExpireTs = expireTs
	// obj.checkPurchaseSubscriptionReward()

	obj.player.WriteJSON("UpdatePurchaseSubscription", cmd.M{
		"ExpireTs": expireTs,
	})
}

func GetShopObj(player *service.Player) *shopObj {
	action := player.GetAction(actionKeyShop)
	return action.(*shopObj)
}

func funcPay(ctx *cmd.Context, data any) {
	args := data.(*payArgs)

	uid := args.UId
	rmb := args.Price
	num := int(args.ItemNum)
	shopId := args.ItemId

	var price float64
	var reward string
	config.Scan("Shop", shopId, "ShopsPrice,Reward", &price, &reward)
	log.Infof("player %d pay id %d num %d rmb %v test %v", uid, shopId, num, rmb, args.IsTest)
	if false && price*float64(num) > rmb {
		return
	}
	if false && price <= 0 {
		return
	}
	items := gameutils.ParseItems(reward)
	//log.Debugf("ply %v buy items %v", uid, reward)
	service.AddItems(uid, &gameutils.ItemLog{
		Kind:       "sys",
		Way:        "pay",
		ShopId:     shopId,
		IsTestPay:  args.IsTest,
		PaySDK:     args.PaySDK,
		OrderId:    args.OrderId,
		IsFirstPay: args.IsFirstPay,
		Items:      items,
	})
}

func funcUpdatePurchaseSubcription(ctx *cmd.Context, data any) {
	args := data.(*payArgs)
	ply := service.GetPlayer(args.UId)
	if ply == nil {
		return
	}
	GetShopObj(ply).updatePurchaseSubscription(args.ExpireTs)
}
