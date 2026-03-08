package common

import (
	"context"
	"log"
	"strings"
	"sync"
)

const (
	EventIPPoolChanged       = "ip_pool_changed"
	EventIPSyncPolicyChanged = "ip_sync_policy_changed"
	EventIPSyncRun           = "ip_sync_run"

	EventSitePoolChanged = "site_pool_changed"

	EventMMDBUpdate                = "mmdb_update"
	EventIntelligenceSourceChanged = "intelligence_source_changed"

	EventWorkflowExecute        = "workflow_execute"
	EventWorkflowTriggerChanged = "workflow_trigger_changed"
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

// GetEventHandlers returns registered handlers for a given event (used for testing).
func GetEventHandlers(event string) []EventHandler {
	eventHandlersMu.RLock()
	defer eventHandlersMu.RUnlock()
	return eventHandlers[event]
}

// ResetEventHandlers clears all registered handlers (used for testing initialization).
func ResetEventHandlers() {
	eventHandlersMu.Lock()
	defer eventHandlersMu.Unlock()
	eventHandlers = make(map[string][]EventHandler)

	eventLoopMu.Lock()
	defer eventLoopMu.Unlock()
	eventLoopStarted = false
}
