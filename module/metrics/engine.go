package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type EngineCollector struct {
	sent     *prometheus.CounterVec
	received *prometheus.CounterVec
	handled  *prometheus.CounterVec
}

func NewEngineCollector(registerer *Registerer) *EngineCollector {

	ec := &EngineCollector{

		sent: registerer.RegisterNewCounterVec(prometheus.CounterOpts{
			Name:      "messages_sent_total",
			Namespace: namespaceNetwork,
			Subsystem: subsystemEngine,
			Help:      "the number of messages sent by engines",
		}, []string{EngineLabel, LabelMessage}),

		received: registerer.RegisterNewCounterVec(prometheus.CounterOpts{
			Name:      "messages_received_total",
			Namespace: namespaceNetwork,
			Subsystem: subsystemEngine,
			Help:      "the number of messages received by engines",
		}, []string{EngineLabel, LabelMessage}),

		handled: registerer.RegisterNewCounterVec(prometheus.CounterOpts{
			Name:      "messages_handled_total",
			Namespace: namespaceNetwork,
			Subsystem: subsystemEngine,
			Help:      "the number of messages handled by engines",
		}, []string{EngineLabel, LabelMessage}),
	}

	return ec
}

func (ec *EngineCollector) MessageSent(engine string, message string) {
	ec.sent.With(prometheus.Labels{EngineLabel: engine, LabelMessage: message}).Inc()
}

func (ec *EngineCollector) MessageReceived(engine string, message string) {
	ec.received.With(prometheus.Labels{EngineLabel: engine, LabelMessage: message}).Inc()
}

func (ec *EngineCollector) MessageHandled(engine string, message string) {
	ec.handled.With(prometheus.Labels{EngineLabel: engine, LabelMessage: message}).Inc()
}
