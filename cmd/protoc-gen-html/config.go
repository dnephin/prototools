package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"

	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/pkg/errors"
	"sourcegraph.com/sourcegraph/prototools/tmpl"
)

func loadConfig(request *plugin.CodeGeneratorRequest) (tmpl.Config, error) {
	config := tmpl.Config{}
	params := paramsToMap(request)

	if conf, ok := params["conf"]; ok {
		confData, err := ioutil.ReadFile(conf)
		if err != nil {
			return config, errors.Wrapf(err, "failed to read conf file %s", conf)
		}
		if err := json.Unmarshal(confData, &config); err != nil {
			return config, errors.Wrapf(err, "failed to unmarshal config %s", conf)
		}
	}

	if value, ok := params["root"]; ok {
		config.Root = value
	}

	if config.Root == "" {
		var err error
		config.Root, err = os.Getwd()
		if err != nil {
			return config, err
		}
	}

	// TODO: set other config fields from params
	return config, nil
}

// paramsToMap parses the comma-separated command-line parameters passed to the
// generator by protoc via r.GetParameters. Returned is a map of key=value
// parameters with whitespace preserved.
func paramsToMap(r *plugin.CodeGeneratorRequest) map[string]string {
	items := strings.Split(r.GetParameter(), ",")
	params := make(map[string]string, len(items))

	for _, p := range items {
		parts := strings.Split(p, "=")
		var value string
		if len(parts) > 1 {
			value = strings.TrimSpace(parts[1])
		}
		params[strings.TrimSpace(parts[0])] = value
	}
	return params
}
