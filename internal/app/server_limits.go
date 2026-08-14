package app

import "time"

type ServerLimits struct {
	MaxRecvMsgSize int
	MaxSendMsgSize int

	KeepaliveTime              time.Duration
	KeepaliveTimeout           time.Duration
	KeepaliveMaxConnectionIdle time.Duration
	KeepaliveMaxConnectionAge  time.Duration
}

func DefaultServerLimits() ServerLimits {
	return ServerLimits{
		MaxRecvMsgSize:             defaultMaxRecvMsgSize,
		MaxSendMsgSize:             0,
		KeepaliveTime:              keepaliveTime,
		KeepaliveTimeout:           keepaliveTimeout,
		KeepaliveMaxConnectionIdle: keepaliveMaxIdle,
		KeepaliveMaxConnectionAge:  keepaliveMaxAge,
	}
}

func (l ServerLimits) withDefaults() ServerLimits {
	def := DefaultServerLimits()

	return ServerLimits{
		MaxRecvMsgSize:             orDefault(l.MaxRecvMsgSize, def.MaxRecvMsgSize),
		MaxSendMsgSize:             max(l.MaxSendMsgSize, 0),
		KeepaliveTime:              orDefault(l.KeepaliveTime, def.KeepaliveTime),
		KeepaliveTimeout:           orDefault(l.KeepaliveTimeout, def.KeepaliveTimeout),
		KeepaliveMaxConnectionIdle: orDefault(l.KeepaliveMaxConnectionIdle, def.KeepaliveMaxConnectionIdle),
		KeepaliveMaxConnectionAge:  orDefault(l.KeepaliveMaxConnectionAge, def.KeepaliveMaxConnectionAge),
	}
}

//nolint:ireturn // T is a value type union, not an interface.
func orDefault[T int | time.Duration](value, fallback T) T {
	if value <= 0 {
		return fallback
	}

	return value
}
