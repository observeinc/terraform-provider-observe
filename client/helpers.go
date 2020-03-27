package client

import (
	"github.com/mitchellh/mapstructure"
)

func decode(input interface{}, output interface{}, strict bool) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused: strict,
		Result:      output,
	})
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}

func decodeLoose(input interface{}, output interface{}) error {
	return decode(input, output, false)
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
