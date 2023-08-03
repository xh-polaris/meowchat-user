package util

import (
	"github.com/cloudwego/hertz/pkg/common/json"
	"meowchat-user/biz/infrastructure/util/log"
)

func JSONF(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		log.Error("JSONF fail, v=%v, err=%v", v, err)
	}
	return string(data)
}
