package badger

import (
	"fmt"

	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/module"
	"github.com/onflow/flow-go/module/irrecoverable"
	"github.com/onflow/flow-go/module/metrics"
	"github.com/onflow/flow-go/storage"
	"github.com/onflow/flow-go/storage/badger/operation"
	"github.com/onflow/flow-go/storage/badger/transaction"
)

// DefaultEpochProtocolStateCacheSize is the default size for primary epoch protocol state entry cache.
// Minimally, we have 3 entries per epoch (one on epoch Switchover, one on receiving the Epoch Setup and one when seeing the Epoch Commit event).
// Let's be generous and assume we have 20 different epoch state entries per epoch.
var DefaultEpochProtocolStateCacheSize uint = 20

// DefaultProtocolStateIndexCacheSize is the default value for secondary byBlockIdCache.
// We want to be able to cover a broad interval of views without cache misses, so we use a bigger value.
var DefaultProtocolStateIndexCacheSize uint = 1000

// EpochProtocolStateEntries implements a persistent, fork-aware storage for the Epoch-related
// sub-states of the overall of the overall Protocol State (KV Store). It uses an embedded cache
// which is populated on first retrieval to speed up access to frequently used epoch sub-state.
type EpochProtocolStateEntries struct {
	db *badger.DB

	// cache is essentially an in-memory map from `MinEpochStateEntry.ID()` -> `RichEpochStateEntry`
	// We do _not_ populate this cache which holds the RichEpochStateEntry's on store. This is because
	//   (i) we don't have the RichEpochStateEntry on store readily available and
	//  (ii) new RichEpochStateEntry are really rare throughout an epoch, so the total cost of populating
	//       the cache becomes negligible over several views.
	// In the future, we might want to populate the cache on store, if we want to maintain frequently-changing
	// information in the protocol state, like the latest sealed block. This should be a smaller amount of work,
	// because the `MinEpochStateEntry` is generated by `StateMutator.Build()`. The `StateMutator` should already
	// have the needed Epoch Setup and Commit events, since it starts with a RichEpochStateEntry for the parent
	// state and consumes Epoch Setup and Epoch Commit events. Though, we leave this optimization for later.
	//
	// `cache` only holds the distinct state entries. On the happy path, we expect something like 3 entries per epoch.
	// On the optimal happy path we have 3 entries per epoch: one entry on epoch Switchover, one on receiving the Epoch Setup
	// and one when seeing the Epoch Commit event. Let's be generous and assume we have 20 different state entries per epoch.
	// Beyond that, we are certainly leaving the domain of normal operations that we optimize for. Therefore, a cache size of
	// roughly 100 is a reasonable balance between performance and memory consumption.
	cache *Cache[flow.Identifier, *flow.RichEpochStateEntry]

	// byBlockIdCache is essentially an in-memory map from `Block.ID()` -> `MinEpochStateEntry.ID()`. The full
	// flow.RichEpochStateEntry can be retrieved from the `cache` above.
	// We populate the `byBlockIdCache` on store, because a new entry is added for every block and we probably also
	// query the Protocol state for every block. So argument (ii) from above does not apply here. Furthermore,
	// argument (i) from above also does not apply, because we already have the state entry's ID on store,
	// so populating the cache is easy.
	//
	// `byBlockIdCache` will contain an entry for every block. We want to be able to cover a broad interval of views
	// without cache misses, so a cache size of roughly 1000 entries is reasonable.
	byBlockIdCache *Cache[flow.Identifier, flow.Identifier]
}

var _ storage.EpochProtocolStateEntries = (*EpochProtocolStateEntries)(nil)

// NewEpochProtocolStateEntries creates a EpochProtocolStateEntries instance, which stores a subset of the
// state stored by the Dynamic Protocol State.
// It supports storing, caching and retrieving by ID or the additionally indexed block ID.
func NewEpochProtocolStateEntries(collector module.CacheMetrics,
	epochSetups storage.EpochSetups,
	epochCommits storage.EpochCommits,
	db *badger.DB,
	stateCacheSize uint,
	stateByBlockIDCacheSize uint,
) *EpochProtocolStateEntries {
	retrieveByEntryID := func(epochProtocolStateEntryID flow.Identifier) func(tx *badger.Txn) (*flow.RichEpochStateEntry, error) {
		var entry flow.MinEpochStateEntry
		return func(tx *badger.Txn) (*flow.RichEpochStateEntry, error) {
			err := operation.RetrieveEpochProtocolState(epochProtocolStateEntryID, &entry)(tx)
			if err != nil {
				return nil, err
			}
			result, err := newRichEpochProtocolStateEntry(&entry, epochSetups, epochCommits)
			if err != nil {
				return nil, fmt.Errorf("could not create RichEpochStateEntry: %w", err)
			}
			return result, nil
		}
	}

	storeByBlockID := func(blockID flow.Identifier, epochProtocolStateEntryID flow.Identifier) func(*transaction.Tx) error {
		return func(tx *transaction.Tx) error {
			err := transaction.WithTx(operation.IndexEpochProtocolState(blockID, epochProtocolStateEntryID))(tx)
			if err != nil {
				return fmt.Errorf("could not index EpochProtocolState for block (%x): %w", blockID[:], err)
			}
			return nil
		}
	}

	retrieveByBlockID := func(blockID flow.Identifier) func(tx *badger.Txn) (flow.Identifier, error) {
		return func(tx *badger.Txn) (flow.Identifier, error) {
			var entryID flow.Identifier
			err := operation.LookupEpochProtocolState(blockID, &entryID)(tx)
			if err != nil {
				return flow.ZeroID, fmt.Errorf("could not lookup epoch protocol state entry ID for block (%x): %w", blockID[:], err)
			}
			return entryID, nil
		}
	}

	return &EpochProtocolStateEntries{
		db: db,
		cache: newCache[flow.Identifier, *flow.RichEpochStateEntry](collector, metrics.ResourceProtocolState,
			withLimit[flow.Identifier, *flow.RichEpochStateEntry](stateCacheSize),
			withStore(noopStore[flow.Identifier, *flow.RichEpochStateEntry]),
			withRetrieve(retrieveByEntryID)),
		byBlockIdCache: newCache[flow.Identifier, flow.Identifier](collector, metrics.ResourceProtocolStateByBlockID,
			withLimit[flow.Identifier, flow.Identifier](stateByBlockIDCacheSize),
			withStore(storeByBlockID),
			withRetrieve(retrieveByBlockID)),
	}
}

// StoreTx returns an anonymous function (intended to be executed as part of a badger transaction),
// which persists the given epoch protocol state entry as part of a DB tx. Per convention, the identities in
// the flow.MinEpochStateEntry must be in canonical order for the current and next epoch (if present),
// otherwise an exception is returned.
// Expected errors of the returned anonymous function:
//   - storage.ErrAlreadyExists if a state entry with the given id is already stored
func (s *EpochProtocolStateEntries) StoreTx(epochProtocolStateEntryID flow.Identifier, epochStateEntry *flow.MinEpochStateEntry) func(*transaction.Tx) error {
	// front-load sanity checks:
	if !epochStateEntry.CurrentEpoch.ActiveIdentities.Sorted(flow.IdentifierCanonical) {
		return transaction.Fail(fmt.Errorf("sanity check failed: identities are not sorted"))
	}
	if epochStateEntry.NextEpoch != nil && !epochStateEntry.NextEpoch.ActiveIdentities.Sorted(flow.IdentifierCanonical) {
		return transaction.Fail(fmt.Errorf("sanity check failed: next epoch identities are not sorted"))
	}

	// happy path: return anonymous function, whose future execution (as part of a transaction) will store the state entry.
	return transaction.WithTx(operation.InsertEpochProtocolState(epochProtocolStateEntryID, epochStateEntry))
}

// Index returns an anonymous function that is intended to be executed as part of a database transaction.
// In a nutshell, we want to maintain a map from `blockID` to `epochStateEntry`, where `blockID` references the
// block that _proposes_ the referenced epoch protocol state entry.
// Upon call, the anonymous function persists the specific map entry in the node's database.
// Protocol convention:
//   - Consider block B, whose ingestion might potentially lead to an updated protocol state. For example,
//     the protocol state changes if we seal some execution results emitting service events.
//   - For the key `blockID`, we use the identity of block B which _proposes_ this Protocol State. As value,
//     the hash of the resulting protocol state at the end of processing B is to be used.
//   - CAUTION: The protocol state requires confirmation by a QC and will only become active at the child block,
//     _after_ validating the QC.
//
// Expected errors during normal operations:
//   - storage.ErrAlreadyExists if a state entry for the given blockID has already been indexed
func (s *EpochProtocolStateEntries) Index(blockID flow.Identifier, epochProtocolStateEntryID flow.Identifier) func(*transaction.Tx) error {
	return s.byBlockIdCache.PutTx(blockID, epochProtocolStateEntryID)
}

// ByID returns the epoch protocol state entry by its ID.
// Expected errors during normal operations:
//   - storage.ErrNotFound if no protocol state with the given Identifier is known.
func (s *EpochProtocolStateEntries) ByID(epochProtocolStateEntryID flow.Identifier) (*flow.RichEpochStateEntry, error) {
	tx := s.db.NewTransaction(false)
	defer tx.Discard()
	return s.cache.Get(epochProtocolStateEntryID)(tx)
}

// ByBlockID retrieves the epoch protocol state entry that the block with the given ID proposes.
// CAUTION: this protocol state requires confirmation by a QC and will only become active at the child block,
// _after_ validating the QC. Protocol convention:
//   - Consider block B, whose ingestion might potentially lead to an updated protocol state. For example,
//     the protocol state changes if we seal some execution results emitting service events.
//   - For the key `blockID`, we use the identity of block B which _proposes_ this Protocol State. As value,
//     the hash of the resulting protocol state at the end of processing B is to be used.
//   - CAUTION: The protocol state requires confirmation by a QC and will only become active at the child block,
//     _after_ validating the QC.
//
// Expected errors during normal operations:
//   - storage.ErrNotFound if no state entry has been indexed for the given block.
func (s *EpochProtocolStateEntries) ByBlockID(blockID flow.Identifier) (*flow.RichEpochStateEntry, error) {
	tx := s.db.NewTransaction(false)
	defer tx.Discard()
	epochProtocolStateEntryID, err := s.byBlockIdCache.Get(blockID)(tx)
	if err != nil {
		return nil, fmt.Errorf("could not lookup epoch protocol state ID for block (%x): %w", blockID[:], err)
	}
	return s.cache.Get(epochProtocolStateEntryID)(tx)
}

// newRichEpochProtocolStateEntry constructs a RichEpochStateEntry from an epoch sub-state entry.
// It queries and fills in epoch setups and commits for previous and current epochs and possibly next epoch.
// No errors are expected during normal operation.
func newRichEpochProtocolStateEntry(
	minEpochStateEntry *flow.MinEpochStateEntry,
	setups storage.EpochSetups,
	commits storage.EpochCommits,
) (*flow.RichEpochStateEntry, error) {
	var (
		previousEpochSetup  *flow.EpochSetup
		previousEpochCommit *flow.EpochCommit
		nextEpochSetup      *flow.EpochSetup
		nextEpochCommit     *flow.EpochCommit
		err                 error
	)
	// query and fill in epoch setups and commits for previous and current epochs
	if minEpochStateEntry.PreviousEpoch != nil {
		previousEpochSetup, err = setups.ByID(minEpochStateEntry.PreviousEpoch.SetupID)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve previous epoch setup: %w", err)
		}
		previousEpochCommit, err = commits.ByID(minEpochStateEntry.PreviousEpoch.CommitID)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve previous epoch commit: %w", err)
		}
	}

	currentEpochSetup, err := setups.ByID(minEpochStateEntry.CurrentEpoch.SetupID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve current epoch setup: %w", err)
	}
	currentEpochCommit, err := commits.ByID(minEpochStateEntry.CurrentEpoch.CommitID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve current epoch commit: %w", err)
	}

	// if next epoch has been set up, fill in data for it as well
	nextEpoch := minEpochStateEntry.NextEpoch
	if nextEpoch != nil {
		nextEpochSetup, err = setups.ByID(nextEpoch.SetupID)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve next epoch's setup event: %w", err)
		}
		if nextEpoch.CommitID != flow.ZeroID {
			nextEpochCommit, err = commits.ByID(nextEpoch.CommitID)
			if err != nil {
				return nil, fmt.Errorf("could not retrieve next epoch's commit event: %w", err)
			}
		}
	}

	epochStateEntry, err := flow.NewEpochStateEntry(minEpochStateEntry,
		previousEpochSetup,
		previousEpochCommit,
		currentEpochSetup,
		currentEpochCommit,
		nextEpochSetup,
		nextEpochCommit)
	if err != nil {
		// observing an error here would be an indication of severe data corruption or bug in our code since
		// all data should be available and correctly structured at this point.
		return nil, irrecoverable.NewExceptionf("critical failure while instantiating EpochStateEntry: %w", err)
	}

	result, err := flow.NewRichEpochStateEntry(epochStateEntry)
	if err != nil {
		// observing an error here would be an indication of severe data corruption or bug in our code since
		// all data should be available and correctly structured at this point.
		return nil, irrecoverable.NewExceptionf("critical failure while constructing RichEpochStateEntry from EpochStateEntry: %w", err)
	}
	return result, nil
}
