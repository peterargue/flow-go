package badger

import (
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/consensus/hotstuff/model"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/model/flow/filter"
	"github.com/onflow/flow-go/module/irrecoverable"
	"github.com/onflow/flow-go/state/fork"
	"github.com/onflow/flow-go/state/protocol"
	"github.com/onflow/flow-go/state/protocol/inmem"
	"github.com/onflow/flow-go/state/protocol/protocol_state/kvstore"
	"github.com/onflow/flow-go/storage"
	"github.com/onflow/flow-go/storage/badger/operation"
	"github.com/onflow/flow-go/storage/badger/procedure"
)

// Snapshot implements the protocol.Snapshot interface.
// It represents a read-only immutable snapshot of the protocol state at the
// block it is constructed with. It allows efficient access to data associated directly
// with blocks at a given state (finalized, sealed), such as the related header, commit,
// seed or descending blocks. A block snapshot can lazily convert to an epoch snapshot in
// order to make data associated directly with epochs accessible through its API.
type Snapshot struct {
	state   *State
	blockID flow.Identifier // reference block for this snapshot
}

// FinalizedSnapshot represents a read-only immutable snapshot of the protocol state
// at a finalized block. It is guaranteed to have a header available.
type FinalizedSnapshot struct {
	Snapshot
	header *flow.Header
}

var _ protocol.Snapshot = (*Snapshot)(nil)
var _ protocol.Snapshot = (*FinalizedSnapshot)(nil)

// newSnapshotWithIncorporatedReferenceBlock creates a new state snapshot with the given reference block.
// CAUTION: The caller is responsible for ensuring that the reference block has been incorporated.
func newSnapshotWithIncorporatedReferenceBlock(state *State, blockID flow.Identifier) *Snapshot {
	return &Snapshot{
		state:   state,
		blockID: blockID,
	}
}

// NewFinalizedSnapshot instantiates a `FinalizedSnapshot`.
// CAUTION: the header's ID _must_ match `blockID` (not checked)
func NewFinalizedSnapshot(state *State, blockID flow.Identifier, header *flow.Header) *FinalizedSnapshot {
	return &FinalizedSnapshot{
		Snapshot: Snapshot{
			state:   state,
			blockID: blockID,
		},
		header: header,
	}
}

func (s *FinalizedSnapshot) Head() (*flow.Header, error) {
	return s.header, nil
}

func (s *Snapshot) Head() (*flow.Header, error) {
	head, err := s.state.headers.ByBlockID(s.blockID)
	return head, err
}

// QuorumCertificate (QC) returns a valid quorum certificate pointing to the
// header at this snapshot.
// The sentinel error storage.ErrNotFound is returned if the QC is unknown.
func (s *Snapshot) QuorumCertificate() (*flow.QuorumCertificate, error) {
	qc, err := s.state.qcs.ByBlockID(s.blockID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve quorum certificate for (%x): %w", s.blockID, err)
	}
	return qc, nil
}

func (s *Snapshot) EpochPhase() (flow.EpochPhase, error) {
	epochState, err := s.state.protocolState.EpochStateAtBlockID(s.blockID)
	if err != nil {
		return flow.EpochPhaseUndefined, fmt.Errorf("could not retrieve protocol state snapshot: %w", err)
	}
	return epochState.EpochPhase(), nil
}

func (s *Snapshot) Identities(selector flow.IdentityFilter[flow.Identity]) (flow.IdentityList, error) {
	epochState, err := s.state.protocolState.EpochStateAtBlockID(s.blockID)
	if err != nil {
		return nil, err
	}

	// apply the filter to the participants
	identities := epochState.Identities().Filter(selector)
	return identities, nil
}

func (s *Snapshot) Identity(nodeID flow.Identifier) (*flow.Identity, error) {
	// filter identities at snapshot for node ID
	identities, err := s.Identities(filter.HasNodeID[flow.Identity](nodeID))
	if err != nil {
		return nil, fmt.Errorf("could not get identities: %w", err)
	}

	// check if node ID is part of identities
	if len(identities) == 0 {
		return nil, protocol.IdentityNotFoundError{NodeID: nodeID}
	}
	return identities[0], nil
}

// Commit retrieves the latest execution state commitment at the current block snapshot. This
// commitment represents the execution state as currently finalized.
func (s *Snapshot) Commit() (flow.StateCommitment, error) {
	// get the ID of the sealed block
	seal, err := s.state.seals.HighestInFork(s.blockID)
	if err != nil {
		return flow.DummyStateCommitment, fmt.Errorf("could not retrieve sealed state commit: %w", err)
	}
	return seal.FinalState, nil
}

func (s *Snapshot) SealedResult() (*flow.ExecutionResult, *flow.Seal, error) {
	seal, err := s.state.seals.HighestInFork(s.blockID)
	if err != nil {
		return nil, nil, fmt.Errorf("could not look up latest seal: %w", err)
	}
	result, err := s.state.results.ByID(seal.ResultID)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get latest result: %w", err)
	}
	return result, seal, nil
}

// SealingSegment will walk through the chain backward until we reach the block referenced
// by the latest seal and build a SealingSegment. As we visit each block we check each execution
// receipt in the block's payload to make sure we have a corresponding execution result, any
// execution results missing from blocks are stored in the `SealingSegment.ExecutionResults` field.
// See `model/flow/sealing_segment.md` for detailed technical specification of the Sealing Segment
//
// Expected errors during normal operations:
//   - protocol.ErrSealingSegmentBelowRootBlock if sealing segment would stretch beyond the node's local history cut-off
//   - protocol.UnfinalizedSealingSegmentError if sealing segment would contain unfinalized blocks (including orphaned blocks)
func (s *Snapshot) SealingSegment() (*flow.SealingSegment, error) {
	// Lets denote the highest block in the sealing segment `head` (initialized below).
	// Based on the tech spec `flow/sealing_segment.md`, the Sealing Segment must contain
	//  enough history to satisfy _all_ of the following conditions:
	//   (i) The highest sealed block as of `head` needs to be included in the sealing segment.
	//       This is relevant if `head` does not contain any seals.
	//  (ii) All blocks that are sealed by `head`. This is relevant if head` contains _multiple_ seals.
	// (iii) The sealing segment should contain the history back to (including):
	//       limitHeight := max(blockSealedAtHead.Height - flow.DefaultTransactionExpiry, SporkRootBlockHeight)
	// Per convention, we include the blocks for (i) in the `SealingSegment.Blocks`, while the
	// additional blocks for (ii) and optionally (iii) are contained in as `SealingSegment.ExtraBlocks`.
	head, err := s.state.blocks.ByID(s.blockID)
	if err != nil {
		return nil, fmt.Errorf("could not get snapshot's reference block: %w", err)
	}
	if head.Header.Height < s.state.finalizedRootHeight {
		return nil, protocol.ErrSealingSegmentBelowRootBlock
	}

	// Verify that head of sealing segment is finalized.
	finalizedBlockAtHeight, err := s.state.headers.BlockIDByHeight(head.Header.Height)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, protocol.NewUnfinalizedSealingSegmentErrorf("head of sealing segment at height %d is not finalized: %w", head.Header.Height, err)
		}
		return nil, fmt.Errorf("exception while retrieving finzalized bloc, by height: %w", err)
	}
	if finalizedBlockAtHeight != s.blockID { // comparison of fixed-length arrays
		return nil, protocol.NewUnfinalizedSealingSegmentErrorf("head of sealing segment is orphaned, finalized block at height %d is %x", head.Header.Height, finalizedBlockAtHeight)
	}

	// STEP (i): highest sealed block as of `head` must be included.
	seal, err := s.state.seals.HighestInFork(s.blockID)
	if err != nil {
		return nil, fmt.Errorf("could not get seal for sealing segment: %w", err)
	}
	blockSealedAtHead, err := s.state.headers.ByBlockID(seal.BlockID)
	if err != nil {
		return nil, fmt.Errorf("could not get block: %w", err)
	}

	// TODO this is a temporary measure resulting from epoch data being stored outside the
	//  protocol KV Store, once epoch data is in the KV Store, we can pass protocolKVStoreSnapshotsDB.ByID
	//  directly to NewSealingSegmentBuilder (similar to other getters)
	getProtocolStateEntry := func(protocolStateID flow.Identifier) (*flow.ProtocolStateEntryWrapper, error) {
		kvStoreEntry, err := s.state.protocolKVStoreSnapshotsDB.ByID(protocolStateID)
		if err != nil {
			return nil, fmt.Errorf("could not get kv store entry: %w", err)
		}
		kvStoreReader, err := kvstore.VersionedDecode(kvStoreEntry.Version, kvStoreEntry.Data)
		if err != nil {
			return nil, fmt.Errorf("could not decode kv store entry: %w", err)
		}
		epochDataEntry, err := s.state.epochProtocolStateEntriesDB.ByID(kvStoreReader.GetEpochStateID())
		if err != nil {
			return nil, fmt.Errorf("could not get epoch data: %w", err)
		}
		return &flow.ProtocolStateEntryWrapper{
			KVStore: flow.PSKeyValueStoreData{
				Version: kvStoreEntry.Version,
				Data:    kvStoreEntry.Data,
			},
			EpochEntry: epochDataEntry,
		}, nil
	}

	// walk through the chain backward until we reach the block referenced by
	// the latest seal - the returned segment includes this block
	builder := flow.NewSealingSegmentBuilder(s.state.results.ByID, s.state.seals.HighestInFork, getProtocolStateEntry)
	scraper := func(header *flow.Header) error {
		blockID := header.ID()
		block, err := s.state.blocks.ByID(blockID)
		if err != nil {
			return fmt.Errorf("could not get block: %w", err)
		}

		err = builder.AddBlock(block)
		if err != nil {
			return fmt.Errorf("could not add block to sealing segment: %w", err)
		}

		return nil
	}
	err = fork.TraverseForward(s.state.headers, s.blockID, scraper, fork.IncludingBlock(seal.BlockID))
	if err != nil {
		return nil, fmt.Errorf("could not traverse sealing segment: %w", err)
	}

	// STEP (ii): extend history down to the lowest block, whose seal is included in `head`
	lowestSealedByHead := blockSealedAtHead
	for _, sealInHead := range head.Payload.Seals {
		h, e := s.state.headers.ByBlockID(sealInHead.BlockID)
		if e != nil {
			return nil, fmt.Errorf("could not get block (id=%x) for seal: %w", seal.BlockID, e) // storage.ErrNotFound or exception
		}
		if h.Height < lowestSealedByHead.Height {
			lowestSealedByHead = h
		}
	}

	// STEP (iii): extended history to allow checking for duplicated collections, i.e.
	// limitHeight = max(blockSealedAtHead.Height - flow.DefaultTransactionExpiry, SporkRootBlockHeight)
	limitHeight := s.state.sporkRootBlockHeight
	if blockSealedAtHead.Height > s.state.sporkRootBlockHeight+flow.DefaultTransactionExpiry {
		limitHeight = blockSealedAtHead.Height - flow.DefaultTransactionExpiry
	}

	// As we have to satisfy (ii) _and_ (iii), we have to take the longest history, i.e. the lowest height.
	if lowestSealedByHead.Height < limitHeight {
		limitHeight = lowestSealedByHead.Height
		if limitHeight < s.state.sporkRootBlockHeight { // sanity check; should never happen
			return nil, fmt.Errorf("unexpected internal error: calculated history-cutoff at height %d, which is lower than the spork's root height %d", limitHeight, s.state.sporkRootBlockHeight)
		}
	}
	if limitHeight < blockSealedAtHead.Height {
		// we need to include extra blocks in sealing segment
		extraBlocksScraper := func(header *flow.Header) error {
			blockID := header.ID()
			block, err := s.state.blocks.ByID(blockID)
			if err != nil {
				return fmt.Errorf("could not get block: %w", err)
			}

			err = builder.AddExtraBlock(block)
			if err != nil {
				return fmt.Errorf("could not add block to sealing segment: %w", err)
			}

			return nil
		}

		err = fork.TraverseBackward(s.state.headers, blockSealedAtHead.ParentID, extraBlocksScraper, fork.IncludingHeight(limitHeight))
		if err != nil {
			return nil, fmt.Errorf("could not traverse extra blocks for sealing segment: %w", err)
		}
	}

	segment, err := builder.SealingSegment()
	if err != nil {
		return nil, fmt.Errorf("could not build sealing segment: %w", err)
	}

	return segment, nil
}

func (s *Snapshot) Descendants() ([]flow.Identifier, error) {
	descendants, err := s.descendants(s.blockID)
	if err != nil {
		return nil, fmt.Errorf("failed to traverse the descendants tree of block %v: %w", s.blockID, err)
	}
	return descendants, nil
}

func (s *Snapshot) lookupChildren(blockID flow.Identifier) ([]flow.Identifier, error) {
	var children flow.IdentifierList
	err := s.state.db.View(procedure.LookupBlockChildren(blockID, &children))
	if err != nil {
		return nil, fmt.Errorf("could not get children of block %v: %w", blockID, err)
	}
	return children, nil
}

func (s *Snapshot) descendants(blockID flow.Identifier) ([]flow.Identifier, error) {
	descendantIDs, err := s.lookupChildren(blockID)
	if err != nil {
		return nil, err
	}

	for _, descendantID := range descendantIDs {
		additionalIDs, err := s.descendants(descendantID)
		if err != nil {
			return nil, err
		}
		descendantIDs = append(descendantIDs, additionalIDs...)
	}
	return descendantIDs, nil
}

// RandomSource returns the seed for the current block's snapshot.
// Expected error returns:
// * storage.ErrNotFound is returned if the QC is unknown.
func (s *Snapshot) RandomSource() ([]byte, error) {
	qc, err := s.QuorumCertificate()
	if err != nil {
		return nil, err
	}
	randomSource, err := model.BeaconSignature(qc)
	if err != nil {
		return nil, fmt.Errorf("could not create seed from QC's signature: %w", err)
	}
	return randomSource, nil
}

func (s *Snapshot) Epochs() protocol.EpochQuery {
	return &EpochQuery{
		snap: s,
	}
}

func (s *Snapshot) Params() protocol.GlobalParams {
	return s.state.Params()
}

// EpochProtocolState returns the epoch part of dynamic protocol state that the Head block commits to.
// The compliance layer guarantees that only valid blocks are appended to the protocol state.
// Returns state.ErrUnknownSnapshotReference if snapshot reference block is unknown.
// All other errors should be treated as exceptions.
// For each block stored there should be a protocol state stored.
func (s *Snapshot) EpochProtocolState() (protocol.EpochProtocolState, error) {
	return s.state.protocolState.EpochStateAtBlockID(s.blockID)
}

// ProtocolState returns the dynamic protocol state that the Head block commits to.
// The compliance layer guarantees that only valid blocks are appended to the protocol state.
// Returns state.ErrUnknownSnapshotReference if snapshot reference block is unknown.
// All other errors should be treated as exceptions.
// For each block stored there should be a protocol state stored.
func (s *Snapshot) ProtocolState() (protocol.KVStoreReader, error) {
	return s.state.protocolState.KVStoreAtBlockID(s.blockID)
}

func (s *Snapshot) VersionBeacon() (*flow.SealedVersionBeacon, error) {
	head, err := s.state.headers.ByBlockID(s.blockID)
	if err != nil {
		return nil, err
	}

	return s.state.versionBeacons.Highest(head.Height)
}

// EpochQuery encapsulates querying epochs w.r.t. a snapshot.
type EpochQuery struct {
	snap *Snapshot
}

var _ protocol.EpochQuery = (*EpochQuery)(nil)

// Current returns the current epoch.
// No errors are expected during normal operation.
func (q *EpochQuery) Current() (protocol.CommittedEpoch, error) {
	// all errors returned from storage reads here are unexpected, because all
	// snapshots reside within a current epoch, which must be queryable
	epochState, err := q.snap.state.protocolState.EpochStateAtBlockID(q.snap.blockID)
	if err != nil {
		return nil, fmt.Errorf("could not get protocol state snapshot at block %x: %w", q.snap.blockID, err)
	}

	setup := epochState.EpochSetup()
	commit := epochState.EpochCommit()
	firstHeight, _, isFirstHeightKnown, _, err := q.retrieveEpochHeightBounds(setup.Counter)
	if err != nil {
		return nil, fmt.Errorf("could not get current epoch height bounds: %s", err.Error())
	}
	if isFirstHeightKnown {
		return inmem.NewEpochWithStartBoundary(setup, commit, epochState.EpochExtensions(), firstHeight), nil
	}
	return inmem.NewCommittedEpoch(setup, commit, epochState.EpochExtensions()), nil
}

// NextUnsafe returns the next epoch, if it has been set up but not yet committed.
// Error returns:
//   - protocol.ErrNextEpochNotSetup if the next epoch has not yet been set up as of the snapshot's reference block
//     (the reference block resides in the EpochStaking phase)
//   - protocol.ErrNextEpochAlreadyCommitted if the next epoch has already been committed at the snapshot's reference block
//     (the reference block resides in the EpochCommitted phase)
//   - generic error in case of unexpected critical internal corruption or bugs
func (q *EpochQuery) NextUnsafe() (protocol.TentativeEpoch, error) {
	epochState, err := q.snap.state.protocolState.EpochStateAtBlockID(q.snap.blockID)
	if err != nil {
		return nil, fmt.Errorf("could not get protocol state snapshot at block %x: %w", q.snap.blockID, err)
	}
	switch epochState.EpochPhase() {
	// if we are in the staking or fallback phase, the next epoch is not setup yet
	case flow.EpochPhaseStaking, flow.EpochPhaseFallback:
		return nil, protocol.ErrNextEpochNotSetup
	// if we are in setup phase, return a [protocol.TentativeEpoch] backed by the [flow.SetupEpoch] event
	case flow.EpochPhaseSetup:
		return inmem.NewSetupEpoch(epochState.Entry().NextEpochSetup), nil
	// if we are in committed phase, the caller should use the `NextCommitted` method instead, which we indicate by a sentinel error
	case flow.EpochPhaseCommitted:
		return nil, protocol.ErrNextEpochAlreadyCommitted
	default:
		return nil, fmt.Errorf("data corruption: unknown epoch phase implies malformed protocol state epoch data")
	}
}

// NextCommitted returns the next epoch as of this snapshot, only if it has been committed already.
// Error returns:
//   - protocol.ErrNextEpochNotCommitted if the next epoch has not yet been committed at the snapshot's reference block
//     (the reference block does not reside in the EpochCommitted phase)
//   - generic error in case of unexpected critical internal corruption or bugs
func (q *EpochQuery) NextCommitted() (protocol.CommittedEpoch, error) {
	epochState, err := q.snap.state.protocolState.EpochStateAtBlockID(q.snap.blockID)
	if err != nil {
		return nil, fmt.Errorf("could not get protocol state snapshot at block %x: %w", q.snap.blockID, err)
	}
	entry := epochState.Entry()

	switch epochState.EpochPhase() {
	// if we are in the staking or fallback phase, the next epoch is neither setup nor committed yet
	case flow.EpochPhaseStaking, flow.EpochPhaseFallback, flow.EpochPhaseSetup:
		return nil, protocol.ErrNextEpochNotCommitted
	case flow.EpochPhaseCommitted:
		// A protocol state snapshot is immutable and only represents the state as of the corresponding block. The
		// flow protocol implies that future epochs cannot have extensions, because in order to add extensions to
		// an epoch, we have to enter that epoch. Hence, `entry.NextEpoch.EpochExtensions` must be empty:
		if len(entry.NextEpoch.EpochExtensions) > 0 {
			return nil, irrecoverable.NewExceptionf("state with current epoch %d corrupted, because future epoch %d already has %d extensions",
				entry.CurrentEpochCommit.Counter, entry.NextEpochSetup.Counter, len(entry.NextEpoch.EpochExtensions))
		}
		return inmem.NewCommittedEpoch(entry.NextEpochSetup, entry.NextEpochCommit, entry.NextEpoch.EpochExtensions), nil
	default:
		return nil, fmt.Errorf("data corruption: unknown epoch phase implies malformed protocol state epoch data")
	}
}

// Previous returns the previous epoch. During the first epoch after the root
// block, this returns [protocol.ErrNoPreviousEpoch] (since there is no previous epoch).
// For all other epochs, returns the previous epoch.
func (q *EpochQuery) Previous() (protocol.CommittedEpoch, error) {
	epochState, err := q.snap.state.protocolState.EpochStateAtBlockID(q.snap.blockID)
	if err != nil {
		return nil, fmt.Errorf("could not get protocol state snapshot at block %x: %w", q.snap.blockID, err)
	}
	entry := epochState.Entry()

	// CASE 1: there is no previous epoch - this indicates we are in the first
	// epoch after a spork root or genesis block
	if !epochState.PreviousEpochExists() {
		return nil, protocol.ErrNoPreviousEpoch
	}

	// CASE 2: we are in any other epoch - retrieve the setup and commit events
	// for the previous epoch
	setup := entry.PreviousEpochSetup
	commit := entry.PreviousEpochCommit
	extensions := entry.PreviousEpoch.EpochExtensions

	firstHeight, finalHeight, firstHeightKnown, finalHeightKnown, err := q.retrieveEpochHeightBounds(setup.Counter)
	if err != nil {
		return nil, fmt.Errorf("could not get epoch height bounds: %w", err)
	}
	if firstHeightKnown && finalHeightKnown {
		// typical case - we usually know both boundaries for a past epoch
		return inmem.NewEpochWithStartAndEndBoundaries(setup, commit, extensions, firstHeight, finalHeight), nil
	}
	if firstHeightKnown && !finalHeightKnown {
		// this case is possible when the snapshot reference block is un-finalized
		// and is past an un-finalized epoch boundary
		return inmem.NewEpochWithStartBoundary(setup, commit, extensions, firstHeight), nil
	}
	if !firstHeightKnown && finalHeightKnown {
		// this case is possible when this node's lowest known block is after
		// the queried epoch's start boundary
		return inmem.NewEpochWithEndBoundary(setup, commit, extensions, finalHeight), nil
	}
	if !firstHeightKnown && !finalHeightKnown {
		// this case is possible when this node's lowest known block is after
		// the queried epoch's end boundary
		return inmem.NewCommittedEpoch(setup, commit, extensions), nil
	}
	return nil, fmt.Errorf("sanity check failed: impossible combination of boundaries for previous epoch")
}

// retrieveEpochHeightBounds retrieves the height bounds for an epoch.
// Height bounds are NOT fork-aware, and are only determined upon finalization.
//
// Since the protocol state's API is fork-aware, we may be querying an
// un-finalized block, as in the following example:
//
//	Epoch 1    Epoch 2
//	A <- B <-|- C <- D
//
// Suppose block B is the latest finalized block and we have queried block D.
// Then, the transition from epoch 1 to 2 has not been committed, because the first block of epoch 2 has not been finalized.
// In this case, the final block of Epoch 1, from the perspective of block D, is unknown.
// There are edge-case scenarios, where a different fork could exist (as illustrated below)
// that still adds additional blocks to Epoch 1.
//
//	Epoch 1      Epoch 2
//	A <- B <---|-- C <- D
//	     ^
//	     ╰ X <-|- X <- Y <- Z
//
// Returns:
//   - (0, 0, false, false, nil) if neither boundary is known
//   - (firstHeight, 0, true, false, nil) if epoch start boundary is known but end boundary is not known
//   - (firstHeight, finalHeight, true, true, nil) if epoch start and end boundary are known
//   - (0, finalHeight, false, true, nil) if epoch start boundary is known but end boundary is not known
//
// No errors are expected during normal operation.
func (q *EpochQuery) retrieveEpochHeightBounds(epoch uint64) (
	firstHeight, finalHeight uint64,
	isFirstHeightKnown, isLastHeightKnown bool,
	err error,
) {
	err = q.snap.state.db.View(func(tx *badger.Txn) error {
		// Retrieve the epoch's first height
		err = operation.RetrieveEpochFirstHeight(epoch, &firstHeight)(tx)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				isFirstHeightKnown = false // unknown boundary
			} else {
				return err // unexpected error
			}
		} else {
			isFirstHeightKnown = true // known boundary
		}

		var subsequentEpochFirstHeight uint64
		err = operation.RetrieveEpochFirstHeight(epoch+1, &subsequentEpochFirstHeight)(tx)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				isLastHeightKnown = false // unknown boundary
			} else {
				return err // unexpected error
			}
		} else { // known boundary
			isLastHeightKnown = true
			finalHeight = subsequentEpochFirstHeight - 1
		}

		return nil
	})
	if err != nil {
		return 0, 0, false, false, err
	}
	return firstHeight, finalHeight, isFirstHeightKnown, isLastHeightKnown, nil
}
