package common

import (
	"context"
	"encoding/json"
	runtimepkg "homelab/pkg/runtime"
	"log"
	"strings"
	"sync"
)

const (
	EventIPPoolChanged       = "ip_pool_changed"
	EventIPSyncPolicyChanged = "ip_sync_policy_changed"

	EventSitePoolChanged       = "site_pool_changed"
	EventSiteSyncPolicyChanged = "site_sync_policy_changed"

	EventMMDBUpdate                = "mmdb_update"
	EventIntelligenceSourceChanged = "intelligence_source_changed"

	EventWorkflowTriggerChanged = "workflow_trigger_changed"
)

type ResourceEventPayload struct {
	ID string `json:"id"`
}

type eventDispatcher interface {
	Dispatch(ctx context.Context, payload string)
}

type genericDispatcher[T any] struct {
	handler func(ctx context.Context, payload T)
}

func (d *genericDispatcher[T]) Dispatch(ctx context.Context, payloadStr string) {
	var payload T
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		log.Printf("[Events] failed to unmarshal payload %q into %T: %v", payloadStr, payload, err)
		return
	}
	d.handler(ctx, payload)
}

var (
	eventDispatchers = make(map[string][]eventDispatcher)
	eventHandlersMu  sync.RWMutex
	eventLoopStarted bool
	eventLoopMu      sync.Mutex
)

// RegisterEventHandler registers a generic handler for a specific cluster event.
func RegisterEventHandler[T any](event string, handler func(ctx context.Context, payload T)) {
	eventHandlersMu.Lock()
	defer eventHandlersMu.Unlock()
	dispatcher := &genericDispatcher[T]{handler: handler}
	eventDispatchers[event] = append(eventDispatchers[event], dispatcher)
}

// StartEventLoop connects to the cluster Pub/Sub and starts dispatching events.
func StartEventLoop(ctx context.Context) {
	eventLoopMu.Lock()
	defer eventLoopMu.Unlock()
	subscriber := runtimepkg.SubscriberFromContext(ctx)
	if eventLoopStarted || subscriber == nil {
		return
	}
	eventLoopStarted = true

	go func() {
		ch, err := subscriber.Subscribe(ctx, "homelab:cluster:events")
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
					dispatchers := eventDispatchers[event]
					eventHandlersMu.RUnlock()

					for _, d := range dispatchers {
						go d.Dispatch(ctx, payload)
					}
				}
			}
		}
	}()
}

// GetEventHandlersCount returns the number of registered handlers for a given event (used for testing).
func GetEventHandlersCount(event string) int {
	eventHandlersMu.RLock()
	defer eventHandlersMu.RUnlock()
	return len(eventDispatchers[event])
}

// ResetEventHandlers clears all registered handlers (used for testing initialization).
func ResetEventHandlers() {
	eventHandlersMu.Lock()
	defer eventHandlersMu.Unlock()
	eventDispatchers = make(map[string][]eventDispatcher)

	eventLoopMu.Lock()
	defer eventLoopMu.Unlock()
	eventLoopStarted = false
}

// NotifyCluster broadcasts an event with an optional payload to the cluster.
func NotifyCluster(ctx context.Context, event string, payload any) {
	subscriber := runtimepkg.SubscriberFromContext(ctx)
	if subscriber == nil {
		return
	}
	msg := event
	if payload != nil {
		data, _ := json.Marshal(payload)
		msg = event + ":" + string(data)
	}

	if err := subscriber.Publish(ctx, "homelab:cluster:events", msg); err != nil {
		log.Printf("[Events] failed to notify cluster: %v", err)
	}
}

// TriggerEvent directly triggers handlers for an event (used for testing).
func TriggerEvent(ctx context.Context, event string, payload any) {
	eventHandlersMu.RLock()
	dispatchers := eventDispatchers[event]
	eventHandlersMu.RUnlock()

	payloadStr := ""
	if payload != nil {
		data, _ := json.Marshal(payload)
		payloadStr = string(data)
	}
	for _, d := range dispatchers {
		d.Dispatch(ctx, payloadStr)
	}
}
