package api

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/mitchellh/mapstructure"
)

func stringToObjectIdScalarHookFunc(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if f.Kind() != reflect.String {
		return data, nil
	}
	if t != reflect.TypeOf(ObjectIdScalar(0)) {
		return data, nil
	}
	dataVal := reflect.ValueOf(data)
	return strconv.ParseInt(dataVal.String(), 10, 64)
}

func stringToTimeScalarHookFunc(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if f.Kind() != reflect.String {
		return data, nil
	}
	if t != reflect.TypeOf(time.Duration(5)) {
		return data, nil
	}
	// Convert it by parsing
	return time.ParseDuration(data.(string) + "ns")
}

func stringToInt64(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if f.Kind() != reflect.String {
		return data, nil
	}
	if t != reflect.TypeOf(int64(0)) {
		return data, nil
	}
	v, err := strconv.ParseInt(data.(string), 10, 64)
	return v, err
}

func decode(input interface{}, output interface{}, strict bool) error {
	if input == nil {
		return fmt.Errorf("not found")
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused: strict,
		Result:      output,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			stringToObjectIdScalarHookFunc,
			stringToTimeScalarHookFunc,
			stringToInt64,
		),
	})
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}

func decodeStrict(input interface{}, output interface{}) error {
	return decode(input, output, true)
}

func getNested(i interface{}, keys ...string) interface{} {
	for _, k := range keys {
		v, ok := i.(map[string]interface{})
		if !ok {
			return nil
		}
		i = v[k]
	}
	return i
}
