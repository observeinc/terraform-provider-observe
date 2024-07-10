package descriptions

import (
	"embed"
	"fmt"
	"sync"

	yaml "gopkg.in/yaml.v2"
)

// content holds our static web server content.
//
//go:embed *.yaml
var content embed.FS
var cache map[string]interface{}
var once sync.Once

func load(content embed.FS) (map[string]interface{}, error) {
	direntries, err := content.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded dir: %w", err)
	}

	result := make(map[string]interface{}, len(direntries))
	for _, l := range direntries {
		filename := l.Name()

		data, err := content.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
		}

		var v map[interface{}]interface{}
		if err := yaml.Unmarshal(data, &v); err != nil {
			return nil, fmt.Errorf("failed to unmarshal yaml %s: %w", filename, err)
		}
		result[filename] = v
	}
	return result, nil
}

func Get(filename string, fields ...string) string {
	once.Do(func() {
		var err error
		if cache, err = load(content); err != nil {
			panic(err)
		}
	})
	contents := cache[filename+".yaml"]
	for _, field := range fields {
		contents = contents.(map[interface{}]interface{})[field]
	}
	s, ok := contents.(string)
	if !ok {
		panic(fmt.Sprintf("failed to load %s description from %s\n", fields, filename))
	}
	return s
}
