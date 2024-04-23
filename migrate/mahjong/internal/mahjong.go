package internal

import (
	"gofishing-game/internal/cardutils"
	mjutils "gofishing-game/migrate/mahjong/utils"

	"github.com/guogeer/quasar/log"
)

type Meld = mjutils.Meld
type WinOption = mjutils.WinOption

const (
	InvalidCard = -1
)

const (
	OpGenZhuang  = iota + 64 // 跟庄
	OpZhuangXian             // 庄闲
	OpJiangMa                // 罚马
	OpMaiMa                  // 买马
	OpFaMa                   // 罚马
	OpMaGenGang              // 马跟杠
)

const (
	NoneCard = mjutils.MahjongGhostCard // 赖子牌
	MaxCard  = NoneCard + 10
)

const (
	_ = iota + 10
	roomStatusExchangeTriCards
	roomStatusChooseColor
	roomStatusChoosePiao
	roomStatusBuyHorse
	roomStatusChoosePao
)

const (
	OptZiMoJiaDi          = "ziMoJiaDi"
	OptZiMoJiaFan         = "ziMoJiaFan"
	OptDianGangHuaFangPao = "dianGangHuaFangPao"
	OptDianGangHuaZiMo    = "dianGangHuaZiMo"
	OptHuanSanZhang       = "huanSanZhang"
	OptYaoJiuJiangDui     = "yaoJiuJiangDui"
	OptMenQingZhongZhang  = "menQingZhongZhang"
	OptTianDiHu           = "tianDiHu"
	OptBuyHorse           = "buyHorse"
	OptCostAfterKong      = "costAfterKong"
	OptChooseColor        = "chooseColor"
	OptBoom               = "boom"             // 放炮
	OptAbleRobKong        = "ableRobKong"      // 可抢杠胡
	OptSevenPairs         = "sevenPairs"       // 七对
	OptAbleChow           = "chow"             // 允许吃牌
	OptStraightKong2      = "straightKong2"    // 直杠2分
	OptBuyHorse2          = "buyHorse2"        // 买2马
	OptBuyHorse4          = "buyHorse4"        // 买4马
	OptBuyHorse6          = "buyHors6"         // 买6马
	OptBuyHorse8          = "buyHorse8"        // 买8马
	OptBuyHorse10         = "buyHorse10"       // 买10马
	OptBuyHorse20         = "buyHorse20"       // 买20马
	OptWuGui              = "wuGui"            // 无鬼牌
	OptBaiBanZuoGui       = "baiBanZuoGui"     // 白板做鬼
	OptFanGui1            = "fanGui1"          // 翻鬼
	OptFanGui2            = "fanGui2"          // 翻双鬼
	OptWuGuiJiaBei        = "wuGuiJiaBei"      // 无鬼加倍
	OptJieJieGao          = "jieJieGao"        // 节节高
	OptQiangGangQuanBao   = "qiangGangQuanBao" // 抢杠全包
	OptGangBaoQuanBao     = "gangBaoQuanBao"   // 杠爆全包
	OptBuDaiZiPai         = "buDaiZiPai"       // 不带字牌
	OptGenZhuang          = "genZhuang"        // 跟庄
	OptBuDaiWan           = "buDaiWan"         // 不带万
	OptMaGenGang          = "maGenGang"        // 马跟杠
	OptMaGenDiFen         = "maGenDiFen"       // 马跟底分
	OptMingGangKeQiang    = "mingGangKeQiang"  // 明杠可抢
	OptQiDuiJiaFan        = "qiDuiJiaFan"      // 七对加翻
	OptMaiMaJieJieGao     = "maiMaJieJieGao"   // 买马节节高
	OptShaGui             = "shaGui"           // 杀鬼，无鬼加倍
	OptFanGui             = "fanGui"           // 翻鬼
	OptHongZhongZuoGui    = "hongZhongZuoGui"  // 红中做鬼
	OptStraightKong1      = "straightKong1"    // 直杠1分
	OptZhuangXian         = "zhuangXian"       // 庄闲
	OptBuyHorse3          = "buyHorse3"        // 买3马
	OptSiPiSiLai          = "siPiSiLai"        // 四痞四赖
	OptBaPiSiLai          = "baPiSiLai"        // 八痞四赖
	OptYiKouXiang         = "yiKouXiang"       // 一口香
	OptPiao10             = "piao10"           // 漂10
	OptPiao20             = "piao20"           // 漂20
	OptPiao30             = "piao30"           // 漂30
	OptMultiple5          = "multiple5"        // 5的倍数
	OptMultiple10         = "multiple10"       // 10的倍数
	OptBaoPai2            = "baoPai2"          // 包两家
	OptBaoPai3            = "baoPai3"          // 包三家
	OptFangPaoJiuJinHu    = "fangPaoJiuJinHu"  // 离放炮最近的人胡牌
	OptJiangYiSe          = "jiangYiSe"        // 将一色
	OptShiSanYao          = "shiSanYao"        // 十三幺

	OptAbleKongAfterChowOrPong = "ableKongAfterChowOrPong" // 吃碰后可杠
	// 郑州麻将
	OptDaiPao                 = "daiPao" // 带跑
	OptDaiHun                 = "daiHun"
	OptGangPao                = "gangPao"
	OptQiDuiJiaBei            = "qiDuiJiaBei"
	OptGangShangHuaJiaBei     = "gangShangHuaJiaBei"
	OptSiHunJiaBei            = "siHunJiaBei"
	OptZhuangJiaJiaDi         = "zhuangJiaJiaDi"
	OptHuangZhuangBuHuangGang = "huangZhuangBuHuangGang"
	OptQiDuiLaiZiZuoDui       = "qiDuiLaiZiZuoDui" // 七对时，赖子不能做任意牌
	// 湖北麻将。晃晃
	OptZiYouXuanPiao                = "ziYouXuanPiao"     // 自由选漂
	OptPingHuLaiZi2                 = "pingHuLaiZi2"      // 平胡2个癞子仅可自摸
	OptKanZhang                     = "kanZhang"          // 坎张
	OptBaoTing                      = "baoTing"           // 报听
	OptBuDaiFeng                    = "buDaiFeng"         // 不带风
	OptShengYiQuanBuPeng            = "shengYiQuanBuPeng" // 最后一圈不碰
	OptBiHu                         = "biHu"              // 必胡
	OptJiHuBuNengChiHu              = "jiHuBuNengChiHu"   // 鸡胡不能吃胡
	OptXiaoHu                       = "xiaoHu"
	OptShiBeiBuJiFen                = "shiBeiBuJiFen"                // 10倍不计分
	OptLianZhuang                   = "lianZhuang"                   // 连庄
	OptAbleDouble                   = "ableDouble"                   // 加倍
	OptAbleLookOthersAfterReadyHand = "ableLookOthersAfterReadyHand" // 听牌后看牌
)

// 番型
const (
	PingHu = iota + 64
	DuiDuiHu
	QingYiSe
	DaiYaoJiu
	QiDui
	JinGouDiao
	QingDui
	JiangDui
	LongQiDui
	QingQiDui
	QingYaoJiu
	JiangJinGouDiao
	QingJinGouDiao
	QingLongQiDui
	TianHu
	DiHu
	ShiBaLuoHan
	QingShiBaLuoHan
	JiangQiDui
	MenQing
	DuanYaoJiu
	JiangYiSe
	YiTiaoLong
	ShiSanYao
	ShuangHaoHuaQiDui
	SanHaoHuaQiDui
	HunYiSe
	DaSanYuan
	DaSiXi
	XiaoSanYuan
	XiaoSiXi
	HunDuiHu
	ZiYiSe
	HuaYaoJiu
	QingYiSeYiTiaoLong
	PaiXingQiangGangHu
	PaiXingGangShangHua
	PaiXingGangShangPao
)

// 另计番
const (
	ZiMo = iota
	Gen
	GangShangPao
	GangShangHua
	QiangGangHu
	AllAdditionNum
)

/*type WinOption struct {
	WinCard  int
	Score    int  `json:",omitempty"`
	Points   int  `json:",omitempty"`
	kanZhang bool // 坎张 1 1 6 8
	piaoLai  bool // 单调癞子 1 2 3 250
	piaoDan  bool // 有一对，一张癞子，一个序数牌 1 1 7 250
	jiangDui bool // 将对 2 5 8
}
*/

type ReadyHandOption struct {
	DiscardCard int         `json:"discardCard"`
	WinOptions  []WinOption `json:"winOptions"`
}

func PrintCards(cards []int) {
	var a []int
	for c, n := range cards {
		for i := 0; i < n; i++ {
			a = append(a, c)
		}
	}
	log.Debug("print cards", a)

	for c, n := range cards {
		if n < 0 {
			panic(c)
		}
	}
}

func SortCards(handCards []int) []int {
	cards := make([]int, 0, 16)
	for _, c := range cardutils.GetAllCards() {
		for i := 0; i < handCards[c]; i++ {
			cards = append(cards, c)
		}
	}
	return cards
}

func GetNextCards(c, n int) []int {
	var cards []int
	for i := 0; i < n && c > 0; i++ {
		if mod := c % 10; mod == 0 {
			if cardutils.IsCardValid(c + 10) {
				c = c + 10
			} else {
				for _, c1 := range cardutils.GetAllCards() {
					if c1%10 == 0 {
						c = c1
						break
					}
				}
			}
		} else if mod > 0 {
			if mod == 9 {
				c = c - mod + 1
			} else {
				c = c + 1
			}
		}
		cards = append(cards, c)
	}
	return cards
}

func IsFlower(c int) bool {
	switch c {
	case 130, 140, 150, 160, 170, 180, 190, 200:
		return true
	}
	return false
}

func IsSameValue(c int, some ...int) bool {
	for _, v := range some {
		if v == c%10 {
			return true
		}
	}
	return false
}

// 统计坎
func CountMeldsByType(melds []mjutils.Meld, type_ int) int {
	num := 0
	for _, m := range melds {
		if m.Type == type_ {
			num++
		}
	}
	return num
}

func CountMeldsByValue(melds []mjutils.Meld, some ...int) int {
	var whiteList [MaxCard]int
	for _, v := range some {
		whiteList[v]++
	}

	num := 0
	for _, m := range melds {
		t, v := m.Type, m.Card%10
		if t == mjutils.MeldSequence { // 顺子
			if whiteList[v] > 0 || whiteList[v+1] > 0 || whiteList[v+2] > 0 {
				num++
			}
		} else {
			if whiteList[v] > 0 { // 刻子
				num++
			}
		}
	}
	return num
}

func CountAllCards(cards []int, melds []mjutils.Meld) []int {
	var counter [MaxCard]int
	for _, c := range cardutils.GetAllCards() {
		counter[c] += cards[c]
	}
	for _, m := range melds {
		t, c := m.Type, m.Card
		switch t {
		case mjutils.MeldSequence:
			counter[c]++
			counter[c+1]++
			counter[c+2]++
		case mjutils.MeldTriplet, mjutils.MeldVisibleTriplet, mjutils.MeldInvisibleTriplet:
			counter[c] += 3
		case mjutils.MeldBentKong, mjutils.MeldStraightKong, mjutils.MeldInvisibleKong:
			counter[c] += 4
		}
	}
	return counter[:]
}

func CountCardsByValue(cards []int, melds []mjutils.Meld, some ...int) int {
	var whiteList [MaxCard]int
	for _, v := range some {
		whiteList[v]++
	}

	num := 0
	counter := CountAllCards(cards, melds)
	for _, c := range cardutils.GetAllCards() {
		if counter[c] > 0 && whiteList[c%10] > 0 {
			num += counter[c]
		}
	}
	return num
}

func CountSomeCards(cards []int, melds []mjutils.Meld, some ...int) int {
	var whiteList [MaxCard]int
	for _, c := range some {
		whiteList[c]++
	}

	num := 0
	counter := CountAllCards(cards, melds)
	for _, c := range cardutils.GetAllCards() {
		if counter[c] > 0 && whiteList[c] > 0 {
			num += counter[c]
		}
	}
	return num
}

func HasColor(cards []int, color int) bool {
	for _, c := range cardutils.GetAllCards() {
		if cards[c] > 0 && c/10 == color {
			return true
		}
	}
	return false
}
