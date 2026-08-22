// Package events holds the event dispatcher that excel/exporter and
// excel/importer use internally to run BeforeExport/AfterExport-style
// hooks registered via BasicExporter.On/BasicImporter.On.
//
// EventDispatcher moved here from the top-level excel package because
// nothing in this module's public API names it directly: callers
// register hooks through BasicExporter.On/BasicImporter.On, which hold
// an *EventDispatcher in an unexported field. excel.EventType,
// excel.Event, and excel.Hook stay exported from the excel package,
// since Hook's signature (and therefore On's) is part of the public
// contract.
package events

import (
	"context"
	"sync"

	"github.com/bonarizki-dat/go-excel/excel"
)

// EventDispatcher manages event listeners and dispatches events.
type EventDispatcher struct {
	listeners map[excel.EventType][]excel.Hook
	mu        sync.RWMutex
}

// NewEventDispatcher creates a new EventDispatcher.
func NewEventDispatcher() *EventDispatcher {
	return &EventDispatcher{
		listeners: make(map[excel.EventType][]excel.Hook),
	}
}

// On registers a hook for a specific event type.
func (d *EventDispatcher) On(eventType excel.EventType, hook excel.Hook) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.listeners == nil {
		d.listeners = make(map[excel.EventType][]excel.Hook)
	}
	d.listeners[eventType] = append(d.listeners[eventType], hook)
}

// Dispatch triggers an event and calls all registered hooks.
// If a hook returns an error, the dispatching stops and the error is returned.
func (d *EventDispatcher) Dispatch(ctx context.Context, event *excel.Event) error {
	d.mu.RLock()
	hooks, ok := d.listeners[event.Type]
	d.mu.RUnlock()

	if !ok || len(hooks) == 0 {
		return nil
	}

	for _, hook := range hooks {
		if err := hook(ctx, event); err != nil {
			return err
		}
	}

	return nil
}

// HasListeners checks if there are any listeners for a specific event type.
// This can be used to avoid creating event objects if no one is listening.
func (d *EventDispatcher) HasListeners(eventType excel.EventType) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	hooks, ok := d.listeners[eventType]
	return ok && len(hooks) > 0
}
