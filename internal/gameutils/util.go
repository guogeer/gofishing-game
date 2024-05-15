package gameutils

import (
	"bytes"
	"encoding/json"
	"gofishing-game/internal/errcode"
	"maps"
	"reflect"
)

func InitNilFields(obj any) {
	if obj == nil {
		return
	}

	objv := reflect.Indirect(reflect.ValueOf(obj))
	if objv.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < objv.NumField(); i++ {
		objfield := objv.Field(i)
		if objfield.CanSet() && objfield.Kind() == reflect.Map && objfield.IsNil() {
			objfield.Set(reflect.MakeMap(objfield.Type()))
		}
		if objfield.CanSet() && objfield.Kind() == reflect.Ptr && objfield.IsNil() {
			objfield.Set(reflect.New(objfield.Type().Elem()))
		}
	}
}

func marshalJSONObjects(objs ...any) ([]byte, error) {
	result := map[string]json.RawMessage{}
	for _, obj := range objs {
		buf, _ := json.Marshal(obj)

		m := map[string]json.RawMessage{}
		if bytes.HasPrefix(buf, []byte("{")) {
			json.Unmarshal(buf, &m)
			maps.Copy(m, result)
		}
	}
	return json.Marshal(result)
}

func MergeError(e errcode.Error, obj any) []byte {
	if e == nil {
		e = errcode.New(errcode.CodeOk, "")
	}
	buf, _ := marshalJSONObjects(e, obj)
	return buf
}
