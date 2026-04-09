package filesystem

import "github.com/axiom-ide/axiom/api"

// EventPublisher est l'interface minimale dont le filesystem.Handler a besoin.
// Elle abstrait le bus.EventBus pour permettre l'injection et les tests.
type EventPublisher interface {
	Subscribe(topic api.Topic, handler func(api.Event)) string
	Publish(event api.Event)
}