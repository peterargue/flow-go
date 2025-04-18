package internal_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/onflow/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	p2pbuilder "github.com/onflow/flow-go/network/p2p/builder"
	"github.com/onflow/flow-go/network/p2p/node/internal"
	"github.com/onflow/flow-go/utils/unittest"
)

func TestProtocolPeerCache(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p1 := protocol.ID("p1")
	p2 := protocol.ID("p2")
	p3 := protocol.ID("p3")

	// create three hosts, and a pcache for the first
	// the cache supports all 3
	h1, err := p2pbuilder.DefaultLibP2PHost(unittest.DefaultAddress, unittest.KeyFixture(crypto.ECDSASecp256k1))
	require.NoError(t, err)
	pcache, err := internal.NewProtocolPeerCache(zerolog.Nop(), h1, []protocol.ID{p1, p2, p3})
	require.NoError(t, err)
	h2, err := p2pbuilder.DefaultLibP2PHost(unittest.DefaultAddress, unittest.KeyFixture(crypto.ECDSASecp256k1))
	require.NoError(t, err)
	h3, err := p2pbuilder.DefaultLibP2PHost(unittest.DefaultAddress, unittest.KeyFixture(crypto.ECDSASecp256k1))
	require.NoError(t, err)

	// register each host on a separate protocol
	noopHandler := func(s network.Stream) {}
	h1.SetStreamHandler(p1, noopHandler)
	h2.SetStreamHandler(p2, noopHandler)
	h3.SetStreamHandler(p3, noopHandler)

	// connect the hosts to each other
	require.NoError(t, h1.Connect(ctx, *host.InfoFromHost(h2)))
	require.NoError(t, h1.Connect(ctx, *host.InfoFromHost(h3)))
	require.NoError(t, h2.Connect(ctx, *host.InfoFromHost(h3)))

	// check that h1's pcache reflects the protocols supported by h2 and h3
	assert.Eventually(t, func() bool {
		peers2 := pcache.GetPeers(p2)
		peers3 := pcache.GetPeers(p3)
		ok2 := slices.Contains(peers2, h2.ID())
		ok3 := slices.Contains(peers3, h3.ID())
		return len(peers2) == 1 && len(peers3) == 1 && ok2 && ok3
	}, 3*time.Second, 50*time.Millisecond)

	// remove h2's support for p2
	h2.RemoveStreamHandler(p2)

	// check that h1's pcache reflects the change
	assert.Eventually(t, func() bool {
		return len(pcache.GetPeers(p2)) == 0
	}, 3*time.Second, 50*time.Millisecond)

	// add support for p4 on h2 and h3
	// note: pcache does NOT support p4 and should not cache it
	p4 := protocol.ID("p4")
	h2.SetStreamHandler(p4, noopHandler)
	h3.SetStreamHandler(p4, noopHandler)

	// check that h1's pcache never contains p4
	assert.Never(t, func() bool {
		peers4 := pcache.GetPeers(p4)
		ok2 := slices.Contains(peers4, h2.ID())
		ok3 := slices.Contains(peers4, h3.ID())
		return len(peers4) == 2 && ok2 && ok3
	}, 1*time.Second, 50*time.Millisecond)
}
