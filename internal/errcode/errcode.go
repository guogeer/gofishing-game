package errcode

import "encoding/json"

type Error interface {
	Code() string
	Error() string
}

var errorCodes = map[string]Error{}

type BaseError struct {
	code string
	msg  string
}

func (e BaseError) Code() string {
	return e.code
}

func (e BaseError) Error() string {
	return e.msg
}

type fakeBaseError struct {
	Code string `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
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

var (
	Retry = New("retry", "catch error, please retry")
)
