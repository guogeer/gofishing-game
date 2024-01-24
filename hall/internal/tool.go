// 测试工具
package internal

import (
	"gofishing-game/service"
)

func init() {
	service.AddTestTool(&hallTestTool{})
}

type hallTestTool struct {
}
