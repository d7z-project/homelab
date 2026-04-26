package common

import (
	"context"
	"encoding/json"
	runtimepkg "homelab/pkg/runtime"
	"log"
	"reflect"
	"strings"
	"sync"
)

const (
	EventIPPoolChanged       = "ip_pool_changed"
	EventIPSyncPolicyChanged = "ip_sync_policy_changed"
	EventIPSyncRun           = "ip_sync_run"

	EventSitePoolChanged       = "site_pool_changed"
	EventSiteSyncPolicyChanged = "site_sync_policy_changed"
	EventSiteSyncRun           = "site_sync_run"

	EventMMDBUpdate                = "mmdb_update"
	EventIntelligenceSourceChanged = "intelligence_source_changed"

	EventWorkflowExecute        = "workflow_execute"
	EventWorkflowTriggerChanged = "workflow_trigger_changed"
)

type eventDispatcher interface {
	Dispatch(ctx context.Context, payload string)
}

type genericDispatcher[T any] struct {
	handler func(ctx context.Context, payload T)
}

func (d *genericDispatcher[T]) Dispatch(ctx context.Context, payloadStr string) {
	var payload T
	// 使用反射安全地检查 T 的底层类型是否为 string
	t := reflect.TypeOf(payload)
	if t != nil && t.Kind() == reflect.String {
		if s, ok := any(&payload).(*string); ok {
			*s = payloadStr
			d.handler(ctx, payload)
			return
		}
	}

	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		// 如果解析失败且期望的是 string，直接透传 (兼容旧逻辑)
		if s, ok := any(&payload).(*string); ok {
			*s = payloadStr
		} else {
			log.Printf("[Events] failed to unmarshal payload %q into %T: %v", payloadStr, payload, err)
			return
		}
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
		var pStr string
		if s, ok := payload.(string); ok {
			pStr = s
		} else {
			data, _ := json.Marshal(payload)
			pStr = string(data)
		}
		msg = event + ":" + pStr
	}

	if err := subscriber.Publish(ctx, "homelab:cluster:events", msg); err != nil {
		log.Printf("[Events] failed to notify cluster: %v", err)
	}
}

// TriggerEvent directly triggers handlers for an event (used for testing).
func TriggerEvent(ctx context.Context, event string, payload string) {
	eventHandlersMu.RLock()
	dispatchers := eventDispatchers[event]
	eventHandlersMu.RUnlock()

	for _, d := range dispatchers {
		d.Dispatch(ctx, payload)
	}
}
