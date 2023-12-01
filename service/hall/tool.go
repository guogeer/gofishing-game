// 测试工具
package hall

import (
	"gofishing-game/service"
)

func init() {
	service.AddTestTool(&hallTestTool{})
}

type hallTestTool struct {
}
