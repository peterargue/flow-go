package main

import (
	"math/rand"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	"github.com/dapperlabs/flow-go/module/metrics"
	"github.com/dapperlabs/flow-go/module/metrics/example"
	"github.com/dapperlabs/flow-go/module/trace"
	"github.com/dapperlabs/flow-go/utils/unittest"
)

// main runs a local tracer server on the machine and starts monitoring some metrics for sake of execution, which
// increases result approvals counter and checked chunks counter 100 times each
func main() {
	example.WithMetricsServer(func(logger zerolog.Logger) {
		tracer, err := trace.NewTracer(logger, "collection")
		if err != nil {
			panic(err)
		}
		registerer := metrics.NewRegisterer(prometheus.DefaultRegisterer)
		collector := struct {
			*metrics.HotstuffCollector
			*metrics.ExecutionCollector
			*metrics.NetworkCollector
		}{
			HotstuffCollector:  metrics.NewHotstuffCollector("some_chain_id", registerer),
			ExecutionCollector: metrics.NewExecutionCollector(tracer, registerer),
			NetworkCollector:   metrics.NewNetworkCollector(registerer),
		}
		diskTotal := rand.Int63n(1024 ^ 3)
		for i := 0; i < 1000; i++ {
			blockID := unittest.BlockFixture().ID()
			collector.StartBlockReceivedToExecuted(blockID)

			// adds a random delay for execution duration, between 0 and 2 seconds
			time.Sleep(time.Duration(rand.Int31n(2000)) * time.Millisecond)

			collector.ExecutionGasUsedPerBlock(uint64(rand.Int63n(1e6)))
			collector.ExecutionStateReadsPerBlock(uint64(rand.Int63n(1e6)))

			diskIncrease := rand.Int63n(1024 ^ 2)
			diskTotal += diskIncrease
			collector.ExecutionStateStorageDiskTotal(diskTotal)
			collector.ExecutionStorageStateCommitment(diskIncrease)

			collector.FinishBlockReceivedToExecuted(blockID)
		}
	})
}
