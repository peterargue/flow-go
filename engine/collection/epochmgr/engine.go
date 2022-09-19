package epochmgr

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/module"
	"github.com/onflow/flow-go/module/component"
	"github.com/onflow/flow-go/module/irrecoverable"
	"github.com/onflow/flow-go/module/mempool/epochs"
	"github.com/onflow/flow-go/module/util"
	"github.com/onflow/flow-go/state/protocol"
	"github.com/onflow/flow-go/state/protocol/events"
)

// DefaultStartupTimeout is the default time we wait when starting epoch components before giving up.
const DefaultStartupTimeout = time.Minute

// ErrNotAuthorizedForEpoch is returned when we attempt to create epoch components
// for an epoch in which we are not an authorized network participant. This is the
// case for epochs during which this node is joining or leaving the network.
var ErrNotAuthorizedForEpoch = fmt.Errorf("we are not an authorized participant for the epoch")

// ErrTimeout is returned when the timeout is exceeded before an epoch's components
// are successfully started or stopped.
var ErrTimeout = fmt.Errorf("timeout exceeded")

// Engine is the epoch manager, which coordinates the lifecycle of other modules
// and processes that are epoch-dependent. The manager is responsible for
// spinning up engines when a new epoch is about to start and spinning down
// engines for an epoch that has ended.
type Engine struct {
	events.Noop // satisfy protocol events consumer interface

	log            zerolog.Logger
	me             module.Local
	state          protocol.State
	pools          *epochs.TransactionPools  // epoch-scoped transaction pools
	factory        EpochComponentsFactory    // consolidates creating epoch for an epoch
	voter          module.ClusterRootQCVoter // manages process of voting for next epoch's QC
	heightEvents   events.Heights            // allows subscribing to particular heights
	startupTimeout time.Duration             // how long we wait for epoch components to start up

	mu     sync.Mutex                           // protects epochs map
	epochs map[uint64]*StartableEpochComponents // epoch-scoped components per epoch

	// internal event notifications
	epochTransitionEvents        chan *flow.Header // sends first block of new epoch
	epochSetupPhaseStartedEvents chan *flow.Header // sends first block of EpochSetup phase
	epochStopEvents              chan uint64       // sends counter of epoch to stop
	epochStartedEvents           chan chan error   // sends signaller context error channel for a new epoch

	cm *component.ComponentManager
	component.Component
}

var _ component.Component = (*Engine)(nil)

func New(
	log zerolog.Logger,
	me module.Local,
	state protocol.State,
	pools *epochs.TransactionPools,
	voter module.ClusterRootQCVoter,
	factory EpochComponentsFactory,
	heightEvents events.Heights,
) (*Engine, error) {

	e := &Engine{
		log:                          log.With().Str("engine", "epochmgr").Logger(),
		me:                           me,
		state:                        state,
		pools:                        pools,
		voter:                        voter,
		factory:                      factory,
		heightEvents:                 heightEvents,
		epochs:                       make(map[uint64]*StartableEpochComponents),
		startupTimeout:               DefaultStartupTimeout,
		epochTransitionEvents:        make(chan *flow.Header, 1),
		epochSetupPhaseStartedEvents: make(chan *flow.Header, 1),
		epochStopEvents:              make(chan uint64, 1),
	}

	e.cm = component.NewComponentManagerBuilder().
		AddWorker(e.handleEpochSetupPhaseStartedLoop).
		AddWorker(e.handleEpochTransitionLoop).
		AddWorker(e.handleStopEpochLoop).
		Build()
	e.Component = e.cm

	return e, nil
}

// Start starts the engine.
func (e *Engine) Start(ctx irrecoverable.SignalerContext) {
	// 1 - start engine-scoped workers
	e.cm.Start(ctx)

	// 2 - check if we should attempt to vote after startup
	err := e.checkShouldVoteOnStartup()
	if err != nil {
		ctx.Throw(err)
	}

	// 3 - start epoch-scoped components
	// set up epoch-scoped epoch managed by this engine for the current epoch
	epoch := e.state.Final().Epochs().Current()
	counter, err := epoch.Counter()
	if err != nil {
		ctx.Throw(err)
	}
	components, err := e.createEpochComponents(epoch)
	if err != nil {
		if errors.Is(err, ErrNotAuthorizedForEpoch) {
			// don't set up consensus components if we aren't authorized in current epoch
			e.log.Info().Msg("node is not authorized for current epoch - skipping initializing cluster consensus")
			return
		}
		ctx.Throw(err)
	}
	err = e.startEpochComponents(ctx, counter, components)
	if err != nil {
		ctx.Throw(err)
	}

	// TODO if we are within the first 600 blocks of an epoch, we should resume the previous epoch's cluster consensus here https://github.com/dapperlabs/flow-go/issues/5659
}

func (e *Engine) checkShouldVoteOnStartup() error {
	// check the current phase on startup, in case we are in setup phase
	// and haven't yet voted for the next root QC
	finalSnapshot := e.state.Final()
	phase, err := finalSnapshot.Phase()
	if err != nil {
		return fmt.Errorf("could not get epoch phase for finalized snapshot: %w", err)
	}
	if phase == flow.EpochPhaseSetup {
		header, err := finalSnapshot.Head()
		if err != nil {
			return fmt.Errorf("could not get header for finalized snapshot: %w", err)
		}
		e.epochSetupPhaseStartedEvents <- header
	}
	return nil
}

// Ready returns a ready channel that is closed once the engine has fully started.
// This is true when the engine-scoped worker threads have started, and all presently
// running epoch components (max 2) have started.
func (e *Engine) Ready() <-chan struct{} {
	e.mu.Lock()
	components := make([]module.ReadyDoneAware, 0, len(e.epochs)+1)
	components = append(components, e.cm)
	for _, epoch := range e.epochs {
		components = append(components, epoch)
	}
	e.mu.Unlock()

	return util.AllReady(components...)
}

// Done returns a done channel that is closed once the engine has fully stopped.
// This is true when the engine-scoped worker threads have stopped, and all presently
// running epoch components (max 2) have stopped.
func (e *Engine) Done() <-chan struct{} {
	e.mu.Lock()
	components := make([]module.ReadyDoneAware, 0, len(e.epochs)+1)
	components = append(components, e.cm)
	for _, epoch := range e.epochs {
		components = append(components, epoch)
	}
	e.mu.Unlock()

	return util.AllDone(components...)
}

// createEpochComponents instantiates and returns epoch-scoped components for
// the given epoch, using the configured factory.
//
// Error returns:
// - ErrNotAuthorizedForEpoch if this node is not authorized in the epoch.
func (e *Engine) createEpochComponents(epoch protocol.Epoch) (*EpochComponents, error) {

	state, prop, sync, hot, voteAggregator, timeoutAggregator, err := e.factory.Create(epoch)
	if err != nil {
		return nil, fmt.Errorf("could not setup requirements for epoch (%d): %w", epoch, err)
	}

	components := NewEpochComponents(state, prop, sync, hot, voteAggregator, timeoutAggregator)
	return components, nil
}

// EpochTransition handles the epoch transition protocol event.
func (e *Engine) EpochTransition(_ uint64, first *flow.Header) {
	e.epochTransitionEvents <- first
}

// EpochSetupPhaseStarted handles the epoch setup phase started protocol event.
func (e *Engine) EpochSetupPhaseStarted(_ uint64, first *flow.Header) {
	e.epochSetupPhaseStartedEvents <- first
}

// handleEpochTransitionLoop handles EpochTransition protocol events.
func (e *Engine) handleEpochTransitionLoop(ctx irrecoverable.SignalerContext, ready component.ReadyFunc) {
	ready()

	for {
		select {
		case <-ctx.Done():
			return
		case firstBlock := <-e.epochTransitionEvents:
			err := e.onEpochTransition(ctx, firstBlock)
			if err != nil {
				ctx.Throw(err)
			}
		}
	}
}

// handleEpochSetupPhaseStartedLoop handles EpochSetupPhaseStarted protocol events.
func (e *Engine) handleEpochSetupPhaseStartedLoop(ctx irrecoverable.SignalerContext, ready component.ReadyFunc) {
	ready()

	for {
		select {
		case <-ctx.Done():
			return
		case firstBlock := <-e.epochSetupPhaseStartedEvents:
			nextEpoch := e.state.AtBlockID(firstBlock.ID()).Epochs().Next()
			e.onEpochSetupPhaseStarted(ctx, nextEpoch)
		}
	}
}

// handleStopEpochLoop handles internal events indicating that a prior epoch's
// components should be stopped.
func (e *Engine) handleStopEpochLoop(ctx irrecoverable.SignalerContext, ready component.ReadyFunc) {
	ready()

	for {
		select {
		case <-ctx.Done():
			return
		case epochCounter := <-e.epochStopEvents:
			err := e.stopEpochComponents(epochCounter)
			if err != nil {
				ctx.Throw(err)
			}
		}
	}
}

func (e *Engine) handleEpochStartedLoop(ctx irrecoverable.SignalerContext, ready component.ReadyFunc) {
	ready()

	for {
		select {
		case <-ctx.Done():
			return
		case errCh := <-e.epochStartedEvents:
			go e.handleEpochErrors(ctx, errCh)
		}
	}
}

func (e *Engine) handleEpochErrors(ctx irrecoverable.SignalerContext, errCh <-chan error) {
	select {
	case <-ctx.Done():
		return
	case err := <-errCh:
		if err != nil {
			ctx.Throw(err)
		}
	}
}

// onEpochTransition is called when we transition to a new epoch. It arranges
// to shut down the last epoch's components and starts up the new epoch's.
//
// No errors are expected during normal operation.
func (e *Engine) onEpochTransition(ctx irrecoverable.SignalerContext, first *flow.Header) error {
	epoch := e.state.AtBlockID(first.ID()).Epochs().Current()
	counter, err := epoch.Counter()
	if err != nil {
		return fmt.Errorf("could not get epoch counter: %w", err)
	}

	// greatest block height in the previous epoch is one less than the first
	// block in current epoch
	lastEpochMaxHeight := first.Height - 1

	log := e.log.With().
		Uint64("last_epoch_max_height", lastEpochMaxHeight).
		Uint64("cur_epoch_counter", counter).
		Logger()

	// exit early and log if the epoch already exists
	_, exists := e.getEpochComponents(counter)
	if exists {
		log.Warn().Msg("epoch transition: components for new epoch already setup, exiting...")
		return nil
	}

	// register a callback to stop the just-ended epoch at the appropriate block height
	e.prepareToStopEpochComponents(counter-1, lastEpochMaxHeight)

	log.Info().Msg("epoch transition: creating components for new epoch...")

	// create components for new epoch
	components, err := e.createEpochComponents(epoch)
	if err != nil {
		if errors.Is(err, ErrNotAuthorizedForEpoch) {
			// if we are not authorized in this epoch, skip starting up cluster consensus
			log.Info().Msg("epoch transition: we are not authorized for new epoch, exiting...")
			return nil
		}
		return fmt.Errorf("could not create epoch components: %w", err)
	}

	// start up components
	err = e.startEpochComponents(ctx, counter, components)
	if err != nil {
		return fmt.Errorf("could not start epoch components: %w", err)
	}

	log.Info().Msg("epoch transition: new epoch components started successfully")

	// set up callback to stop previous epoch
	e.prepareToStopEpochComponents(counter-1, lastEpochMaxHeight)

	return nil
}

// prepareToStopEpochComponents registers a callback to stop the epoch with the
// given counter once it is no longer possible to receive transactions from that
// epoch. This occurs when we finalize sufficiently many blocks in the new epoch
// that a transaction referencing any block from the previous epoch would be
// considered immediately expired.
//
// Transactions referencing blocks from the previous epoch are only valid for
// inclusion in collections built by clusters from that epoch. Consequently, it
// remains possible for the previous epoch's cluster to produce valid collections
// until all such transactions have expired. In fact, since these transactions
// can NOT be included by clusters in the new epoch, we MUST continue producing
// these collections within the previous epoch's clusters.
func (e *Engine) prepareToStopEpochComponents(epochCounter, epochMaxHeight uint64) {

	stopAtHeight := epochMaxHeight + flow.DefaultTransactionExpiry + 1
	e.log.Info().
		Uint64("stopping_epoch_max_height", epochMaxHeight).
		Uint64("stopping_epoch_counter", epochCounter).
		Uint64("stop_at_height", stopAtHeight).
		Str("step", "epoch_transition").
		Msgf("preparing to stop epoch components at height %d", stopAtHeight)

	e.heightEvents.OnHeight(stopAtHeight, func() {
		e.epochStopEvents <- epochCounter
	})
}

// onEpochSetupPhaseStarted is called either when we transition into the epoch
// setup phase, or when the node is restarted during the epoch setup phase. It
// kicks off setup tasks for the phase, in particular submitting a vote for the
// next epoch's root cluster QC.
func (e *Engine) onEpochSetupPhaseStarted(ctx context.Context, nextEpoch protocol.Epoch) {
	// TODO error handling
	err := e.voter.Vote(ctx, nextEpoch)
	if err != nil {
		e.log.Error().Err(err).Msg("failed to submit QC vote for next epoch")
	}
}

// startEpochComponents starts the components for the given epoch and adds them
// to the engine's internal mapping.
//
// Error returns:
// - ErrTimeout if the timeout is exceeded before the epoch components are done.
func (e *Engine) startEpochComponents(engineCtx irrecoverable.SignalerContext, counter uint64, components *EpochComponents) error {

	epochCtx, cancel, errCh := irrecoverable.WithSignallerAndCancel(engineCtx)

	// start component using its own context
	components.Start(epochCtx)
	go e.handleEpochErrors(engineCtx, errCh)

	select {
	case <-components.Ready():
		e.storeEpochComponents(counter, NewStartableEpochComponents(components, cancel))
		return nil
	case <-time.After(e.startupTimeout):
		cancel() // cancel current context if we didn't start in time
		return fmt.Errorf("could not stop epoch %d components after %s: %w", counter, e.startupTimeout, ErrTimeout)
	}
}

// stopEpochComponents stops the components for the given epoch and removes them
// from the engine's internal mapping. If no components exit for the given epoch,
// this is a no-op and a warning is logged.
//
// Error returns:
// - ErrTimeout if the timeout is exceeded before the epoch components are done.
func (e *Engine) stopEpochComponents(counter uint64) error {

	components, exists := e.getEpochComponents(counter)
	if !exists {
		e.log.Warn().Msgf("attempted to stop non-existent epoch %d", counter)
		return nil
	}

	// stop individual component
	components.cancel()

	select {
	case <-components.Done():
		e.removeEpoch(counter)
		e.pools.ForEpoch(counter).Clear()
		return nil
	case <-time.After(e.startupTimeout):
		return fmt.Errorf("could not stop epoch %d components after %s: %w", counter, e.startupTimeout, ErrTimeout)
	}
}

// getEpochComponents retrieves the stored (running) epoch components for the given epoch counter.
// If no epoch with the counter is stored, returns (nil, false).
// Safe for concurrent use.
func (e *Engine) getEpochComponents(counter uint64) (*StartableEpochComponents, bool) {
	e.mu.Lock()
	epoch, ok := e.epochs[counter]
	e.mu.Unlock()
	return epoch, ok
}

// storeEpochComponents stores the given epoch components in the engine's mapping.
// Safe for concurrent use.
func (e *Engine) storeEpochComponents(counter uint64, components *StartableEpochComponents) {
	e.mu.Lock()
	e.epochs[counter] = components
	e.mu.Unlock()
}

// removeEpoch removes the epoch components with the given counter.
// Safe for concurrent use.
func (e *Engine) removeEpoch(counter uint64) {
	e.mu.Lock()
	delete(e.epochs, counter)
	e.mu.Unlock()
}
