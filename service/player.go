// 玩家数据

package service

import (
	"context"
	"encoding/json"
	"time"

	"gofishing-game/internal/errcode"
	"gofishing-game/internal/gameutils"
	"gofishing-game/internal/pb"
	"gofishing-game/internal/rpc"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/config"
	"github.com/guogeer/quasar/log"
	"github.com/guogeer/quasar/util"
	"google.golang.org/protobuf/proto"
)

var errKickOut = errcode.New("kick_out", "kick out")
var errDelayLeave = errcode.New("delay_leave", "delay leave")

// 必须可直接复制
type UserInfo struct {
	Id       int    `json:"uid,omitempty" alias:"Uid"`
	Nickname string `json:"nickname,omitempty"` // 昵称
	Icon     string `json:"icon,omitempty"`     // 头像
	Sex      int    `json:"sex,omitempty"`      // 0:女性，1:男性
	VIP      int    `json:"vip,omitempty"`      // VIP等级
	CurExp   int    `json:"curExp,omitempty"`   // 升级后消耗后剩余的经验值

	Level  int    `json:"level,omitempty"`
	IP     string `json:"ip,omitempty"`     // IP地址
	ChanId string `json:"chanId,omitempty"` // 渠道号
	OS     string `json:"os,omitempty"`     // 系统版本

	IsRobot bool `json:"-,omitempty"` // 机器人 true
}

type GameAction interface {
	TryEnter() errcode.Error
	TryLeave() errcode.Error
	BeforeEnter() // 消息GetPlayerInfo之前
	AfterEnter()  // 消息GetPlayerInfo进入游戏后
	BeforeLeave() // 离开游戏前

	OnClose()
	OnAddItems(items []gameutils.Item, way string)
	//OnPurchaseSubscription()
}

// 想了一个小时，最后决定叫这个名字，我尽力了
type Player struct {
	UserInfo        // 基本信息
	Phone    string // 手机号
	enterReq *enterRequest

	// IsClose、IsClone容易混淆
	IsSessionClose bool // 断开连接
	isBusy         bool // 玩家进入游戏，离开游戏时

	dataObj    *dataObj
	itemObj    *ItemObj
	GameAction GameAction

	TimerGroup *util.TimerGroup // 定时器组
	closeTimer *util.Timer

	// IsFirstEnter bool // 第一次进入游戏（非房间）

	clientValues map[string]any // 登录时通知客户端的键值
	LeaveErr     errcode.Error  // 玩家异常离开房间
	leaveCtx     *cmd.Context   // 离开游戏的上下文
	CreateTime   string         // 注册时间

	tempValues   map[string]any // 临时上下文数据
	allNotify    map[string]any // 通知数据
	enterActions map[string]EnterAction
}

func NewPlayer(action GameAction) *Player {
	player := &Player{
		GameAction:   action,
		enterActions: make(map[string]EnterAction),
	}

	// 加载顺序room->base->item
	player.dataObj = newDataObj(player)
	player.itemObj = newItemObj(player)
	return player
}

func (player *Player) EnterReq() *enterRequest {
	return player.enterReq
}

func (player *Player) SetTempValue(key string, value any) {
	player.tempValues[key] = value
}

func (player *Player) GetTempValue(key string) any {
	return player.tempValues[key]
}

func (player *Player) DataObj() *dataObj {
	return player.dataObj
}

func (player *Player) ItemObj() *ItemObj {
	return player.itemObj
}

func (player *Player) Enter() errcode.Error {
	data := player.enterReq.Data
	gameutils.InitNilFields(data)

	util.DeepCopy(&player.UserInfo, data.UserInfo)
	player.CreateTime = data.UserInfo.CreateTime

	player.isBusy = true
	player.IsRobot = (player.ChanId == "robot")
	player.TimerGroup = &util.TimerGroup{}
	player.IsSessionClose = false
	player.tempValues = map[string]any{}
	player.allNotify = map[string]any{}

	if e := player.dataObj.Enter(); e != nil {
		return e
	}

	if e := player.GameAction.TryEnter(); e != nil {
		return e
	}
	return nil
}

func (player *Player) OnEnter() {
	// enter ok
	player.isBusy = true
	player.IsSessionClose = true
	player.clientValues = map[string]any{}
	player.LeaveErr = nil

	if obj := player.dataObj; obj != nil {
		obj.BeforeEnter()
	}

	player.ItemObj().BeforeEnter()
	for _, action := range player.enterActions {
		action.BeforeEnter()
	}

	player.updateLevel("enter")
	player.GameAction.BeforeEnter()
	if !player.isBusy {
		return
	}

	player.IsSessionClose = false
	player.WriteJSON("enter", nil)
	items := player.itemObj.GetItems()
	//log.Debugf("player %v OnEnter items %v", player.Id, items)
	player.SetClientValue("items", items)

	// 角色个人信息
	player.MergeClientValue("userInfo", player.UserInfo)
	// 最后更新背包
	player.WriteJSON("getPlayerInfo", player.clientValues)
	player.GameAction.AfterEnter()

	//测试工具推送
	if tools := getTestTools(player.Id); len(tools) > 0 {
		player.WriteJSON("getTestTools", map[string]any{"tools": tools})
	}
	if len(player.allNotify) > 0 {
		player.WriteJSON("notify", player.allNotify)
	}

	player.isBusy = false
	player.clientValues = nil
	util.StopTimer(player.closeTimer) // 移除断线超时T人
}

// 进入游戏时，需通知客户端的数据
func (player *Player) SetClientValue(key string, data any) {
	if _, ok := player.clientValues[key]; ok {
		log.Warnf("player %d client info %s init already", player.Id, key)
	}
	player.clientValues[key] = data
}

func (player *Player) MergeClientValue(key string, data any) {
	mergeValues := []any{data}
	if v, ok := player.clientValues[key]; ok {
		mergeValues = append(mergeValues, v)
	}

	mergeMap := map[string]json.RawMessage{}
	for _, mergeValue := range mergeValues {
		var m map[string]json.RawMessage
		buf, _ := json.Marshal(mergeValue)
		json.Unmarshal(buf, &m)
		for k, v := range m {
			mergeMap[k] = v
		}
	}
	player.clientValues[key] = mergeMap
}

func (player *Player) TryLeave() errcode.Error {
	return nil
}

func (player *Player) Leave() {
	player.Leave2(nil, nil)
}

func (player *Player) Leave2(leaveCtx *cmd.Context, cause errcode.Error) {
	log.Debugf("player %d try leave room", player.Id)
	if player.IsBusy() {
		return
	}

	e := player.GameAction.TryLeave()
	if e != nil {
		return
	}

	player.isBusy = true
	if e == errDelayLeave {
		return
	}

	player.LeaveErr = cause
	player.leaveCtx = leaveCtx
	player.leaveOk()

}

func (player *Player) CallWithoutSession(f func()) {
	ss := player.enterReq.session
	player.enterReq.session = nil
	f()
	player.enterReq.session = ss
}

func (player *Player) onLeave() {
	if player.IsBusy() {
		return
	}

	uid := player.Id
	leaveCtx := player.leaveCtx
	if leaveCtx == nil {
		player.WriteJSON("leave", cmd.M{
			"error": nil,
			"uid":   uid,
		})
	}
	// 克隆的玩家，会话为空，此时不能清理
	if ss := player.enterReq.session; ss != nil {
		delete(gGatewayPlayers, ss.Id)
	}
	player.enterReq.session = nil
	player.isBusy = false
	if player == gAllPlayers[uid] {
		delete(gAllPlayers, uid)
	}

	// 离开后
	player.dataObj.OnLeave()
	if len(playerObjectPool) < cap(playerObjectPool) {
		playerObjectPool = append(playerObjectPool, player)
	}

	// 连接关闭等，会增加定时器
	player.TimerGroup.StopAllTimer()
	player.IsSessionClose = true // 离开房间算断线

	if leaveCtx != nil {
		leaveCtx.Out.WriteJSON("FUNC_Leave", map[string]any{"uid": uid})
	}
	player.leaveCtx = nil
}

func (player *Player) IsBusy() bool {
	return player.isBusy
}

func (player *Player) leaveOk() {
	// leave ok
	uid := player.Id
	bin := &pb.UserBin{}

	log.Infof("player %d leave ok", uid)
	player.TimerGroup.StopAllTimer()

	// 离开前，不需要与客户端同步
	player.CallWithoutSession(func() {
		player.GameAction.BeforeLeave()
		player.dataObj.saveAll(bin)

		for _, action := range player.enterActions {
			if h, ok := action.(actionLeave); ok {
				h.OnLeave()
			}
		}
	})

	bin = proto.Clone(bin).(*pb.UserBin)
	go func() {
		rpc.CacheClient().SaveBin(context.Background(), &pb.SaveBinReq{Uid: int32(uid), Bin: bin})
		rpc.CacheClient().Visit(context.Background(), &pb.VisitReq{Uid: int32(uid)})
		rpc.OnResponse(func() {
			player.isBusy = false
			player.onLeave()
		})
	}()
}
func (player *Player) OnClose() {
	log.Infof("player %d lose connection", player.Id)

	for _, action := range player.enterActions {
		if h, ok := action.(actionClose); ok {
			h.OnClose()
		}
	}

	player.IsSessionClose = true
	if ss := player.enterReq.session; ss != nil {
		delete(gGatewayPlayers, ss.Id)
		player.enterReq.session = nil
	}
	// 大厅
	player.TimerGroup.ResetTimer(&player.closeTimer, func() { player.Leave2(nil, errKickOut) }, 10*time.Minute)
}

func (player *Player) WriteJSON(name string, data any) {
	if data == nil {
		data = json.RawMessage(`{"code":"ok","msg":"success"}`)
	}
	if !player.IsSessionClose {
		WriteMessage(player.enterReq.session, player.enterReq.ServerId, name, data)
	}
}

// 更新经验等级
func (player *Player) updateLevel(reason string) {
	level := player.Level
	totalNeedExp := 0

	for _, rowId := range config.Rows("level") {
		var id, needExp int
		config.Scan("level", rowId, "Level,Exp", &id, &needExp)
		totalNeedExp += needExp

		num := player.ItemObj().NumItem(gameutils.ItemIdExp)
		if num >= int64(totalNeedExp) {
			player.Level = id
			player.CurExp = int(num) - totalNeedExp
		}
	}

	if player.Level > level && player.Level > 1 {
		player.WriteJSON("levelUp", map[string]any{
			"level":  player.Level,
			"curExp": player.CurExp,
		})
		// 跨越等级的修复
		var items []gameutils.Item
		for i := level + 1; i <= player.Level; i++ {
			reward, _ := config.String("level", i, "Reward")
			items = append(items, gameutils.ParseNumbericItems(reward)...)
		}

		// player.ItemObj().AddSome(items, util.GUID(), "level_up")
		// player.shopObj.OnLevelUp()
		player.SetTempValue("LevelReward", items)
		for _, action := range player.enterActions {
			if h, ok := action.(actionLevelUp); ok {
				h.OnLevelUp(reason)
			}
		}
	}
}

func (player *Player) OnAddItems(items []gameutils.Item, way string) {
	if len(items) == 0 {
		return
	}
	for _, action := range player.enterActions {
		if h, ok := action.(actionAddItems); ok {
			h.OnAddItems(items, way)
		}
	}
	player.WriteJSON("addItems", cmd.M{
		"uid":   player.Id,
		"items": items,
		"way":   way,
	})
}

// 通知
func (player *Player) Notify(data any) {
	newNotify := map[string]any{}
	oldNotify := map[string]any{}
	buf, _ := json.Marshal(data)
	json.Unmarshal(buf, &newNotify)
	for k := range newNotify {
		oldNotify[k] = player.allNotify[k]
		player.allNotify[k] = newNotify[k]
	}
	if util.EqualJSON(newNotify, oldNotify) {
		return
	}

	player.WriteJSON("notify", data)
}
