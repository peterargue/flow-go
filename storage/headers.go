// (c) 2019 Dapper Labs - ALL RIGHTS RESERVED

package storage

import (
	"github.com/onflow/flow-go/model/flow"
)

// Headers represents persistent storage for blocks.
type Headers interface {

	// Store will store a header.
	Store(header *flow.Header) error

	// ByBlockID returns the header with the given ID. It is available for finalized and ambiguous blocks.
	// Error returns:
	//  - ErrNotFound if no block header with the given ID exists
	ByBlockID(blockID flow.Identifier) (*flow.Header, error)

	// ByHeight returns the block with the given number. It is only available for finalized blocks.
	ByHeight(height uint64) (*flow.Header, error)

	// BlockIDByHeight the block ID that is finalized at the given height. It is an optimized version
	// of `ByHeight` that skips retrieving the block. Expected errors during normal operations:
	//  * `storage.ErrNotFound` if no finalized block is known at given height
	BlockIDByHeight(height uint64) (flow.Identifier, error)

	// ByParentID finds all children for the given parent block. The returned headers
	// might be unfinalized; if there is more than one, at least one of them has to
	// be unfinalized.
	ByParentID(parentID flow.Identifier) ([]*flow.Header, error)

	// IndexByChunkID indexes block ID by chunk ID.
	IndexByChunkID(headerID, chunkID flow.Identifier) error

	// BatchIndexByChunkID indexes block ID by chunk ID in a given batch.
	BatchIndexByChunkID(headerID, chunkID flow.Identifier, batch BatchStorage) error

	// IDByChunkID finds the ID of the block corresponding to given chunk ID.
	IDByChunkID(chunkID flow.Identifier) (flow.Identifier, error)
}
