package unit_test

import "context"

type okPinger struct{}

func (okPinger) Ping(context.Context) error { return nil }
