package main

import (
	"gofishing-game/service"
	"gofishing-game/service/roomutils"

	_ "gofishing-game/games/demo/fingerguessing"
	_ "gofishing-game/games/migrate/dice"
	_ "gofishing-game/games/migrate/doudizhu"
	_ "gofishing-game/games/migrate/lottery"
	_ "gofishing-game/games/migrate/mahjong"
	_ "gofishing-game/games/migrate/niuniu"
	_ "gofishing-game/games/migrate/paodekuai"
	_ "gofishing-game/games/migrate/sangong"
	_ "gofishing-game/games/migrate/texas"
	_ "gofishing-game/games/migrate/xiaojiu"
	_ "gofishing-game/games/migrate/zhajinhua"
)

func main() {
	roomutils.LoadGames()
	service.Start()
}
