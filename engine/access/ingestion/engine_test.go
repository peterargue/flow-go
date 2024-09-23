package ingestion

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v2"
	execproto "github.com/onflow/flow/protobuf/go/flow/execution"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	hotmodel "github.com/onflow/flow-go/consensus/hotstuff/model"
	accessmock "github.com/onflow/flow-go/engine/access/mock"
	"github.com/onflow/flow-go/engine/access/rpc/backend"
	connectionmock "github.com/onflow/flow-go/engine/access/rpc/connection/mock"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/model/flow/filter"
	"github.com/onflow/flow-go/module"
	"github.com/onflow/flow-go/module/component"
	"github.com/onflow/flow-go/module/counters"
	downloadermock "github.com/onflow/flow-go/module/executiondatasync/execution_data/mock"
	"github.com/onflow/flow-go/module/irrecoverable"
	"github.com/onflow/flow-go/module/mempool/stdmap"
	"github.com/onflow/flow-go/module/metrics"
	modulemock "github.com/onflow/flow-go/module/mock"
	"github.com/onflow/flow-go/module/signature"
	"github.com/onflow/flow-go/module/state_synchronization/indexer"
	"github.com/onflow/flow-go/network/channels"
	"github.com/onflow/flow-go/network/mocknetwork"
	protocol "github.com/onflow/flow-go/state/protocol/mock"
	storerr "github.com/onflow/flow-go/storage"
	bstorage "github.com/onflow/flow-go/storage/badger"
	storage "github.com/onflow/flow-go/storage/mock"
	"github.com/onflow/flow-go/utils/unittest"
	"github.com/onflow/flow-go/utils/unittest/mocks"
)

const expectedErrorMsg = "expected test error"

type Suite struct {
	suite.Suite

	// protocol state
	proto struct {
		state    *protocol.FollowerState
		snapshot *protocol.Snapshot
		params   *protocol.Params
	}

	me              *modulemock.Local
	net             *mocknetwork.Network
	request         *modulemock.Requester
	obsIdentity     *flow.Identity
	provider        *mocknetwork.Engine
	blocks          *storage.Blocks
	headers         *storage.Headers
	collections     *storage.Collections
	transactions    *storage.Transactions
	receipts        *storage.ExecutionReceipts
	results         *storage.ExecutionResults
	seals           *storage.Seals
	txErrorMessages *storage.TransactionResultErrorMessages
	conduit         *mocknetwork.Conduit
	downloader      *downloadermock.Downloader
	sealedBlock     *flow.Header
	finalizedBlock  *flow.Header
	log             zerolog.Logger
	blockMap        map[uint64]*flow.Block
	rootBlock       flow.Block

	enNodeIDs flow.IdentifierList
	backend   *backend.Backend

	collectionExecutedMetric *indexer.CollectionExecutedMetricImpl

	ctx    context.Context
	cancel context.CancelFunc

	db                  *badger.DB
	dbDir               string
	lastFullBlockHeight *counters.PersistentStrictMonotonicCounter
}

func TestIngestEngine(t *testing.T) {
	suite.Run(t, new(Suite))
}

// TearDownTest stops the engine and cleans up the db
func (s *Suite) TearDownTest() {
	s.cancel()
	err := os.RemoveAll(s.dbDir)
	s.Require().NoError(err)
}

func (s *Suite) SetupTest() {
	s.log = zerolog.New(os.Stderr)
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.db, s.dbDir = unittest.TempBadgerDB(s.T())

	s.obsIdentity = unittest.IdentityFixture(unittest.WithRole(flow.RoleAccess))

	s.blocks = storage.NewBlocks(s.T())
	// mock out protocol state
	s.proto.state = new(protocol.FollowerState)
	s.proto.snapshot = new(protocol.Snapshot)
	s.proto.params = new(protocol.Params)
	s.finalizedBlock = unittest.BlockHeaderFixture(unittest.WithHeaderHeight(0))
	s.proto.state.On("Identity").Return(s.obsIdentity, nil)
	s.proto.state.On("Final").Return(s.proto.snapshot, nil)
	s.proto.state.On("Params").Return(s.proto.params)
	s.proto.snapshot.On("Head").Return(
		func() *flow.Header {
			return s.finalizedBlock
		},
		nil,
	).Maybe()

	s.me = modulemock.NewLocal(s.T())
	s.me.On("NodeID").Return(s.obsIdentity.NodeID).Maybe()
	s.net = mocknetwork.NewNetwork(s.T())
	conduit := mocknetwork.NewConduit(s.T())
	s.net.On("Register", channels.ReceiveReceipts, mock.Anything).
		Return(conduit, nil).
		Once()
	s.request = modulemock.NewRequester(s.T())

	s.provider = mocknetwork.NewEngine(s.T())
	s.blocks = storage.NewBlocks(s.T())
	s.headers = storage.NewHeaders(s.T())
	s.collections = new(storage.Collections)
	s.receipts = new(storage.ExecutionReceipts)
	s.transactions = new(storage.Transactions)
	s.results = new(storage.ExecutionResults)
	s.txErrorMessages = new(storage.TransactionResultErrorMessages)
	collectionsToMarkFinalized, err := stdmap.NewTimes(100)
	require.NoError(s.T(), err)
	collectionsToMarkExecuted, err := stdmap.NewTimes(100)
	require.NoError(s.T(), err)
	blocksToMarkExecuted, err := stdmap.NewTimes(100)
	require.NoError(s.T(), err)

	s.proto.state.On("Identity").Return(s.obsIdentity, nil)
	s.proto.state.On("Params").Return(s.proto.params)

	blockCount := 5
	s.blockMap = make(map[uint64]*flow.Block, blockCount)
	s.rootBlock = unittest.BlockFixture()
	s.rootBlock.Header.Height = 0
	parent := s.rootBlock.Header

	for i := 0; i < blockCount; i++ {
		block := unittest.BlockWithParentFixture(parent)
		// update for next iteration
		parent = block.Header
		s.blockMap[block.Header.Height] = block
	}
	s.finalizedBlock = parent

	s.blocks.On("ByHeight", mock.AnythingOfType("uint64")).Return(
		mocks.ConvertStorageOutput(
			mocks.StorageMapGetter(s.blockMap),
			func(block *flow.Block) *flow.Block { return block },
		),
	).Maybe()

	s.proto.snapshot.On("Head").Return(
		func() *flow.Header {
			return s.finalizedBlock
		},
		nil,
	).Maybe()
	s.proto.state.On("Final").Return(s.proto.snapshot, nil)

	s.collectionExecutedMetric, err = indexer.NewCollectionExecutedMetricImpl(
		s.log,
		metrics.NewNoopCollector(),
		collectionsToMarkFinalized,
		collectionsToMarkExecuted,
		blocksToMarkExecuted,
		s.collections,
		s.blocks,
	)
	require.NoError(s.T(), err)
}

// initIngestionEngine create new instance of ingestion engine and waits when it starts
func (s *Suite) initIngestionEngine(ctx irrecoverable.SignalerContext) *Engine {
	processedHeight := bstorage.NewConsumerProgress(s.db, module.ConsumeProgressIngestionEngineBlockHeight)

	var err error
	s.lastFullBlockHeight, err = counters.NewPersistentStrictMonotonicCounter(
		bstorage.NewConsumerProgress(s.db, module.ConsumeProgressLastFullBlockHeight),
		s.finalizedBlock.Height,
	)
	require.NoError(s.T(), err)

	eng, err := New(
		s.log,
		s.net,
		s.proto.state,
		s.me,
		s.request,
		s.blocks,
		s.headers,
		s.collections,
		s.transactions,
		s.results,
		s.receipts,
		s.txErrorMessages,
		s.collectionExecutedMetric,
		processedHeight,
		s.lastFullBlockHeight,
		s.backend,
		s.enNodeIDs.Strings(),
		nil,
	)

	require.NoError(s.T(), err)

	eng.ComponentManager.Start(ctx)
	<-eng.Ready()

	return eng
}

// mockCollectionsForBlock mocks collections for block
func (s *Suite) mockCollectionsForBlock(block flow.Block) {
	// we should query the block once and index the guarantee payload once
	for _, g := range block.Payload.Guarantees {
		collection := unittest.CollectionFixture(1)
		light := collection.Light()
		s.collections.On("LightByID", g.CollectionID).Return(&light, nil).Twice()
	}
}

// generateBlock prepares block with payload and specified guarantee.SignerIndices
func (s *Suite) generateBlock(clusterCommittee flow.IdentitySkeletonList, snap *protocol.Snapshot) flow.Block {
	block := unittest.BlockFixture()
	block.SetPayload(unittest.PayloadFixture(
		unittest.WithGuarantees(unittest.CollectionGuaranteesFixture(4)...),
		unittest.WithExecutionResults(unittest.ExecutionResultFixture()),
		unittest.WithSeals(unittest.Seal.Fixture()),
	))

	refBlockID := unittest.IdentifierFixture()
	for _, guarantee := range block.Payload.Guarantees {
		guarantee.ReferenceBlockID = refBlockID
		// guarantee signers must be cluster committee members, so that access will fetch collection from
		// the signers that are specified by guarantee.SignerIndices
		indices, err := signature.EncodeSignersToIndices(clusterCommittee.NodeIDs(), clusterCommittee.NodeIDs())
		require.NoError(s.T(), err)
		guarantee.SignerIndices = indices
	}

	s.proto.state.On("AtBlockID", refBlockID).Return(snap)

	return block
}

// TestOnFinalizedBlock checks that when a block is received, a request for each individual collection is made
func (s *Suite) TestOnFinalizedBlockSingle() {
	cluster := new(protocol.Cluster)
	epoch := new(protocol.Epoch)
	epochs := new(protocol.EpochQuery)
	snap := new(protocol.Snapshot)

	epoch.On("ClusterByChainID", mock.Anything).Return(cluster, nil)
	epochs.On("Current").Return(epoch)
	snap.On("Epochs").Return(epochs)

	// prepare cluster committee members
	clusterCommittee := unittest.IdentityListFixture(32 * 4).Filter(filter.HasRole[flow.Identity](flow.RoleCollection)).ToSkeleton()
	cluster.On("Members").Return(clusterCommittee, nil)

	irrecoverableCtx := irrecoverable.NewMockSignalerContext(s.T(), s.ctx)
	eng := s.initIngestionEngine(irrecoverableCtx)

	block := s.generateBlock(clusterCommittee, snap)
	block.Header.Height = s.finalizedBlock.Height + 1
	s.blockMap[block.Header.Height] = &block
	s.mockCollectionsForBlock(block)
	s.finalizedBlock = block.Header

	hotstuffBlock := hotmodel.Block{
		BlockID: block.ID(),
	}

	// expect that the block storage is indexed with each of the collection guarantee
	s.blocks.On("IndexBlockForCollections", block.ID(), []flow.Identifier(flow.GetIDs(block.Payload.Guarantees))).Return(nil).Once()
	for _, seal := range block.Payload.Seals {
		s.results.On("Index", seal.BlockID, seal.ResultID).Return(nil).Once()
	}

	missingCollectionCount := 4
	wg := sync.WaitGroup{}
	wg.Add(missingCollectionCount)

	for _, cg := range block.Payload.Guarantees {
		s.request.On("EntityByID", cg.CollectionID, mock.Anything).Return().Run(func(args mock.Arguments) {
			// Ensure the test does not complete its work faster than necessary
			wg.Done()
		}).Once()
	}

	// process the block through the finalized callback
	eng.OnFinalizedBlock(&hotstuffBlock)

	unittest.RequireReturnsBefore(s.T(), wg.Wait, 100*time.Millisecond, "expect to process new block before timeout")

	// assert that the block was retrieved and all collections were requested
	s.headers.AssertExpectations(s.T())
	s.request.AssertNumberOfCalls(s.T(), "EntityByID", len(block.Payload.Guarantees))
	s.results.AssertNumberOfCalls(s.T(), "Index", len(block.Payload.Seals))
}

// TestOnFinalizedBlockSeveralBlocksAhead checks OnFinalizedBlock with a block several blocks newer than the last block processed
func (s *Suite) TestOnFinalizedBlockSeveralBlocksAhead() {
	cluster := new(protocol.Cluster)
	epoch := new(protocol.Epoch)
	epochs := new(protocol.EpochQuery)
	snap := new(protocol.Snapshot)

	epoch.On("ClusterByChainID", mock.Anything).Return(cluster, nil)
	epochs.On("Current").Return(epoch)
	snap.On("Epochs").Return(epochs)

	// prepare cluster committee members
	clusterCommittee := unittest.IdentityListFixture(32 * 4).Filter(filter.HasRole[flow.Identity](flow.RoleCollection)).ToSkeleton()
	cluster.On("Members").Return(clusterCommittee, nil)

	irrecoverableCtx := irrecoverable.NewMockSignalerContext(s.T(), s.ctx)
	eng := s.initIngestionEngine(irrecoverableCtx)

	newBlocksCount := 3
	startHeight := s.finalizedBlock.Height + 1
	blocks := make([]flow.Block, newBlocksCount)

	// generate the test blocks, cgs and collections
	for i := 0; i < newBlocksCount; i++ {
		block := s.generateBlock(clusterCommittee, snap)
		block.Header.Height = startHeight + uint64(i)
		s.blockMap[block.Header.Height] = &block
		blocks[i] = block
		s.mockCollectionsForBlock(block)
		s.finalizedBlock = block.Header
	}

	// latest of all the new blocks which are newer than the last block processed
	latestBlock := blocks[2]

	// block several blocks newer than the last block processed
	hotstuffBlock := hotmodel.Block{
		BlockID: latestBlock.ID(),
	}

	missingCollectionCountPerBlock := 4
	wg := sync.WaitGroup{}
	wg.Add(missingCollectionCountPerBlock * newBlocksCount)

	// expected all new blocks after last block processed
	for _, block := range blocks {
		s.blocks.On("IndexBlockForCollections", block.ID(), []flow.Identifier(flow.GetIDs(block.Payload.Guarantees))).Return(nil).Once()

		for _, cg := range block.Payload.Guarantees {
			s.request.On("EntityByID", cg.CollectionID, mock.Anything).Return().Run(func(args mock.Arguments) {
				// Ensure the test does not complete its work faster than necessary, so we can check all expected results
				wg.Done()
			}).Once()
		}
		for _, seal := range block.Payload.Seals {
			s.results.On("Index", seal.BlockID, seal.ResultID).Return(nil).Once()
		}
	}

	eng.OnFinalizedBlock(&hotstuffBlock)

	unittest.RequireReturnsBefore(s.T(), wg.Wait, 100*time.Millisecond, "expect to process all blocks before timeout")

	expectedEntityByIDCalls := 0
	expectedIndexCalls := 0
	for _, block := range blocks {
		expectedEntityByIDCalls += len(block.Payload.Guarantees)
		expectedIndexCalls += len(block.Payload.Seals)
	}

	s.headers.AssertExpectations(s.T())
	s.blocks.AssertNumberOfCalls(s.T(), "IndexBlockForCollections", newBlocksCount)
	s.request.AssertNumberOfCalls(s.T(), "EntityByID", expectedEntityByIDCalls)
	s.results.AssertNumberOfCalls(s.T(), "Index", expectedIndexCalls)
}

// TestOnCollection checks that when a Collection is received, it is persisted
func (s *Suite) TestOnCollection() {
	irrecoverableCtx := irrecoverable.NewMockSignalerContext(s.T(), s.ctx)
	s.initIngestionEngine(irrecoverableCtx)

	collection := unittest.CollectionFixture(5)
	light := collection.Light()

	// we should store the light collection and index its transactions
	s.collections.On("StoreLightAndIndexByTransaction", &light).Return(nil).Once()

	// for each transaction in the collection, we should store it
	needed := make(map[flow.Identifier]struct{})
	for _, txID := range light.Transactions {
		needed[txID] = struct{}{}
	}
	s.transactions.On("Store", mock.Anything).Return(nil).Run(
		func(args mock.Arguments) {
			tx := args.Get(0).(*flow.TransactionBody)
			_, pending := needed[tx.ID()]
			s.Assert().True(pending, "tx not pending (%x)", tx.ID())
		},
	)

	err := indexer.HandleCollection(&collection, s.collections, s.transactions, s.log, s.collectionExecutedMetric)
	require.NoError(s.T(), err)

	// check that the collection was stored and indexed, and we stored all transactions
	s.collections.AssertExpectations(s.T())
	s.transactions.AssertNumberOfCalls(s.T(), "Store", len(collection.Transactions))
}

// TestExecutionReceiptsAreIndexed checks that execution receipts are properly indexed
func (s *Suite) TestExecutionReceiptsAreIndexed() {
	irrecoverableCtx := irrecoverable.NewMockSignalerContext(s.T(), s.ctx)
	eng := s.initIngestionEngine(irrecoverableCtx)

	originID := unittest.IdentifierFixture()
	collection := unittest.CollectionFixture(5)
	light := collection.Light()

	// we should store the light collection and index its transactions
	s.collections.On("StoreLightAndIndexByTransaction", &light).Return(nil).Once()
	block := &flow.Block{
		Header:  &flow.Header{Height: 0},
		Payload: &flow.Payload{Guarantees: []*flow.CollectionGuarantee{}},
	}
	s.blocks.On("ByID", mock.Anything).Return(block, nil)

	// for each transaction in the collection, we should store it
	needed := make(map[flow.Identifier]struct{})
	for _, txID := range light.Transactions {
		needed[txID] = struct{}{}
	}
	s.transactions.On("Store", mock.Anything).Return(nil).Run(
		func(args mock.Arguments) {
			tx := args.Get(0).(*flow.TransactionBody)
			_, pending := needed[tx.ID()]
			s.Assert().True(pending, "tx not pending (%x)", tx.ID())
		},
	)
	er1 := unittest.ExecutionReceiptFixture()
	er2 := unittest.ExecutionReceiptFixture()

	s.receipts.On("Store", mock.Anything).Return(nil)
	s.blocks.On("ByID", er1.ExecutionResult.BlockID).Return(nil, storerr.ErrNotFound)

	s.receipts.On("Store", mock.Anything).Return(nil)
	s.blocks.On("ByID", er2.ExecutionResult.BlockID).Return(nil, storerr.ErrNotFound)

	err := eng.handleExecutionReceipt(originID, er1)
	require.NoError(s.T(), err)

	err = eng.handleExecutionReceipt(originID, er2)
	require.NoError(s.T(), err)

	s.receipts.AssertExpectations(s.T())
	s.results.AssertExpectations(s.T())
	s.receipts.AssertExpectations(s.T())
}

// TestOnCollectionDuplicate checks that when a duplicate collection is received, the node doesn't
// crash but just ignores its transactions.
func (s *Suite) TestOnCollectionDuplicate() {
	irrecoverableCtx := irrecoverable.NewMockSignalerContext(s.T(), s.ctx)
	s.initIngestionEngine(irrecoverableCtx)

	collection := unittest.CollectionFixture(5)
	light := collection.Light()

	// we should store the light collection and index its transactions
	s.collections.On("StoreLightAndIndexByTransaction", &light).Return(storerr.ErrAlreadyExists).Once()

	// for each transaction in the collection, we should store it
	needed := make(map[flow.Identifier]struct{})
	for _, txID := range light.Transactions {
		needed[txID] = struct{}{}
	}
	s.transactions.On("Store", mock.Anything).Return(nil).Run(
		func(args mock.Arguments) {
			tx := args.Get(0).(*flow.TransactionBody)
			_, pending := needed[tx.ID()]
			s.Assert().True(pending, "tx not pending (%x)", tx.ID())
		},
	)

	err := indexer.HandleCollection(&collection, s.collections, s.transactions, s.log, s.collectionExecutedMetric)
	require.NoError(s.T(), err)

	// check that the collection was stored and indexed, and we stored all transactions
	s.collections.AssertExpectations(s.T())
	s.transactions.AssertNotCalled(s.T(), "Store", "should not store any transactions")
}

// TestRequestMissingCollections tests that the all missing collections are requested on the call to requestMissingCollections
func (s *Suite) TestRequestMissingCollections() {
	irrecoverableCtx := irrecoverable.NewMockSignalerContext(s.T(), s.ctx)
	eng := s.initIngestionEngine(irrecoverableCtx)

	blkCnt := 3
	startHeight := uint64(1000)

	// prepare cluster committee members
	clusterCommittee := unittest.IdentityListFixture(32 * 4).Filter(filter.HasRole[flow.Identity](flow.RoleCollection)).ToSkeleton()

	// generate the test blocks and collections
	var collIDs []flow.Identifier
	refBlockID := unittest.IdentifierFixture()
	for i := 0; i < blkCnt; i++ {
		block := unittest.BlockFixture()
		block.SetPayload(unittest.PayloadFixture(
			unittest.WithGuarantees(
				unittest.CollectionGuaranteesFixture(4, unittest.WithCollRef(refBlockID))...),
		))
		// some blocks may not be present hence add a gap
		height := startHeight + uint64(i)
		block.Header.Height = height
		s.blockMap[block.Header.Height] = &block
		s.finalizedBlock = block.Header

		for _, c := range block.Payload.Guarantees {
			collIDs = append(collIDs, c.CollectionID)
			c.ReferenceBlockID = refBlockID

			// guarantee signers must be cluster committee members, so that access will fetch collection from
			// the signers that are specified by guarantee.SignerIndices
			indices, err := signature.EncodeSignersToIndices(clusterCommittee.NodeIDs(), clusterCommittee.NodeIDs())
			require.NoError(s.T(), err)
			c.SignerIndices = indices
		}
	}

	// consider collections are missing for all blocks
	err := s.lastFullBlockHeight.Set(startHeight - 1)
	s.Require().NoError(err)

	// consider the last test block as the head

	// p is the probability of not receiving the collection before the next poll and it
	// helps simulate the slow trickle of the requested collections being received
	var p float32

	// rcvdColl is the map simulating the collection storage key-values
	rcvdColl := make(map[flow.Identifier]struct{})

	// for the first lookup call for each collection, it will be reported as missing from db
	// for the subsequent calls, it will be reported as present with the probability p
	s.collections.On("LightByID", mock.Anything).Return(
		func(cID flow.Identifier) *flow.LightCollection {
			return nil // the actual collection object return is never really read
		},
		func(cID flow.Identifier) error {
			if _, ok := rcvdColl[cID]; ok {
				return nil
			}
			if rand.Float32() >= p {
				rcvdColl[cID] = struct{}{}
			}
			return storerr.ErrNotFound
		}).
		// simulate some db i/o contention
		After(time.Millisecond * time.Duration(rand.Intn(5)))

	// setup the requester engine mock
	// entityByID should be called once per collection
	for _, c := range collIDs {
		s.request.On("EntityByID", c, mock.Anything).Return()
	}
	// force should be called once
	s.request.On("Force").Return()

	cluster := new(protocol.Cluster)
	cluster.On("Members").Return(clusterCommittee, nil)
	epoch := new(protocol.Epoch)
	epoch.On("ClusterByChainID", mock.Anything).Return(cluster, nil)
	epochs := new(protocol.EpochQuery)
	epochs.On("Current").Return(epoch)
	snap := new(protocol.Snapshot)
	snap.On("Epochs").Return(epochs)
	s.proto.state.On("AtBlockID", refBlockID).Return(snap)

	assertExpectations := func() {
		s.request.AssertExpectations(s.T())
		s.collections.AssertExpectations(s.T())
		s.proto.snapshot.AssertExpectations(s.T())
		s.blocks.AssertExpectations(s.T())
	}

	// test 1 - collections are not received before timeout
	s.Run("timeout before all missing collections are received", func() {

		// simulate that collection are never received
		p = 1

		// timeout after 3 db polls
		ctx, cancel := context.WithTimeout(context.Background(), 100*defaultCollectionCatchupDBPollInterval)
		defer cancel()

		err := eng.requestMissingCollections(ctx)

		require.Error(s.T(), err)
		require.Contains(s.T(), err.Error(), "context deadline exceeded")

		assertExpectations()
	})
	// test 2 - all collections are eventually received before the deadline
	s.Run("all missing collections are received", func() {

		// 90% of the time, collections are reported as not received when the collection storage is queried
		p = 0.9

		ctx, cancel := context.WithTimeout(context.Background(), defaultCollectionCatchupTimeout)
		defer cancel()

		err := eng.requestMissingCollections(ctx)

		require.NoError(s.T(), err)
		require.Len(s.T(), rcvdColl, len(collIDs))

		assertExpectations()
	})
}

// TestProcessBackgroundCalls tests that updateLastFullBlockReceivedIndex and checkMissingCollections
// function calls keep the FullBlockIndex up-to-date and request collections if blocks with missing
// collections exceed the threshold.
func (s *Suite) TestProcessBackgroundCalls() {
	irrecoverableCtx := irrecoverable.NewMockSignalerContext(s.T(), s.ctx)
	eng := s.initIngestionEngine(irrecoverableCtx)

	blkCnt := 3
	collPerBlk := 10
	startHeight := uint64(1000)
	blocks := make([]flow.Block, blkCnt)
	collMap := make(map[flow.Identifier]*flow.LightCollection, blkCnt*collPerBlk)

	// prepare cluster committee members
	clusterCommittee := unittest.IdentityListFixture(32 * 4).Filter(filter.HasRole[flow.Identity](flow.RoleCollection)).ToSkeleton()

	refBlockID := unittest.IdentifierFixture()
	// generate the test blocks, cgs and collections
	for i := 0; i < blkCnt; i++ {
		guarantees := make([]*flow.CollectionGuarantee, collPerBlk)
		for j := 0; j < collPerBlk; j++ {
			coll := unittest.CollectionFixture(2).Light()
			collMap[coll.ID()] = &coll
			cg := unittest.CollectionGuaranteeFixture(func(cg *flow.CollectionGuarantee) {
				cg.CollectionID = coll.ID()
				cg.ReferenceBlockID = refBlockID
			})

			// guarantee signers must be cluster committee members, so that access will fetch collection from
			// the signers that are specified by guarantee.SignerIndices
			indices, err := signature.EncodeSignersToIndices(clusterCommittee.NodeIDs(), clusterCommittee.NodeIDs())
			require.NoError(s.T(), err)
			cg.SignerIndices = indices
			guarantees[j] = cg
		}
		block := unittest.BlockFixture()
		block.SetPayload(unittest.PayloadFixture(unittest.WithGuarantees(guarantees...)))
		// set the height
		height := startHeight + uint64(i)
		block.Header.Height = height
		s.blockMap[block.Header.Height] = &block
		blocks[i] = block
		s.finalizedBlock = block.Header
	}

	finalizedHeight := s.finalizedBlock.Height

	cluster := new(protocol.Cluster)
	cluster.On("Members").Return(clusterCommittee, nil)
	epoch := new(protocol.Epoch)
	epoch.On("ClusterByChainID", mock.Anything).Return(cluster, nil)
	epochs := new(protocol.EpochQuery)
	epochs.On("Current").Return(epoch)
	snap := new(protocol.Snapshot)
	snap.On("Epochs").Return(epochs)
	s.proto.state.On("AtBlockID", refBlockID).Return(snap)

	// blkMissingColl controls which collections are reported as missing by the collections storage mock
	blkMissingColl := make([]bool, blkCnt)
	for i := 0; i < blkCnt; i++ {
		blkMissingColl[i] = false
		for _, cg := range blocks[i].Payload.Guarantees {
			j := i
			s.collections.On("LightByID", cg.CollectionID).Return(
				func(cID flow.Identifier) *flow.LightCollection {
					return collMap[cID]
				},
				func(cID flow.Identifier) error {
					if blkMissingColl[j] {
						return storerr.ErrNotFound
					}
					return nil
				})
		}
	}

	rootBlk := blocks[0]

	// root block is the last complete block
	err := s.lastFullBlockHeight.Set(rootBlk.Header.Height)
	s.Require().NoError(err)

	s.Run("missing collections are requested when count exceeds defaultMissingCollsForBlkThreshold", func() {
		// lower the block threshold to request missing collections
		defaultMissingCollsForBlkThreshold = 2

		// mark all blocks beyond the root block as incomplete
		for i := 1; i < blkCnt; i++ {
			blkMissingColl[i] = true
			// setup receive engine expectations
			for _, cg := range blocks[i].Payload.Guarantees {
				s.request.On("EntityByID", cg.CollectionID, mock.Anything).Return().Once()
			}
		}

		err := eng.checkMissingCollections()
		s.Require().NoError(err)

		// assert that missing collections are requested
		s.request.AssertExpectations(s.T())

		// last full blk index is not advanced
		s.blocks.AssertExpectations(s.T()) // no new call to UpdateLastFullBlockHeight should be made
	})

	s.Run("missing collections are requested when count exceeds defaultMissingCollsForAgeThreshold", func() {
		// lower the height threshold to request missing collections
		defaultMissingCollsForAgeThreshold = 1

		// raise the block threshold to ensure it does not trigger missing collection request
		defaultMissingCollsForBlkThreshold = blkCnt + 1

		// mark all blocks beyond the root block as incomplete
		for i := 1; i < blkCnt; i++ {
			blkMissingColl[i] = true
			// setup receive engine expectations
			for _, cg := range blocks[i].Payload.Guarantees {
				s.request.On("EntityByID", cg.CollectionID, mock.Anything).Return().Once()
			}
		}

		err := eng.checkMissingCollections()
		s.Require().NoError(err)

		// assert that missing collections are requested
		s.request.AssertExpectations(s.T())

		// last full blk index is not advanced
		s.blocks.AssertExpectations(s.T()) // not new call to UpdateLastFullBlockHeight should be made
	})

	s.Run("missing collections are not requested if defaultMissingCollsForBlkThreshold not reached", func() {
		// raise the thresholds to avoid requesting missing collections
		defaultMissingCollsForAgeThreshold = 3
		defaultMissingCollsForBlkThreshold = 3

		// mark all blocks beyond the root block as incomplete
		for i := 1; i < blkCnt; i++ {
			blkMissingColl[i] = true
		}

		err := eng.checkMissingCollections()
		s.Require().NoError(err)

		// assert that missing collections are not requested even though there are collections missing
		s.request.AssertExpectations(s.T())

		// last full blk index is not advanced
		s.blocks.AssertExpectations(s.T()) // not new call to UpdateLastFullBlockHeight should be made
	})

	// create new block
	finalizedBlk := unittest.BlockFixture()
	height := blocks[blkCnt-1].Header.Height + 1
	finalizedBlk.Header.Height = height
	s.blockMap[height] = &finalizedBlk

	finalizedHeight = finalizedBlk.Header.Height
	s.finalizedBlock = finalizedBlk.Header

	blockBeforeFinalized := blocks[blkCnt-1].Header

	s.Run("full block height index is advanced if newer full blocks are discovered", func() {
		// set lastFullBlockHeight to block
		err = s.lastFullBlockHeight.Set(blockBeforeFinalized.Height)
		s.Require().NoError(err)

		err = eng.updateLastFullBlockReceivedIndex()
		s.Require().NoError(err)
		s.Require().Equal(finalizedHeight, s.lastFullBlockHeight.Value())
		s.Require().NoError(err)

		s.blocks.AssertExpectations(s.T())
	})

	s.Run("full block height index is not advanced beyond finalized blocks", func() {
		err = eng.updateLastFullBlockReceivedIndex()
		s.Require().NoError(err)

		s.Require().Equal(finalizedHeight, s.lastFullBlockHeight.Value())
		s.blocks.AssertExpectations(s.T())
	})
}

type mockCloser struct{}

func (mc *mockCloser) Close() error { return nil }

// TestTransactionResultErrorMessagesAreFetched checks that transaction result error messages
// are properly fetched from the execution nodes, processed, and stored in the protocol database.
func (s *Suite) TestTransactionResultErrorMessagesAreFetched() {
	irrecoverableCtx := irrecoverable.NewMockSignalerContext(s.T(), s.ctx)

	block := unittest.BlockFixture()
	blockId := block.ID()

	// Create a mock execution client to simulate communication with execution nodes.
	execClient := new(accessmock.ExecutionAPIClient)

	// Create identities for 1 execution nodes.
	enIdentities := unittest.IdentityListFixture(1, unittest.WithRole(flow.RoleExecution))
	s.enNodeIDs = enIdentities.NodeIDs()

	// create a mock connection factory
	connFactory := connectionmock.NewConnectionFactory(s.T())
	connFactory.On("GetExecutionAPIClient", mock.Anything).Return(execClient, &mockCloser{}, nil)

	// Mock the protocol snapshot to return fixed execution node IDs.
	_, fixedENIDs := s.setupReceipts(&block)
	s.proto.snapshot.On("Identities", mock.Anything).Return(fixedENIDs, nil)

	// Mock the finalized root block header with height 0.
	header := unittest.BlockHeaderFixture(unittest.WithHeaderHeight(0))
	s.proto.params.On("FinalizedRoot").Return(header, nil)

	// Initialize the backend with the mocked state, blocks, headers, transactions, etc.
	var err error
	s.backend, err = backend.New(backend.Params{
		State:                 s.proto.state,
		Blocks:                s.blocks,
		Headers:               s.headers,
		Transactions:          s.transactions,
		ExecutionReceipts:     s.receipts,
		ExecutionResults:      s.results,
		ConnFactory:           connFactory,
		MaxHeightRange:        backend.DefaultMaxHeightRange,
		FixedExecutionNodeIDs: s.enNodeIDs.Strings(),
		Log:                   s.log,
		SnapshotHistoryLimit:  backend.DefaultSnapshotHistoryLimit,
		Communicator:          backend.NewNodeCommunicator(false),
		ScriptExecutionMode:   backend.IndexQueryModeExecutionNodesOnly,
		TxResultQueryMode:     backend.IndexQueryModeExecutionNodesOnly,
		ChainID:               flow.Testnet,
	})
	require.NoError(s.T(), err)

	// Initialize the ingestion engine with the irrecoverable context.
	eng := s.initIngestionEngine(irrecoverableCtx)

	// Create mock transaction results with a mix of failed and non-failed transactions.
	resultsByBlockID := make([]flow.LightTransactionResult, 0)
	for i := 0; i < 5; i++ {
		resultsByBlockID = append(resultsByBlockID, flow.LightTransactionResult{
			TransactionID:   unittest.IdentifierFixture(),
			Failed:          i%2 == 0, // create a mix of failed and non-failed transactions
			ComputationUsed: 0,
		})
	}

	// Prepare a request to fetch transaction error messages by block ID from execution nodes.
	exeEventReq := &execproto.GetTransactionErrorMessagesByBlockIDRequest{
		BlockId: blockId[:],
	}
	exeErrMessagesResp := &execproto.GetTransactionErrorMessagesResponse{}

	// Prepare the expected transaction error messages that should be stored.
	var expectedStoreTxErrorMessages []flow.TransactionResultErrorMessage

	for i, result := range resultsByBlockID {
		if result.Failed {
			errMsg := fmt.Sprintf("%s.%s", expectedErrorMsg, result.TransactionID)
			exeErrMessagesResp.Results = append(exeErrMessagesResp.Results, &execproto.GetTransactionErrorMessagesResponse_Result{
				TransactionId: result.TransactionID[:],
				ErrorMessage:  errMsg,
				Index:         uint32(i),
			})

			expectedStoreTxErrorMessages = append(expectedStoreTxErrorMessages,
				flow.TransactionResultErrorMessage{
					TransactionID: result.TransactionID,
					ErrorMessage:  errMsg,
					Index:         uint32(i),
					ExecutorID:    fixedENIDs.NodeIDs()[0],
				})
		}
	}
	execClient.On("GetTransactionErrorMessagesByBlockID", mock.Anything, exeEventReq).
		Return(exeErrMessagesResp, nil).
		Once()

	// Mock the txErrorMessages storage to confirm that error messages do not exist yet.
	s.txErrorMessages.On("Exists", blockId).
		Return(false, nil).Once()

	// Mock the storage of the fetched error messages into the protocol database.
	s.txErrorMessages.On("Store", blockId, expectedStoreTxErrorMessages).
		Return(nil).Once()

	err = eng.handleTransactionResultErrorMessages(irrecoverableCtx, blockId)
	require.NoError(s.T(), err)

	// Verify that the mock expectations for storing the error messages were met.
	s.txErrorMessages.AssertExpectations(s.T())

	// Now simulate the second try when the error messages already exist in storage.
	// Mock the txErrorMessages storage to confirm that error messages exist.
	s.txErrorMessages.On("Exists", blockId).
		Return(true, nil).Once()
	err = eng.handleTransactionResultErrorMessages(irrecoverableCtx, blockId)
	require.NoError(s.T(), err)

	// Verify that the mock expectations for storing the error messages were not met.
	s.txErrorMessages.AssertExpectations(s.T())
}

// setupReceipts sets up mock execution receipts for a block and returns the receipts along
// with the identities of the execution nodes that processed them.
func (s *Suite) setupReceipts(block *flow.Block) ([]*flow.ExecutionReceipt, flow.IdentityList) {
	ids := unittest.IdentityListFixture(2, unittest.WithRole(flow.RoleExecution))
	receipt1 := unittest.ReceiptForBlockFixture(block)
	receipt1.ExecutorID = ids[0].NodeID
	receipt2 := unittest.ReceiptForBlockFixture(block)
	receipt2.ExecutorID = ids[1].NodeID
	receipt1.ExecutionResult = receipt2.ExecutionResult

	receipts := flow.ExecutionReceiptList{receipt1, receipt2}
	s.receipts.
		On("ByBlockID", block.ID()).
		Return(receipts, nil)

	return receipts, ids
}

func (s *Suite) TestComponentShutdown() {
	irrecoverableCtx := irrecoverable.NewMockSignalerContext(s.T(), s.ctx)
	eng := s.initIngestionEngine(irrecoverableCtx)

	// start then shut down the engine
	unittest.AssertClosesBefore(s.T(), eng.Ready(), 10*time.Millisecond)
	s.cancel()
	unittest.AssertClosesBefore(s.T(), eng.Done(), 10*time.Millisecond)

	err := eng.Process(channels.ReceiveReceipts, unittest.IdentifierFixture(), &flow.ExecutionReceipt{})
	s.Assert().ErrorIs(err, component.ErrComponentShutdown)
}
