package errcode

type Error interface {
	GetCode() string
	Error() string
}

var errorCodes = map[string]Error{}

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
	if _, ok := errorCodes[code]; ok {
		panic("redefined error code: " + code)
	}

	e := &BaseError{Code: code, Msg: msg}
	errorCodes[code] = e
	return e
}

func Get(key string) Error {
	return errorCodes[key]
}

var Retry = New("retry", "catch error, please retry")
