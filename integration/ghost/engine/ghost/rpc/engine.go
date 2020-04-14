package rpc

import (
	"fmt"
	"net"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"

	"github.com/dapperlabs/flow-go/engine"
	protobuf "github.com/dapperlabs/flow-go/integration/ghost/protobuf"
	"github.com/dapperlabs/flow-go/model/flow"
	"github.com/dapperlabs/flow-go/module"
	"github.com/dapperlabs/flow-go/network"
)

// Config defines the configurable options for the gRPC server.
type Config struct {
	ListenAddr string
}

// Engine implements a gRPC server for the Ghost Node
type Engine struct {
	unit    *engine.Unit
	log     zerolog.Logger
	handler *Handler     // the gRPC service implementation
	server  *grpc.Server // the gRPC server
	config  Config
	me      module.Local

	// the channel between the engine (producer) and the handler (consumer). The engine receives libp2p messages and
	// writes it to the channel as bytes. The Handler reads from the channel and returns it as GRPC stream to the client
	messages chan []byte
}

// New returns a new RPC engine.
func New(net module.Network, log zerolog.Logger, me module.Local, config Config) (*Engine, error) {

	log = log.With().Str("engine", "rpc").Logger()

	messages := make(chan []byte, 100)

	eng := &Engine{
		log:      log,
		unit:     engine.NewUnit(),
		me:       me,
		server:   grpc.NewServer(),
		config:   config,
		messages: messages,
	}

	conduitMap, err := registerConduits(net, eng)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Engine: %w", err)
	}

	handler := NewHandler(log, conduitMap, messages)
	eng.handler = handler

	protobuf.RegisterGhostNodeAPIServer(eng.server, eng.handler)

	return eng, nil
}

// registerConduits registers for ALL channels and returns a map of engine id to conduit
func registerConduits(net module.Network, eng network.Engine) (map[int]network.Conduit, error) {

	allEngineIDs := []int{engine.CollectionProvider, engine.CollectionIngest, engine.ProtocolClusterConsensus,
		engine.BlockProvider, engine.BlockPropagation, engine.ProtocolConsensus, engine.ProtocolSynchronization,
		engine.ExecutionReceiptProvider, engine.ExecutionStateProvider, engine.ChunkDataPackProvider,
		engine.ApprovalProvider}

	conduitMap := make(map[int]network.Conduit, len(allEngineIDs))

	// Register for ALL channels here and return a map of conduits
	for _, e := range allEngineIDs {
		c, err := net.Register(engine.CollectionProvider, eng)
		if err != nil {
			return nil, fmt.Errorf("could not register collection provider engine: %w", err)
		}
		conduitMap[e] = c
	}

	return conduitMap, nil

}

// Ready returns a ready channel that is closed once the engine has fully
// started. The RPC engine is ready when the gRPC server has successfully
// started.
func (e *Engine) Ready() <-chan struct{} {
	e.unit.Launch(e.serve)
	return e.unit.Ready()
}

// Done returns a done channel that is closed once the engine has fully stopped.
// It sends a signal to stop the gRPC server, then closes the channel.
func (e *Engine) Done() <-chan struct{} {
	return e.unit.Done(e.server.GracefulStop)
}

// SubmitLocal submits an event originating on the local node.
func (e *Engine) SubmitLocal(event interface{}) {
	e.Submit(e.me.NodeID(), event)
}

// Submit submits the given event from the node with the given origin ID
// for processing in a non-blocking manner. It returns instantly and logs
// a potential processing error internally when done.
func (e *Engine) Submit(originID flow.Identifier, event interface{}) {
	e.unit.Launch(func() {
		err := e.process(originID, event)
		if err != nil {
			e.log.Error().Err(err).Msg("could not process submitted event")
		}
	})
}

// ProcessLocal processes an event originating on the local node.
func (e *Engine) ProcessLocal(event interface{}) error {
	return e.Process(e.me.NodeID(), event)
}

// Process processes the given event from the node with the given origin ID in
// a blocking manner. It returns the potential processing error when done.
func (e *Engine) Process(originID flow.Identifier, event interface{}) error {
	return e.unit.Do(func() error {
		return e.process(originID, event)
	})
}

func (e *Engine) process(originID flow.Identifier, event interface{}) error {
	return nil
}

// serve starts the gRPC server .
//
// When this function returns, the server is considered ready.
func (e *Engine) serve() {
	e.log.Info().Msgf("starting server on address %s", e.config.ListenAddr)

	l, err := net.Listen("tcp", e.config.ListenAddr)
	if err != nil {
		e.log.Err(err).Msg("failed to start server")
		return
	}

	err = e.server.Serve(l)
	if err != nil {
		e.log.Err(err).Msg("fatal error in server")
	}
}
