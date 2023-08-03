package es

import (
	"github.com/xh-polaris/paginator-go/esp"
	"meowchat-user/biz/infrastructure/data/db"
)

const (
	ScoreSort = db.ScoreSorter
)

var Sorters = map[int]esp.EsSorter{
	ScoreSort: (*esp.ScoreSorter)(nil),
}
