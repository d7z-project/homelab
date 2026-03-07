package common

import (
	"context"
	"log"
	"strings"
	"sync"
)

type EventHandler func(ctx context.Context, payload string)

var (
	eventHandlers    = make(map[string][]EventHandler)
	eventHandlersMu  sync.RWMutex
	eventLoopStarted bool
	eventLoopMu      sync.Mutex
)

// RegisterEventHandler registers a handler for a specific cluster event.
func RegisterEventHandler(event string, handler EventHandler) {
	eventHandlersMu.Lock()
	defer eventHandlersMu.Unlock()
	eventHandlers[event] = append(eventHandlers[event], handler)
}

// StartEventLoop connects to the cluster Pub/Sub and starts dispatching events.
func StartEventLoop(ctx context.Context) {
	eventLoopMu.Lock()
	defer eventLoopMu.Unlock()
	if eventLoopStarted || Subscriber == nil {
		return
	}
	eventLoopStarted = true

	go func() {
		ch, err := Subscriber.Subscribe(ctx, "homelab:cluster:events")
		if err != nil {
			log.Printf("Failed to subscribe to cluster events: %v", err)
			eventLoopMu.Lock()
			eventLoopStarted = false
			eventLoopMu.Unlock()
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				parts := strings.SplitN(msg, ":", 2)
				if len(parts) == 2 {
					event := parts[0]
					payload := parts[1]

					eventHandlersMu.RLock()
					handlers := eventHandlers[event]
					eventHandlersMu.RUnlock()

					for _, h := range handlers {
						go h(ctx, payload)
					}
				}
			}
		}
	}()
}
