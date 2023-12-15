package service

import (
	"strings"

	"gofishing-game/internal/gameutils"
)

type ActionConstructor func(player *Player) EnterAction

var (
	actionKeys         []string // AddAction调用顺序
	actionConstructors = map[string]ActionConstructor{}
)

// BeforeEnter在GameAction前调用
func AddAction(key string, h ActionConstructor) {
	key = strings.ToLower(key)
	actionKeys = append(actionKeys, key)
	actionConstructors[key] = h
}

type EnterAction interface {
	BeforeEnter()
}

func (player *Player) GetAction(key string) EnterAction {
	key = strings.ToLower(key)
	return player.enterActions[key]
}

type actionLevelUp interface {
	OnLevelUp(reason string)
}

type actionAddItems interface {
	OnAddItems(*gameutils.ItemLog)
}

type actionClose interface {
	OnClose()
}

type actionLeave interface {
	OnLeave()
}
