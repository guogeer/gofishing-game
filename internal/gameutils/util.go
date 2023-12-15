package gameutils

import (
	"fmt"
	"reflect"

	"github.com/guogeer/quasar/util"
)

var _ = fmt.Println

func addStructFieldsWithMulti(obj, add any, multi int) {
	if obj == nil || add == nil {
		return
	}
	objv := reflect.Indirect(reflect.ValueOf(obj))
	addv := reflect.Indirect(reflect.ValueOf(add))
	if objv.Kind() != reflect.Struct {
		return
	}
	if addv.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < objv.NumField(); i++ {
		objfield := objv.Field(i)
		objname := objv.Type().Field(i).Name
		addfield := addv.FieldByName(objname)

		objkind := util.ConvertKind(objfield.Kind())
		addkind := util.ConvertKind(addfield.Kind())
		// fmt.Println(objkind, addkind)
		if !objfield.CanSet() {
			continue
		}
		if objkind != addkind {
			continue
		}
		switch objkind {
		case reflect.Int64:
			objfield.SetInt(objfield.Int() + addfield.Int()*int64(multi))
		case reflect.Uint64:
			objfield.SetUint(objfield.Uint() + addfield.Uint()*uint64(multi))
		case reflect.Float64:
			objfield.SetFloat(objfield.Float() + addfield.Float()*float64(multi))
		}
	}
}

func AddStructFields(obj, add any) {
	addStructFieldsWithMulti(obj, add, 1)
}

func SubStructFields(obj, add any) {
	addStructFieldsWithMulti(obj, add, -1)
}

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

func ConvertMap(obj any) map[string]any {
	values := map[string]any{}
	objv := reflect.Indirect(reflect.ValueOf(obj))
	switch objv.Kind() {
	default:
		panic("current only support struct map")
	case reflect.Map:
		for _, key := range objv.MapKeys() {
			skey := fmt.Sprintf("%s", key.Interface())
			values[skey] = objv.MapIndex(key).Interface()
		}
	case reflect.Struct:
		for i := 0; i < objv.NumField(); i++ {
			if objfield := objv.Field(i); objfield.CanInterface() {
				objname := objv.Type().Field(i).Name
				values[objname] = objfield.Interface()
			}
		}
	}
	return values
}
