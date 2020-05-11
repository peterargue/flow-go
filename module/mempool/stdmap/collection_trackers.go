package stdmap

import (
	"github.com/dapperlabs/flow-go/model/flow"
	"github.com/dapperlabs/flow-go/model/verification/tracker"
)

// CollectionTrackers implements the CollectionTrackers memory pool.
type CollectionTrackers struct {
	*Backend
}

// NewCollectionTrackers creates a new memory pool for CollectionTrackers.
func NewCollectionTrackers(limit uint) (*CollectionTrackers, error) {
	a := &CollectionTrackers{
		Backend: NewBackend(WithLimit(limit)),
	}
	return a, nil
}

// Add adds a CollectionTracker to the mempool.
func (c *CollectionTrackers) Add(ct *tracker.CollectionTracker) bool {
	return c.Backend.Add(ct)
}

// Has checks whether the CollectionTracker with the given collection ID is currently in
// the memory pool.
func (c *CollectionTrackers) Has(collID flow.Identifier) bool {
	return c.Backend.Has(collID)
}

// Rem removes tracker with the given collection Id.
func (c *CollectionTrackers) Rem(collID flow.Identifier) bool {
	return c.Backend.Rem(collID)
}

// ByCollectionID returns the collection tracker for the given collection ID.
func (c *CollectionTrackers) ByCollectionID(collID flow.Identifier) (*tracker.CollectionTracker, bool) {
	entity, exists := c.Backend.ByID(collID)
	if !exists {
		return nil, false
	}
	collectionTracker := entity.(*tracker.CollectionTracker)
	return collectionTracker, true
}

// All returns all collection trackers from the pool.
func (c *CollectionTrackers) All() []*tracker.CollectionTracker {
	entities := c.Backend.All()
	collectionTrackers := make([]*tracker.CollectionTracker, 0, len(entities))
	for _, entity := range entities {
		collectionTrackers = append(collectionTrackers, entity.(*tracker.CollectionTracker))
	}
	return collectionTrackers
}
