package common

import (
	"gopkg.d7z.net/middleware/kv"
	"gopkg.d7z.net/middleware/lock"
	"gopkg.d7z.net/middleware/subscribe"
)

var infrastructure struct {
	db         kv.KV
	locker     lock.Locker
	subscriber subscribe.Subscriber
}

func ConfigureInfrastructure(db kv.KV, locker lock.Locker, subscriber subscribe.Subscriber) {
	infrastructure.db = db
	infrastructure.locker = locker
	infrastructure.subscriber = subscriber
}
