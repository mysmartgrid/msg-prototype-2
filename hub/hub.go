package hub

import "time"

type Hub struct {
	subscribers map[string]map[*Conn]bool

	subscribe   chan subscription
	unsubscribe chan subscription
	detach      chan *Conn

	publish chan Value
}

type subscription struct {
	topic string
	conn  *Conn
}

type Value struct {
	Topic  string
	Sensor string
	Time   time.Time
	Value  float64
}

type Conn struct {
	Value <-chan Value

	parent *Hub
	valueQ chan Value
}

func (h *Hub) doUnsubscribe(topic string, conn *Conn) {
	delete(h.subscribers[topic], conn)
	if len(h.subscribers[topic]) == 0 {
		delete(h.subscribers, topic)
	}
}

func New() *Hub {
	hub := &Hub{
		subscribers: make(map[string]map[*Conn]bool),
		subscribe:   make(chan subscription),
		unsubscribe: make(chan subscription),
		detach:      make(chan *Conn),
		publish:     make(chan Value),
	}

	go func() {
		for {
			select {
			case s := <-hub.subscribe:
				if hub.subscribers[s.topic] == nil {
					hub.subscribers[s.topic] = make(map[*Conn]bool)
				}
				hub.subscribers[s.topic][s.conn] = true

			case s := <-hub.unsubscribe:
				hub.doUnsubscribe(s.topic, s.conn)

			case hc := <-hub.detach:
				for topic := range hub.subscribers {
					hub.doUnsubscribe(topic, hc)
				}

			case value := <-hub.publish:
				for conn := range hub.subscribers[value.Topic] {
					conn.valueQ <- value
				}
			}
		}
	}()

	return hub
}

func (h *Hub) PublishValue(topic, sensor string, time time.Time, val float64) {
	h.publish <- Value{topic, sensor, time, val}
}

func (h *Hub) Connect() *Conn {
	r := &Conn{
		parent: h,
		valueQ: make(chan Value, 16),
	}

	r.Value = r.valueQ
	return r
}

func (hc *Conn) Subscribe(topic string) {
	hc.parent.subscribe <- subscription{topic, hc}
}

func (hc *Conn) Unsubscribe(topic string) {
	hc.parent.unsubscribe <- subscription{topic, hc}
}

func (hc *Conn) Close() {
	hc.parent.detach <- hc
	close(hc.valueQ)
}
