package util

import (
	"github.com/bytedance/sonic"

	"github.com/xh-polaris/meowchat-user/biz/infrastructure/util/log"
)

func JSONF(v any) string {
	data, err := sonic.Marshal(v)
	if err != nil {
		log.Error("JSONF fail, v=%v, err=%v", v, err)
	}
	return string(data)
}
