package client

import (
	"github.com/mitchellh/mapstructure"
)

func decode(input interface{}, output interface{}) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused: true,
		Result:      output,
	})
	if err != nil {
		return err
	}
	return decoder.Decode(input)
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
