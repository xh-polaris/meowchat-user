package consts

import (
	"google.golang.org/grpc/status"
)

var (
	ErrNotFound        = status.Error(12001, "data not found")
	ErrInvalidObjectId = status.Error(12002, "invalid objectId")
	ErrDataBase        = status.Error(10002, "database error")
	ErrNoThisItem      = status.Error(10003, "no this item")
	ErrOutOfTime       = status.Error(10004, "out of time")
)
