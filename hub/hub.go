// Package hub is used to allow different connections pertaining to the same user to communicate with each other.
// To do so, every connection that is interested in communication may listen on the hub by subscribing to topics,
// or it may send messages with arbitrary message content to a specific topic.
//
// The hub is used by the user and device API to broadcast updates of sensors values, sensor metadata and other information.
package hub

// Hub manages all subscriptions and communications for a single hub.
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

// Value represents a message of arbitrary data to a specific topic sent through the hub.
type Value struct {
	Topic string
	Data  interface{}
}

// Conn is used to get data out of the hub for a number of subscriptons by a single client.
type Conn struct {
	// All messeges published on the subscribed topics a passed to the Valeu channel by the hub.
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

// New creates a new hub and starts its management process.
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

// Publish publishes a new data entity to a specific topic to all subscribers to this topic on the hub.
func (h *Hub) Publish(topic string, data interface{}) {
	h.publish <- Value{topic, data}
}

// Connect creates a new connection to the hub with no subscriptions.
func (h *Hub) Connect() *Conn {
	r := &Conn{
		parent: h,
		valueQ: make(chan Value, 16),
	}

	r.Value = r.valueQ
	return r
}

// Subscribe add a subscription to a specific topic to the connection.
func (hc *Conn) Subscribe(topic string) {
	hc.parent.subscribe <- subscription{topic, hc}
}

// Unsubscribe cancels the subscription to the given topic on the connection.
func (hc *Conn) Unsubscribe(topic string) {
	hc.parent.unsubscribe <- subscription{topic, hc}
}

// Close removes the connection from the hub and cancles all subsciptions.
func (hc *Conn) Close() {
	hc.parent.detach <- hc
	close(hc.valueQ)
}
