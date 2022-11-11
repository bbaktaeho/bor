package bor

import (
	"crypto/ecdsa"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/common/fdlimit"
	"github.com/ethereum/go-ethereum/core"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

func TestMiningAfterLocking(t *testing.T) {

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	fdlimit.Raise(2048)

	// Generate a batch of accounts to seal and fund with
	faucets := make([]*ecdsa.PrivateKey, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i], _ = crypto.GenerateKey()
	}

	// Create an Ethash network based off of the Ropsten config
	genesis := InitGenesis(t, faucets, "./testdata/genesis_2val.json")

	var (
		stacks []*node.Node
		nodes  []*eth.Ethereum
		enodes []*enode.Node
	)
	for i := 0; i < 2; i++ {
		// Start the node and wait until it's up
		stack, ethBackend, err := InitMiner(genesis, keys[i], true)
		if err != nil {
			panic(err)
		}
		defer stack.Close()

		for stack.Server().NodeInfo().Ports.Listener == 0 {
			time.Sleep(250 * time.Millisecond)
		}
		// Connect the node to all the previous ones
		for _, n := range enodes {
			stack.Server().AddPeer(n)
		}
		// Start tracking the node and its enode
		stacks = append(stacks, stack)
		nodes = append(nodes, ethBackend)
		enodes = append(enodes, stack.Server().Self())
	}

	// Iterate over all the nodes and start mining
	time.Sleep(3 * time.Second)
	for _, node := range nodes {
		if err := node.StartMining(1); err != nil {
			panic(err)
		}
	}

	for {

		// for block 0 to 7, the primary validator is node0
		// for block 8 to 15, the primary validator is node1
		// for block 16 to 23, the primary validator is node0
		// for block 24 to 31, the primary validator is node1
		blockHeaderVal0 := nodes[0].BlockChain().CurrentHeader()
		blockHeaderVal1 := nodes[1].BlockChain().CurrentHeader()

		//Lock the sprint at 8th block
		if blockHeaderVal0.Number.Uint64() == 8 {
			block8Hash := blockHeaderVal0.Hash()
			nodes[0].Downloader().ChainValidator.LockMutex(uint64(8))
			nodes[0].Downloader().ChainValidator.UnlockMutex(true, "MilestoneID1", block8Hash)
		}

		//Unlock the locked sprint
		if blockHeaderVal0.Number.Uint64() == 12 {
			nodes[0].Downloader().ChainValidator.UnlockSprint(8)
		}

		if blockHeaderVal1.Number.Uint64() == 16 {
			block16Hash := blockHeaderVal1.Hash()
			nodes[1].Downloader().ChainValidator.LockMutex(uint64(16))
			nodes[1].Downloader().ChainValidator.UnlockMutex(true, "MilestoneID2", block16Hash)
		}

		if blockHeaderVal1.Number.Uint64() == 20 {
			nodes[1].Downloader().ChainValidator.UnlockSprint(16)
		}

		if blockHeaderVal0.Number.Uint64() == 30 {
			break
		}

	}

	blockHeaderVal0 := nodes[0].BlockChain().GetHeaderByNumber(29)
	blockHeaderVal1 := nodes[1].BlockChain().GetHeaderByNumber(29)

	//Both nodes should have same blockheader at 29th block
	assert.Equal(t, blockHeaderVal0, blockHeaderVal1)

	milestoneListVal0 := nodes[0].Downloader().ChainValidator.GetMilestoneIDsList()

	assert.Equal(t, len(milestoneListVal0), int(0))

	milestoneListVal1 := nodes[1].Downloader().ChainValidator.GetMilestoneIDsList()

	assert.Equal(t, len(milestoneListVal1), int(0))

}

func TestReorgingAfterLockingSprint(t *testing.T) {

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	fdlimit.Raise(2048)

	// Generate a batch of accounts to seal and fund with
	faucets := make([]*ecdsa.PrivateKey, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i], _ = crypto.GenerateKey()
	}

	// Create an Ethash network based off of the Ropsten config
	genesis := InitGenesis(t, faucets, "./testdata/genesis_2val.json")

	var (
		stacks []*node.Node
		nodes  []*eth.Ethereum
		enodes []*enode.Node
	)
	for i := 0; i < 2; i++ {
		// Start the node and wait until it's up
		stack, ethBackend, err := InitMiner(genesis, keys[i], true)
		if err != nil {
			panic(err)
		}
		defer stack.Close()

		for stack.Server().NodeInfo().Ports.Listener == 0 {
			time.Sleep(250 * time.Millisecond)
		}
		// Connect the node to all the previous ones
		for _, n := range enodes {
			stack.Server().AddPeer(n)
		}
		// Start tracking the node and its enode
		stacks = append(stacks, stack)
		nodes = append(nodes, ethBackend)
		enodes = append(enodes, stack.Server().Self())
	}

	// Iterate over all the nodes and start mining
	time.Sleep(3 * time.Second)
	for _, node := range nodes {
		if err := node.StartMining(1); err != nil {
			panic(err)
		}
	}

	for {

		// for block 0 to 7, the primary validator is node0
		// for block 8 to 15, the primary validator is node1
		// for block 16 to 23, the primary validator is node0
		// for block 24 to 31, the primary validator is node1
		blockHeaderVal0 := nodes[0].BlockChain().CurrentHeader()

		//Disconnect the peers at block 9
		if blockHeaderVal0.Number.Uint64() == 9 {

			stacks[0].Server().RemovePeer(enodes[1])
			stacks[1].Server().RemovePeer(enodes[0])
		}

		//Node0 will be sealing out of turn and lock it till 12th block
		if blockHeaderVal0.Number.Uint64() == 12 {
			block12Hash := blockHeaderVal0.Hash()
			nodes[0].Downloader().ChainValidator.LockMutex(uint64(12))
			nodes[0].Downloader().ChainValidator.UnlockMutex(true, "MilestoneID1", block12Hash)
		}

		//Connect both the nodes
		if blockHeaderVal0.Number.Uint64() == 14 {

			stacks[0].Server().AddPeer(enodes[1])
			stacks[1].Server().AddPeer(enodes[0])
		}

		authorVal0, err := nodes[0].Engine().Author(blockHeaderVal0)

		//This will be true only when Node 0 has received the block from node 1 after 12th block.
		if err == nil && blockHeaderVal0.Number.Uint64() > 12 && authorVal0 == nodes[1].AccountManager().Accounts()[0] {
			break
		}

		if blockHeaderVal0.Number.Uint64() == 30 {
			break
		}

	}

	// check block 10 block ; expected author is node1 signer
	blockHeader10Val0 := nodes[0].BlockChain().GetHeaderByNumber(10)

	author10Val0, err := nodes[0].Engine().Author(blockHeader10Val0)

	if err == nil {
		assert.Equal(t, author10Val0, nodes[0].AccountManager().Accounts()[0])
	}

	blockHeader12Val0 := nodes[0].BlockChain().GetHeaderByNumber(12)

	author12Val0, err := nodes[0].Engine().Author(blockHeader12Val0)

	if err == nil {
		assert.Equal(t, author12Val0, nodes[0].AccountManager().Accounts()[0])
	}

	//milestoneIDList should contain only one milestoneID
	milestoneListVal1 := nodes[0].Downloader().ChainValidator.GetMilestoneIDsList()

	assert.Equal(t, len(milestoneListVal1), int(1))

}

func TestReorgingAfterWhitelisting(t *testing.T) {

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	fdlimit.Raise(2048)

	// Generate a batch of accounts to seal and fund with
	faucets := make([]*ecdsa.PrivateKey, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i], _ = crypto.GenerateKey()
	}

	// Create an Ethash network based off of the Ropsten config
	genesis := InitGenesis(t, faucets, "./testdata/genesis_2val.json")

	var (
		stacks []*node.Node
		nodes  []*eth.Ethereum
		enodes []*enode.Node
	)
	for i := 0; i < 2; i++ {
		// Start the node and wait until it's up
		stack, ethBackend, err := InitMiner(genesis, keys[i], true)
		if err != nil {
			panic(err)
		}
		defer stack.Close()

		for stack.Server().NodeInfo().Ports.Listener == 0 {
			time.Sleep(250 * time.Millisecond)
		}
		// Connect the node to all the previous ones
		for _, n := range enodes {
			stack.Server().AddPeer(n)
		}
		// Start tracking the node and its enode
		stacks = append(stacks, stack)
		nodes = append(nodes, ethBackend)
		enodes = append(enodes, stack.Server().Self())
	}

	// Iterate over all the nodes and start mining
	time.Sleep(3 * time.Second)
	for _, node := range nodes {
		if err := node.StartMining(1); err != nil {
			panic(err)
		}
	}

	for {

		// for block 0 to 7, the primary validator is node0
		// for block 8 to 15, the primary validator is node1
		// for block 16 to 23, the primary validator is node0
		// for block 24 to 31, the primary validator is node1
		blockHeaderVal0 := nodes[0].BlockChain().CurrentHeader()

		//Disconnect the peers at block 9
		if blockHeaderVal0.Number.Uint64() == 9 {

			stacks[0].Server().RemovePeer(enodes[1])
			stacks[1].Server().RemovePeer(enodes[0])
		}

		//Node0 will be sealing out of turn and lock it till 12th block
		if blockHeaderVal0.Number.Uint64() == 12 {
			block12Hash := blockHeaderVal0.Hash()
			nodes[0].Downloader().ChainValidator.LockMutex(uint64(12))
			nodes[0].Downloader().ChainValidator.UnlockMutex(true, "MilestoneID1", block12Hash)
		}

		if blockHeaderVal0.Number.Uint64() == 13 {
			block13Hash := blockHeaderVal0.Hash()
			nodes[0].Downloader().ChainValidator.ProcessMilestone(13, block13Hash)

		}

		if blockHeaderVal0.Number.Uint64() == 14 {

			stacks[0].Server().AddPeer(enodes[1])
			stacks[1].Server().AddPeer(enodes[0])
		}

		authorVal0, err := nodes[0].Engine().Author(blockHeaderVal0)

		//This condition is true when Node 0 has received the block from node 1 after block 12
		if err == nil && blockHeaderVal0.Number.Uint64() > 12 && authorVal0 == nodes[1].AccountManager().Accounts()[0] {
			break
		}

		if blockHeaderVal0.Number.Uint64() == 30 {
			break
		}

	}

	arr := []uint64{10, 12, 13}

	for _, val := range arr {
		blockHeaderVal := nodes[0].BlockChain().GetHeaderByNumber(val)

		authorVal, err := nodes[0].Engine().Author(blockHeaderVal)

		if err == nil {
			assert.Equal(t, authorVal, nodes[0].AccountManager().Accounts()[0])
		}

	}

}

func TestPeerConnectionAfterWhitelisting(t *testing.T) {

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	fdlimit.Raise(2048)

	// Generate a batch of accounts to seal and fund with
	faucets := make([]*ecdsa.PrivateKey, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i], _ = crypto.GenerateKey()
	}

	// Create an Ethash network based off of the Ropsten config
	genesis := InitGenesis(t, faucets, "./testdata/genesis_2val.json")

	var (
		stacks []*node.Node
		nodes  []*eth.Ethereum
		enodes []*enode.Node
	)
	for i := 0; i < 2; i++ {
		// Start the node and wait until it's up
		stack, ethBackend, err := InitMiner(genesis, keys[i], true)
		if err != nil {
			panic(err)
		}
		defer stack.Close()

		for stack.Server().NodeInfo().Ports.Listener == 0 {
			time.Sleep(250 * time.Millisecond)
		}
		// Connect the node to all the previous ones
		for _, n := range enodes {
			stack.Server().AddPeer(n)
		}
		// Start tracking the node and its enode
		stacks = append(stacks, stack)
		nodes = append(nodes, ethBackend)
		enodes = append(enodes, stack.Server().Self())
	}

	// Iterate over all the nodes and start mining
	time.Sleep(3 * time.Second)
	for _, node := range nodes {
		if err := node.StartMining(1); err != nil {
			panic(err)
		}
	}

	disconnectFlag := false

	for {

		// for block 0 to 7, the primary validator is node0
		// for block 8 to 15, the primary validator is node1
		// for block 16 to 23, the primary validator is node0
		// for block 24 to 31, the primary validator is node1
		blockHeaderVal0 := nodes[0].BlockChain().CurrentHeader()
		blockHeaderVal1 := nodes[1].BlockChain().CurrentHeader()

		//Disconnect the peers at block 8
		if blockHeaderVal0.Number.Uint64() == 8 && blockHeaderVal1.Number.Uint64() < 12 {
			disconnectFlag = true
			stacks[0].Server().RemovePeer(enodes[1])
			stacks[1].Server().RemovePeer(enodes[0])
		}

		//Whitelist the validator0 with milestone at 12
		if blockHeaderVal0.Number.Uint64() == 12 {
			block12Hash := blockHeaderVal0.Hash()
			nodes[0].Downloader().ChainValidator.ProcessMilestone(uint64(12), block12Hash)
		}

		///Whitelist the validator1 with milestone at 12
		if blockHeaderVal1.Number.Uint64() == 12 {
			block12Hash := blockHeaderVal1.Hash()
			nodes[1].Downloader().ChainValidator.ProcessMilestone(uint64(12), block12Hash)
		}

		if blockHeaderVal0.Number.Uint64() > 12 && blockHeaderVal0.Number.Uint64() > 12 {
			stacks[0].Server().AddPeer(enodes[1])
			stacks[1].Server().AddPeer(enodes[0])
		}

		if blockHeaderVal0.Number.Uint64() == 30 {
			break
		}

	}

	// validator one peer count
	val0PeerCount := stacks[0].Server().PeerCount()
	val1PeerCount := stacks[1].Server().PeerCount()

	if disconnectFlag {
		assert.Equal(t, val0PeerCount, 0)
		assert.Equal(t, val1PeerCount, 0)

		blockHeader13Val0 := nodes[0].BlockChain().GetHeaderByNumber(13)

		author13Val0, err := nodes[0].Engine().Author(blockHeader13Val0)

		if err == nil {
			assert.Equal(t, author13Val0, nodes[0].AccountManager().Accounts()[0])
		}

		blockHeader13Val1 := nodes[1].BlockChain().GetHeaderByNumber(13)

		author13Val1, err := nodes[1].Engine().Author(blockHeader13Val1)

		if err == nil {
			assert.Equal(t, author13Val1, nodes[1].AccountManager().Accounts()[0])
		}
	}

}

func TestReorgingFutureSprintAfterLocking(t *testing.T) {

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	fdlimit.Raise(2048)

	// Generate a batch of accounts to seal and fund with
	faucets := make([]*ecdsa.PrivateKey, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i], _ = crypto.GenerateKey()
	}

	// Create an Ethash network based off of the Ropsten config
	genesis := InitGenesis(t, faucets, "./testdata/genesis_2val.json")

	var (
		stacks []*node.Node
		nodes  []*eth.Ethereum
		enodes []*enode.Node
	)
	for i := 0; i < 2; i++ {
		// Start the node and wait until it's up
		stack, ethBackend, err := InitMiner(genesis, keys[i], true)
		if err != nil {
			panic(err)
		}
		defer stack.Close()

		for stack.Server().NodeInfo().Ports.Listener == 0 {
			time.Sleep(250 * time.Millisecond)
		}
		// Connect the node to all the previous ones
		for _, n := range enodes {
			stack.Server().AddPeer(n)
		}
		// Start tracking the node and its enode
		stacks = append(stacks, stack)
		nodes = append(nodes, ethBackend)
		enodes = append(enodes, stack.Server().Self())
	}

	// Iterate over all the nodes and start mining
	time.Sleep(3 * time.Second)
	for _, node := range nodes {
		if err := node.StartMining(1); err != nil {
			panic(err)
		}
	}

	for {

		// for block 0 to 7, the primary validator is node0
		// for block 8 to 15, the primary validator is node1
		// for block 16 to 23, the primary validator is node0
		// for block 24 to 31, the primary validator is node1
		blockHeaderVal0 := nodes[0].BlockChain().CurrentHeader()

		//Locking sprint at height 8
		if blockHeaderVal0.Number.Uint64() == 8 {
			block8Hash := blockHeaderVal0.Hash()
			nodes[0].Downloader().ChainValidator.LockMutex(uint64(8))
			nodes[0].Downloader().ChainValidator.UnlockMutex(true, "milestoneID1", block8Hash)
		}

		if blockHeaderVal0.Number.Uint64() == 30 {
			break
		}

	}

	node1Arr := []uint64{8, 15, 24}

	for _, val := range node1Arr {
		blockHeader := nodes[0].BlockChain().GetHeaderByNumber(val)

		authorVal, err := nodes[0].Engine().Author(blockHeader)

		if err == nil {
			assert.Equal(t, authorVal, nodes[1].AccountManager().Accounts()[0])
		}

	}

}

func TestReorgingFutureSprintAfterLockingOnSameHash(t *testing.T) {

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	fdlimit.Raise(2048)

	// Generate a batch of accounts to seal and fund with
	faucets := make([]*ecdsa.PrivateKey, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i], _ = crypto.GenerateKey()
	}

	// Create an Ethash network based off of the Ropsten config
	genesis := InitGenesis(t, faucets, "./testdata/genesis_2val.json")

	var (
		stacks []*node.Node
		nodes  []*eth.Ethereum
		enodes []*enode.Node
	)
	for i := 0; i < 2; i++ {
		// Start the node and wait until it's up
		stack, ethBackend, err := InitMiner(genesis, keys[i], true)
		if err != nil {
			panic(err)
		}
		defer stack.Close()

		for stack.Server().NodeInfo().Ports.Listener == 0 {
			time.Sleep(250 * time.Millisecond)
		}
		// Connect the node to all the previous ones
		for _, n := range enodes {
			stack.Server().AddPeer(n)
		}
		// Start tracking the node and its enode
		stacks = append(stacks, stack)
		nodes = append(nodes, ethBackend)
		enodes = append(enodes, stack.Server().Self())
	}

	// Iterate over all the nodes and start mining
	time.Sleep(3 * time.Second)
	for _, node := range nodes {
		if err := node.StartMining(1); err != nil {
			panic(err)
		}
	}

	for {

		// for block 0 to 7, the primary validator is node0
		// for block 8 to 15, the primary validator is node1
		// for block 16 to 23, the primary validator is node0
		// for block 24 to 31, the primary validator is node1
		blockHeaderVal0 := nodes[0].BlockChain().CurrentHeader()

		//Locking sprint at height 8
		if blockHeaderVal0.Number.Uint64() == 8 {
			block8Hash := blockHeaderVal0.Hash()
			nodes[0].Downloader().ChainValidator.LockMutex(uint64(8))
			nodes[0].Downloader().ChainValidator.UnlockMutex(true, "milestoneID1", block8Hash)
		}

		if blockHeaderVal0.Number.Uint64() == 30 {
			break
		}

	}

	node1Arr := []uint64{8, 15, 24}

	for _, val := range node1Arr {
		blockHeader := nodes[0].BlockChain().GetHeaderByNumber(val)

		authorVal, err := nodes[0].Engine().Author(blockHeader)

		if err == nil {
			assert.Equal(t, authorVal, nodes[1].AccountManager().Accounts()[0])
		}

	}

}

func TestReorgingAfterLockingOnDifferentHash(t *testing.T) {

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	fdlimit.Raise(2048)

	// Generate a batch of accounts to seal and fund with
	faucets := make([]*ecdsa.PrivateKey, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i], _ = crypto.GenerateKey()
	}

	// Create an Ethash network based off of the Ropsten config
	genesis := InitGenesis(t, faucets, "./testdata/genesis_2val.json")

	var (
		stacks []*node.Node
		nodes  []*eth.Ethereum
		enodes []*enode.Node
	)
	for i := 0; i < 2; i++ {
		// Start the node and wait until it's up
		stack, ethBackend, err := InitMiner(genesis, keys[i], true)
		if err != nil {
			panic(err)
		}
		defer stack.Close()

		for stack.Server().NodeInfo().Ports.Listener == 0 {
			time.Sleep(250 * time.Millisecond)
		}

		// Connect the node to all the previous ones
		for _, n := range enodes {
			stack.Server().AddPeer(n)
		}

		// Start tracking the node and its enode
		stacks = append(stacks, stack)
		nodes = append(nodes, ethBackend)
		enodes = append(enodes, stack.Server().Self())
	}

	// Iterate over all the nodes and start mining
	time.Sleep(3 * time.Second)
	for _, node := range nodes {
		if err := node.StartMining(1); err != nil {
			panic(err)
		}
	}

	//Peers are disconnected as we have not connected them

	stacks[0].Server().RemovePeer(enodes[1])
	stacks[1].Server().RemovePeer(enodes[0])

	chain2HeadChNode0 := make(chan core.Chain2HeadEvent, 64)
	chain2HeadChNode1 := make(chan core.Chain2HeadEvent, 64)

	nodes[0].BlockChain().SubscribeChain2HeadEvent(chain2HeadChNode0)
	nodes[1].BlockChain().SubscribeChain2HeadEvent(chain2HeadChNode1)

	for {

		// for block 0 to 7, the primary validator is node0
		// for block 8 to 15, the primary validator is node1
		// for block 16 to 23, the primary validator is node0
		// for block 24 to 31, the primary validator is node1
		blockHeaderVal0 := nodes[0].BlockChain().CurrentHeader()
		blockHeaderVal1 := nodes[1].BlockChain().CurrentHeader()

		//Locking sprint at height 7
		if blockHeaderVal0.Number.Uint64() == 7 {
			block7Hash := blockHeaderVal0.Hash()
			nodes[0].Downloader().ChainValidator.LockMutex(uint64(7))
			nodes[0].Downloader().ChainValidator.UnlockMutex(true, "milestoneID1", block7Hash)
		}

		if blockHeaderVal1.Number.Uint64() == 7 {
			block7Hash := blockHeaderVal1.Hash()
			nodes[1].Downloader().ChainValidator.LockMutex(uint64(7))
			nodes[1].Downloader().ChainValidator.UnlockMutex(true, "milestoneID1", block7Hash)
		}

		if blockHeaderVal0.Number.Uint64() > 15 && blockHeaderVal1.Number.Uint64() > 15 {
			stacks[0].Server().AddPeer(enodes[1])
			stacks[1].Server().AddPeer(enodes[0])
		}

		select {
		case ev := <-chain2HeadChNode0:
			if ev.Type == core.Chain2HeadReorgEvent {
				assert.Fail(t, "Node 0 should not get reorged")
				break

			}

		case ev := <-chain2HeadChNode1:

			if ev.Type == core.Chain2HeadReorgEvent {
				assert.Fail(t, "Node 1 should not get reorged")
				break
			}

		default:
			time.Sleep(1 * time.Millisecond)

		}

		if blockHeaderVal0.Number.Uint64() == 30 {
			break
		}

	}

}

func TestReorgingAfterWhitelistingOnDifferentHash(t *testing.T) {

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	fdlimit.Raise(2048)

	// Generate a batch of accounts to seal and fund with
	faucets := make([]*ecdsa.PrivateKey, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i], _ = crypto.GenerateKey()
	}

	// Create an Ethash network based off of the Ropsten config
	genesis := InitGenesis(t, faucets, "./testdata/genesis_2val.json")

	var (
		stacks []*node.Node
		nodes  []*eth.Ethereum
		enodes []*enode.Node
	)
	for i := 0; i < 2; i++ {
		// Start the node and wait until it's up
		stack, ethBackend, err := InitMiner(genesis, keys[i], true)
		if err != nil {
			panic(err)
		}
		defer stack.Close()

		for stack.Server().NodeInfo().Ports.Listener == 0 {
			time.Sleep(250 * time.Millisecond)
		}

		// Connect the node to all the previous ones
		for _, n := range enodes {
			stack.Server().AddPeer(n)
		}

		// Start tracking the node and its enode
		stacks = append(stacks, stack)
		nodes = append(nodes, ethBackend)
		enodes = append(enodes, stack.Server().Self())
	}

	// Iterate over all the nodes and start mining
	time.Sleep(3 * time.Second)
	for _, node := range nodes {
		if err := node.StartMining(1); err != nil {
			panic(err)
		}
	}

	//Peers are disconnected as we have not connected them

	stacks[0].Server().RemovePeer(enodes[1])
	stacks[1].Server().RemovePeer(enodes[0])

	chain2HeadChNode0 := make(chan core.Chain2HeadEvent, 64)
	chain2HeadChNode1 := make(chan core.Chain2HeadEvent, 64)

	nodes[0].BlockChain().SubscribeChain2HeadEvent(chain2HeadChNode0)
	nodes[1].BlockChain().SubscribeChain2HeadEvent(chain2HeadChNode1)

	for {

		// for block 0 to 7, the primary validator is node0
		// for block 8 to 15, the primary validator is node1
		// for block 16 to 23, the primary validator is node0
		// for block 24 to 31, the primary validator is node1
		blockHeaderVal0 := nodes[0].BlockChain().CurrentHeader()
		blockHeaderVal1 := nodes[1].BlockChain().CurrentHeader()

		if blockHeaderVal0.Number.Uint64() == 20 {
			break
		}

		//whitelisting at height
		if blockHeaderVal0.Number.Uint64() == 1 {
			block1Hash := blockHeaderVal0.Hash()
			nodes[0].Downloader().ChainValidator.ProcessMilestone(uint64(1), block1Hash)
		}

		if blockHeaderVal1.Number.Uint64() == 1 {
			block1Hash := blockHeaderVal1.Hash()
			nodes[1].Downloader().ChainValidator.ProcessMilestone(uint64(1), block1Hash)
		}

		if blockHeaderVal0.Number.Uint64() > 1 && blockHeaderVal1.Number.Uint64() > 1 {
			stacks[0].Server().AddPeer(enodes[1])
			stacks[1].Server().AddPeer(enodes[0])
		}

		select {
		case ev := <-chain2HeadChNode0:
			if ev.Type == core.Chain2HeadReorgEvent {
				assert.Fail(t, "Node 0 should not get reorged as it was whitelisted on different hash")
				break

			}

		case ev := <-chain2HeadChNode1:

			if ev.Type == core.Chain2HeadReorgEvent {
				assert.Fail(t, "Node 1 should not get reorged as it was whiteliseted on different hash")
				break
			}

		default:
			time.Sleep(1 * time.Millisecond)

		}

		if blockHeaderVal0.Number.Uint64() == 30 {
			break
		}

	}

}

func TestNonMinerNodeWithWhitelisting(t *testing.T) {

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	fdlimit.Raise(2048)

	// Generate a batch of accounts to seal and fund with
	faucets := make([]*ecdsa.PrivateKey, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i], _ = crypto.GenerateKey()
	}

	// Create an Ethash network based off of the Ropsten config
	genesis := InitGenesis(t, faucets, "./testdata/genesis_2val.json")

	var (
		stacks []*node.Node
		nodes  []*eth.Ethereum
		enodes []*enode.Node
	)
	for i := 0; i < 2; i++ {
		// Start the node and wait until it's up
		stack, ethBackend, err := InitMiner(genesis, keys[i], true)
		if err != nil {
			panic(err)
		}
		defer stack.Close()

		for stack.Server().NodeInfo().Ports.Listener == 0 {
			time.Sleep(250 * time.Millisecond)
		}

		// Connect the node to all the previous ones
		for _, n := range enodes {
			stack.Server().AddPeer(n)
		}

		// Start tracking the node and its enode
		stacks = append(stacks, stack)
		nodes = append(nodes, ethBackend)
		enodes = append(enodes, stack.Server().Self())
	}

	// Iterate over all the nodes and start mining
	time.Sleep(3 * time.Second)

	//Only started the node 0 and keep the node 1 as non mining
	err := nodes[0].StartMining(1)
	if err != nil {
		panic(err)
	}

	for {

		blockHeaderVal0 := nodes[0].BlockChain().CurrentHeader()
		blockHeaderVal1 := nodes[1].BlockChain().CurrentHeader()

		//Whitelisting milestone
		if blockHeaderVal1.Number.Uint64() == 7 {
			blockHash := blockHeaderVal1.Hash()
			nodes[0].Downloader().ChainValidator.ProcessMilestone(blockHeaderVal1.Number.Uint64(), blockHash)
		}

		//Whitelisting milestone
		if blockHeaderVal1.Number.Uint64() == 15 {
			blockHash := blockHeaderVal1.Hash()
			nodes[0].Downloader().ChainValidator.ProcessMilestone(blockHeaderVal1.Number.Uint64(), blockHash)
		}

		//Whitelisting milestone
		if blockHeaderVal1.Number.Uint64() == 23 {
			blockHash := blockHeaderVal1.Hash()
			nodes[0].Downloader().ChainValidator.ProcessMilestone(blockHeaderVal1.Number.Uint64(), blockHash)
		}

		if blockHeaderVal0.Number.Uint64() == 30 {
			break
		}

	}

	for i := uint64(0); i < nodes[1].BlockChain().CurrentBlock().NumberU64(); i++ {

		blockHeader := nodes[1].BlockChain().GetHeaderByNumber(i)

		authorVal, err := nodes[1].Engine().Author(blockHeader)

		if err == nil {
			assert.Equal(t, authorVal, nodes[0].AccountManager().Accounts()[0])
		}

	}

}

func TestNonMinerNodeWithTryToLock(t *testing.T) {

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	fdlimit.Raise(2048)

	// Generate a batch of accounts to seal and fund with
	faucets := make([]*ecdsa.PrivateKey, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i], _ = crypto.GenerateKey()
	}

	// Create an Ethash network based off of the Ropsten config
	genesis := InitGenesis(t, faucets, "./testdata/genesis_2val.json")

	var (
		stacks []*node.Node
		nodes  []*eth.Ethereum
		enodes []*enode.Node
	)
	for i := 0; i < 2; i++ {
		// Start the node and wait until it's up
		stack, ethBackend, err := InitMiner(genesis, keys[i], true)
		if err != nil {
			panic(err)
		}
		defer stack.Close()

		for stack.Server().NodeInfo().Ports.Listener == 0 {
			time.Sleep(250 * time.Millisecond)
		}

		// Connect the node to all the previous ones
		for _, n := range enodes {
			stack.Server().AddPeer(n)
		}

		// Start tracking the node and its enode
		stacks = append(stacks, stack)
		nodes = append(nodes, ethBackend)
		enodes = append(enodes, stack.Server().Self())
	}

	// Iterate over all the nodes and start mining
	time.Sleep(3 * time.Second)

	//Only started the node 0 and keep the node 1 as non mining
	err := nodes[0].StartMining(1)
	if err != nil {
		panic(err)
	}

	for {

		blockHeaderVal0 := nodes[0].BlockChain().CurrentHeader()
		blockHeaderVal1 := nodes[1].BlockChain().CurrentHeader()

		//Asking for the vote
		if blockHeaderVal1.Number.Uint64() == 7 {
			blockHash := blockHeaderVal1.Hash()
			nodes[1].APIBackend.GetVoteOnRootHash(nil, 0, 7, "0x"+blockHash.String(), "MilestoneID1")
		}

		//Asking for the vote
		if blockHeaderVal1.Number.Uint64() == 15 {
			blockHash := blockHeaderVal1.Hash()
			nodes[1].APIBackend.GetVoteOnRootHash(nil, 0, 7, "0x"+blockHash.String(), "MilestoneID2")
		}

		//Asking for the vote
		if blockHeaderVal1.Number.Uint64() == 23 {
			blockHash := blockHeaderVal1.Hash()
			nodes[1].APIBackend.GetVoteOnRootHash(nil, 0, 7, "0x"+blockHash.String(), "MilestoneID3")
		}

		milestoneList := nodes[0].Downloader().ChainValidator.GetMilestoneIDsList()
		if len(milestoneList) > 0 {
			assert.Fail(t, "MilestoneList should be of zero length")
		}

		if blockHeaderVal0.Number.Uint64() == 30 {
			break
		}

	}

}

func TestRewind(t *testing.T) {

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	fdlimit.Raise(2048)

	// Generate a batch of accounts to seal and fund with
	faucets := make([]*ecdsa.PrivateKey, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i], _ = crypto.GenerateKey()
	}

	// Create an Ethash network based off of the Ropsten config
	genesis := InitGenesis(t, faucets, "./testdata/genesis_2val.json")

	var (
		stacks []*node.Node
		nodes  []*eth.Ethereum
		enodes []*enode.Node
	)
	for i := 0; i < 2; i++ {
		// Start the node and wait until it's up
		stack, ethBackend, err := InitMiner(genesis, keys[i], true)
		if err != nil {
			panic(err)
		}
		defer stack.Close()

		for stack.Server().NodeInfo().Ports.Listener == 0 {
			time.Sleep(250 * time.Millisecond)
		}

		// Connect the node to all the previous ones
		for _, n := range enodes {
			stack.Server().AddPeer(n)
		}

		// Start tracking the node and its enode
		stacks = append(stacks, stack)
		nodes = append(nodes, ethBackend)
		enodes = append(enodes, stack.Server().Self())
	}

	// Iterate over all the nodes and start mining
	time.Sleep(3 * time.Second)

	for _, node := range nodes {
		if err := node.StartMining(1); err != nil {
			panic(err)
		}
	}

	// step1 := false
	// step2 := false
	// step3 := false
	step4 := false

	for {

		blockHeaderVal0 := nodes[0].BlockChain().CurrentHeader()
		blockHeaderVal1 := nodes[1].BlockChain().CurrentHeader()

		// if blockHeaderVal1.Number.Uint64() == 20 && !step1 {
		// 	nodes[1].BlockChain().SetHead(2)

		// 	step1 = true
		// }

		// if blockHeaderVal1.Number.Uint64() == 40 && !step2 {
		// 	nodes[1].BlockChain().SetHead(2)

		// 	step2 = true
		// }

		// if blockHeaderVal1.Number.Uint64() == 80 && !step3 {
		// 	nodes[1].BlockChain().SetHead(2)

		// 	step3 = true
		// }

		// if blockHeaderVal1.Number.Uint64() == 120 && !step4 {
		// 	nodes[1].BlockChain().SetHead(2)

		// 	step4 = true
		// }

		if blockHeaderVal1.Number.Uint64() == 180 && !step4 {

			ch := make(chan struct{})
			nodes[1].Miner().Stop(ch)
			<-ch
			nodes[1].BlockChain().SetHead(2)
			nodes[1].StartMining(1)
			step4 = true
		}

		if blockHeaderVal0.Number.Uint64() == 200 {
			break
		}

	}

}
