// 测试工具
package main

import (
	"gofishing-game/service"
)

func init() {
	service.AddTestTool(&hallTestTool{})
}

type hallTestTool struct {
}
