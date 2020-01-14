package main

import (
	"github.com/dapperlabs/flow-go/cmd"
	"github.com/dapperlabs/flow-go/engine/execution/execution"
	"github.com/dapperlabs/flow-go/engine/execution/execution/executor"
	"github.com/dapperlabs/flow-go/engine/execution/execution/state"
	"github.com/dapperlabs/flow-go/engine/execution/execution/virtualmachine"
	"github.com/dapperlabs/flow-go/language/runtime"
	"github.com/dapperlabs/flow-go/model/flow"
	"github.com/dapperlabs/flow-go/module"
	"github.com/dapperlabs/flow-go/storage"
	"github.com/dapperlabs/flow-go/storage/badger"
	"github.com/dapperlabs/flow-go/storage/ledger"
	"github.com/dapperlabs/flow-go/storage/ledger/databases/leveldb"
)

func main() {

	var stateCommitments storage.StateCommitments
	var ledgerStorage *ledger.TrieStorage

	cmd.
		FlowNode("execution").
		PostInit(func(node *cmd.FlowNodeBuilder) {
			stateCommitments = badger.NewStateCommitments(node.DB)

			levelDB, err := leveldb.NewLevelDB("db/valuedb", "db/triedb")
			node.MustNot(err).Msg("could not initialize LevelDB databases")

			ledgerStorage, err = ledger.NewTrieStorage(levelDB)
			node.MustNot(err).Msg("could not initialize ledger trie storage")
		}).
		GenesisHandler(func(node *cmd.FlowNodeBuilder, genesis *flow.Block) {
			// TODO We boldly assume that if a genesis is being written than a storage tree is also empty
			initialStateCommitment := flow.StateCommitment(ledgerStorage.LatestStateCommitment())

			err := stateCommitments.Persist(genesis.ID(), &initialStateCommitment)
			node.MustNot(err).Msg("could not store initial state commitment for genesis block")

		}).
		Component("execution engine", func(node *cmd.FlowNodeBuilder) module.ReadyDoneAware {

			node.Logger.Info().Msg("initializing execution engine")

			rt := runtime.NewInterpreterRuntime()
			vm := virtualmachine.New(rt)

			execState := state.NewExecutionState(ledgerStorage, stateCommitments)

			blockExec := executor.NewBlockExecutor(vm, execState)

			collections := badger.NewCollections(node.DB)

			engine, err := execution.New(
				node.Logger,
				node.Network,
				node.Me,
				collections,
				blockExec,
			)
			node.MustNot(err).Msg("could not initialize execution engine")
			return engine
		}).Run()

}
