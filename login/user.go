package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	_ "gofishing-game/internal/rpc"

	"github.com/guogeer/quasar/v2/cmd"
	"github.com/guogeer/quasar/v2/log"

	"github.com/guogeer/quasar/v2/config"
)

var gRandNames []string
var table = [...]byte{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0'}

func init() {
	for _, rowId := range config.Rows("robot") {
		var nickname, icon string
		config.Scan("robot", rowId, "nickname,icon", &nickname, &icon)
		if icon == "" {
			gRandNames = append(gRandNames, nickname)
		}
	}
}

func RandStringNum(max int) string {
	nums := make([]byte, max)
	n, err := io.ReadAtLeast(rand.Reader, nums, max)
	if n != max {
		log.Error(err)
	}
	for i := 0; i < len(nums); i++ {
		nums[i] = table[int(nums[i])%len(table)]
	}
	return string(nums)
}

func GetRandName(sex int) string {
	return strings.Join([]string{"guest", RandStringNum(6)}, "_")
}

func saveFacebookIcon(iconName string, path string) (string, error) {
	var response struct{ Path string }

	err := requestPlate("/plate/save_icon", cmd.M{"iconName": iconName, "path": path}, &response)
	if err != nil {
		return "", err
	}
	return response.Path, nil
}

func requestPlate(uri string, in, out any) error {
	addr, err := cmd.RequestServerAddr("plate")
	if err != nil {
		return err
	}

	data, _ := cmd.Encode("", in)
	resp, err := http.Post("http://"+addr+uri, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	buf, _ := io.ReadAll(resp.Body)
	pkg, _ := cmd.Decode(buf)
	return json.Unmarshal(pkg.Data, &out)
}
