package errcode

import (
	"strings"

	"github.com/guogeer/quasar/config"
)

type Error interface {
	GetCode() string
	Error() string
}

type BaseError struct {
	Code string `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
}

func (e BaseError) GetCode() string {
	return e.Code
}

func (e BaseError) Error() string {
	return e.Msg
}

func New(code, msg string) *BaseError {
	e := &BaseError{Code: code, Msg: msg}
	return e
}

type itemError struct {
	BaseError
	ItemId int `json:"itemId,omitempty"`
}

func MoreItem(itemId int) Error {
	itemName, _ := config.String("item", itemId, "name")
	e := *moreItem
	e.Msg = strings.ReplaceAll(e.Msg, "{itemName}", itemName)
	return &itemError{ItemId: itemId, BaseError: e}
}

func TooMuchItem(itemId int) Error {
	itemName, _ := config.String("item", itemId, "name")
	e := *tooMuchItem
	e.Msg = strings.ReplaceAll(e.Msg, "{itemName}", itemName)
	return &itemError{ItemId: itemId, BaseError: e}
}

var Retry = New("retry", "catch error, please retry")
var moreItem = New("more_item", "more item {itemName}")
var tooMuchItem = New("too_much_item", "too much item {itemName}")
