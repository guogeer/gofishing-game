package cardcontrol

import (
	"math/rand"
	"sort"

	"github.com/guogeer/quasar/v2/utils/randutils"
)

// /////////////////////////////////////////////////////////
// 庄家作弊
func HelpDealer(data sort.Interface, percent float64) {
	var rank int
	for i := 0; i+1 < data.Len(); i++ {
		if randutils.IsPercentNice(percent) {
			rank++
		}
	}
	sort.Sort(data)
	data.Swap(rank, data.Len()-1)
	for i := 0; i+1 < data.Len(); i++ {
		end := data.Len() - 1 - i
		t := rand.Intn(end)
		data.Swap(t, end-1)
	}
}
