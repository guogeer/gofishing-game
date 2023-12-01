package errcode

import "encoding/json"

type Error interface {
	IsOk() bool
	Code() string
	Message() string
}

var errorCodes = map[string]Error{}

type BaseError struct {
	code string
	msg  string
}

func (e BaseError) Code() string {
	return e.code
}

func (e BaseError) Message() string {
	return e.msg
}

type fakeBaseError struct {
	Code string
	Msg  string
}

func (e *BaseError) MarshalJSON() ([]byte, error) {
	fakeErr := &fakeBaseError{Code: e.code, Msg: e.msg}
	buf, err := json.Marshal(fakeErr)
	return json.RawMessage(buf), err
}

func (e *BaseError) UnmarshalJSON(buf []byte) error {
	fakeErr := &fakeBaseError{}
	if err := json.Unmarshal(buf, fakeErr); err != nil {
		return err
	}
	e.code = fakeErr.Code
	e.msg = fakeErr.Msg
	return nil
}

func (e *BaseError) IsOk() bool {
	return e.code == Ok.Code()
}

func New(code, msg string) Error {
	if _, ok := errorCodes[code]; ok {
		panic("redefined error code: " + code)
	}

	e := &BaseError{code: code, msg: msg}
	errorCodes[code] = e
	return e
}

func Get(key string) Error {
	return errorCodes[key]
}

func IsOk(e Error) bool {
	return e.Code() == Ok.Code()
}

var (
	Ok    = New("ok", "ok")
	Retry = New("retry", "catch error, please retry")
)
