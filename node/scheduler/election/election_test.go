package election

import (
	"github.com/linguohua/titan/api"
	"github.com/linguohua/titan/node/scheduler/node"
)

var (
	deviceOfBandwidth100Mbps = func(deviceID string) api.DevicesInfo {
		return api.DevicesInfo{
			DeviceId:      deviceID,
			BandwidthDown: 100 << 20,
			BandwidthUp:   10 << 20,
		}
	}

	deviceOfBandwidth300Mbps = func(deviceID string) api.DevicesInfo {
		return api.DevicesInfo{
			DeviceId:      deviceID,
			BandwidthDown: 300 << 20,
			BandwidthUp:   30 << 20,
		}
	}

	deviceOfBandwidth500Mbps = func(deviceID string) api.DevicesInfo {
		return api.DevicesInfo{
			DeviceId:      deviceID,
			BandwidthDown: 500 << 20,
			BandwidthUp:   50 << 20,
		}
	}
)

// func TestWinner(t *testing.T) {
// 	assert := assert.New(t)
// 	manager := node.NewNodeManager(nil, nil, nil)

// 	var (
// 		amountOf100MbpsCandidates = 5
// 		amountOf500MbpsCandidates = 0

// 		amountOf100MbpsEdges = 100
// 		amountOf300MbpsEdges = 0
// 	)

// 	{
// 		addCandidateOfBandwidth100Mbps(manager, amountOf100MbpsCandidates)
// 		addEdgeOfBandwidth100Mbps(manager, amountOf100MbpsEdges)
// 	}

// 	selector := NewElection(manager)

// 	winners, err := selector.winner(false)
// 	require.NoError(t, err)
// 	assert.Equal(len(winners), amountOf100MbpsCandidates)

// 	// the candidates have same download bandwidth
// 	manager = node.NewNodeManager(nil, nil, nil)
// 	amountOf100MbpsCandidates = 30
// 	amountOf100MbpsEdges = 30
// 	amountOf300MbpsEdges = 30

// 	{
// 		addCandidateOfBandwidth100Mbps(manager, amountOf100MbpsCandidates)
// 		addEdgeOfBandwidth100Mbps(manager, amountOf100MbpsEdges)
// 		addEdgeOfBandwidth300Mbps(manager, amountOf300MbpsEdges)
// 	}

// 	selector.manage = manager
// 	winners, err = selector.winner(false)
// 	require.NoError(t, err)
// 	assert.Equal(15, len(winners))

// 	addEdgeOfBandwidth100Mbps(manager, amountOf100MbpsEdges)
// 	for _, winner := range winners {
// 		selector.vlk.Lock()
// 		selector.validators[winner.DeviceInfo.DeviceId] = time.Now()
// 		selector.vlk.Unlock()
// 	}
// 	selector.manage = manager
// 	winnersAfterAppend, err := selector.winner(true)
// 	require.NoError(t, err)
// 	assert.Equal(18, len(winnersAfterAppend))
// 	afterAppendContainsWinners := func() (success bool) {
// 		keys := make(map[string]struct{})
// 		for _, winner := range winnersAfterAppend {
// 			keys[winner.DeviceInfo.DeviceId] = struct{}{}
// 		}
// 		for _, v := range winners {
// 			if _, ok := keys[v.DeviceInfo.DeviceId]; !ok {
// 				return false
// 			}
// 		}
// 		return true
// 	}
// 	assert.Condition(afterAppendContainsWinners)

// 	manager = node.NewNodeManager(nil, nil, nil)
// 	// the candidates have difference download bandwidth
// 	amountOf100MbpsCandidates = 13
// 	amountOf500MbpsCandidates = 5
// 	amountOf300MbpsEdges = 100

// 	{
// 		addCandidateOfBandwidth100Mbps(manager, amountOf100MbpsCandidates)
// 		addCandidateOfBandwidth500Mbps(manager, amountOf500MbpsCandidates)
// 		addEdgeOfBandwidth300Mbps(manager, amountOf300MbpsEdges)
// 	}

// 	selector.manage = manager
// 	winners, err = selector.winner(false)
// 	require.NoError(t, err)

// 	var totalDownloadBdw float64
// 	for _, winner := range winners {
// 		totalDownloadBdw += winner.DeviceInfo.BandwidthDown
// 	}

// 	assert.Greater(totalDownloadBdw, float64(amountOf300MbpsEdges*30<<20))

// 	manager = node.NewNodeManager(nil, nil, nil)
// 	amountOf100MbpsCandidates = 1200
// 	amountOf100MbpsEdges = 10000

// 	{
// 		addCandidateOfBandwidth100Mbps(manager, amountOf100MbpsCandidates)
// 		addEdgeOfBandwidth100Mbps(manager, amountOf100MbpsEdges)
// 	}

// 	selector.manage = manager
// 	winners, err = selector.winner(false)
// 	require.NoError(t, err)
// 	assert.Equal(1120, len(winners))
// }

func addCandidateOfBandwidth100Mbps(manage *node.Manager, amount int) {
	for i := 0; i < amount; i++ {
		// deviceID := uuid.NewString()
		// manage.CandidateNodeMap.Store(deviceID, &node.CandidateNode{
		// 	Node: node.Node{
		// 		DeviceInfo: deviceOfBandwidth100Mbps(deviceID),
		// 	},
		// })
	}
}

func addCandidateOfBandwidth500Mbps(manage *node.Manager, amount int) {
	for i := 0; i < amount; i++ {
		// deviceID := uuid.NewString()
		// manage.CandidateNodeMap.Store(deviceID, &node.CandidateNode{
		// 	Node: node.Node{
		// 		DeviceInfo: deviceOfBandwidth500Mbps(deviceID),
		// 	},
		// })
	}
}

func addEdgeOfBandwidth300Mbps(manage *node.Manager, amount int) {
	for i := 0; i < amount; i++ {
		// deviceID := uuid.NewString()
		// manage.EdgeNodeMap.Store(deviceID, &node.EdgeNode{
		// 	Node: node.Node{
		// 		DeviceInfo: deviceOfBandwidth300Mbps(deviceID),
		// 	},
		// })
	}
}

func addEdgeOfBandwidth100Mbps(manage *node.Manager, amount int) {
	for i := 0; i < amount; i++ {
		// deviceID := uuid.NewString()
		// manage.EdgeNodeMap.Store(deviceID, &node.EdgeNode{
		// 	Node: node.Node{
		// 		DeviceInfo: deviceOfBandwidth100Mbps(deviceID),
		// 	},
		// })
	}
}

func addEdgeOfBandwidth500Mbps(manage *node.Manager, amount int) {
	for i := 0; i < amount; i++ {
		// deviceID := uuid.NewString()
		// manage.EdgeNodeMap.Store(deviceID, &node.EdgeNode{
		// 	Node: node.Node{
		// 		DeviceInfo: deviceOfBandwidth500Mbps(deviceID),
		// 	},
		// })
	}
}
