package controller

import (
	"encoding/json"
	"errors"
	"github.com/kataras/iris/context"
)

type ParamReader struct {
	context context.Context
	params  map[string]interface{}
}

func (self *ParamReader) GetJsonParamString(key string) (value string, success bool) {
	if self.IsValidParam(key) {
		return self.params[key].(string), true
	}
	return "", false
}

func (self *ParamReader) GetJsonParamBool(key string) (value bool) {
	if self.params[key] != nil {
		return self.params[key]==true
	}
	return false
}

func (self *ParamReader) GetJsonParamToMap(key string) (value []map[string]interface{}, err error) {
	if self.IsValidParam(key) {
		var headers []map[string]interface{}
		err := json.Unmarshal([]byte(self.params[key].(string)), &headers)
		if err != nil {
			return nil, err
		}
		return headers, nil
	}
	return nil, errors.New("GetJsonParamToMap: param are not set. key -> "+key)
}

func (self *ParamReader) IsValidParam(key string) bool {
	return  self.params[key] != nil && len(self.params[key].(string))>0
}

func NewParamReader(context context.Context) (*ParamReader, error) {
	params := make(map[string]interface{})
	if err := context.ReadJSON(&params); err != nil {
		return nil, err
	}
	return &ParamReader {
		context: context,
		params: params,
	}, nil
}