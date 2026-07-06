package events

import "fmt"

// EventBus — простий in-memory шина подій для зв'язку між контекстами
type EventBus struct {
	handlers []EventHandler
}

// EventHandler — обробник подій
type EventHandler interface {
	Handle(event interface{})
}

func NewEventBus() *EventBus {
	return &EventBus{}
}

func (bus *EventBus) Subscribe(handler EventHandler) {
	bus.handlers = append(bus.handlers, handler)
}

func (bus *EventBus) Publish(event interface{}) {
	for _, h := range bus.handlers {
		h.Handle(event)
	}
}

// --- Bounded Context: Reporting ---
// Контекст звітності підписується на події розкладу і реагує незалежно

// ReportingEventHandler — обробник подій у контексті звітності
type ReportingEventHandler struct{}

func NewReportingEventHandler() *ReportingEventHandler {
	return &ReportingEventHandler{}
}

func (h *ReportingEventHandler) Handle(event interface{}) {
	switch e := event.(type) {
	case interface{ ScheduleID() int }:
		_ = e
		fmt.Println("[Reporting] Нове заняття додано до розкладу — оновлення звітності")
	}
}
