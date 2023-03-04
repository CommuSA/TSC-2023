package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/fentec-project/gofe/abe"
	"github.com/joho/godotenv"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/multiformats/go-multiaddr"

	"github.com/ipfs/go-log/v2"
)

var logger = log.Logger("synchronization")

type AbeIPFSPara struct {
	abefame *abe.FAME
	mpk     *abe.FAMEPubKey
	keys    *abe.FAMEAttribKeys
}

var ipfsAbePara AbeIPFSPara

func getAbePara() AbeIPFSPara {
	var abepara AbeIPFSPara
	err := godotenv.Load(".Member0Env")
	if err != nil {
		panic(err)
	}
	abefameJson := os.Getenv("ABEFAME")
	err = json.Unmarshal([]byte(abefameJson), &abepara.abefame)
	if err != nil {
		panic(err)
	}
	mpkJSON := os.Getenv("MPK")
	err = json.Unmarshal([]byte(mpkJSON), &abepara.mpk)
	if err != nil {
		panic(err)
	}

	keysJSON := os.Getenv("SecKeys")
	err = json.Unmarshal([]byte(keysJSON), &abepara.keys)
	if err != nil {
		panic(err)
	}

	return abepara
}

func handleStream(stream network.Stream) {
	logger.Info("Got a new stream!")

	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	go Receive(rw)
	go Send(rw)

	// 'stream' will stay open until you close it (or the other side closes it).
}

func Receive(rw *bufio.ReadWriter) {
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from buffer")
			panic(err)
		}

		if str == "" {
			return
		}
		if str != "\n" {
			// str = strings.TrimPrefix(str, "&")
			var cipher *abe.FAMECipher
			fileName := "test.txt"
			dstFile, err := os.Create(fileName)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			defer dstFile.Close()
			dstFile.WriteString(str + "\n")
			// fmt.Printf("\x1b[32m%s\x1b[0m> ", str)
			err = json.Unmarshal([]byte(str), &cipher)
			if err != nil {
				panic(err)
			}
			// Green console colour: 	\x1b[32m
			// Reset console colour: 	\x1b[0m
			dec, _ := ipfsAbePara.abefame.Decrypt(cipher, ipfsAbePara.keys, ipfsAbePara.mpk)
			fmt.Printf("\x1b[32m%s\x1b[0m> ", dec)

		}

	}
}

func Send(rw *bufio.ReadWriter) {
	stdReader := bufio.NewReader(os.Stdin)

	// filePointer, err := os.OpenFile("../msg/kddcup_split_0MB/part_6295.data", os.O_RDONLY, 0751)
	// if err != nil {
	// 	panic(err)
	// }
	// defer filePointer.Close()

	// stdReader := bufio.NewReader(filePointer)

	for {
		fmt.Print("> ")
		sendDataGot, err := stdReader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from stdin")
			panic(err)
		}

		// sendData, _ := a.Encrypt(sendDataGot, ipfsAbePara.msp, ipfsAbePara.mpk)
		// sendDataJson, _ := json.Marshal(sendData)
		// _, err = rw.WriteString(fmt.Sprintf("%v\n", sendDataJson))
		_, err = rw.WriteString(fmt.Sprintf("%v\n", sendDataGot))
		if err != nil {
			fmt.Println("Error writing to buffer")
			panic(err)
		}
		err = rw.Flush()
		if err != nil {
			fmt.Println("Error flushing buffer")
			panic(err)
		}
	}
}

func main() {
	// log.SetAllLoggers(log.LevelWarn)
	log.SetLogLevel("synchronization", "info")
	help := flag.Bool("h", false, "Display Help")
	config, err := ParseFlags()
	ipfsAbePara = getAbePara()

	if err != nil {
		panic(err)
	}

	if *help {
		fmt.Println("This program demonstrates a simple p2p chat application using libp2p")
		fmt.Println()
		fmt.Println("Usage: Run './chat in two different terminals. Let them connect to the bootstrap nodes, announce themselves and connect to the peers")
		flag.PrintDefaults()
		return
	}

	// libp2p.New constructs a new libp2p Host. Other options can be added
	// here.
	host, err := libp2p.New(libp2p.ListenAddrs([]multiaddr.Multiaddr(config.ListenAddresses)...))
	if err != nil {
		panic(err)
	}
	logger.Info("Host created. We are:", host.ID())
	logger.Info(host.Addrs())

	// Set a function as stream handler. This function is called when a peer
	// initiates a connection and starts a stream with this peer.
	host.SetStreamHandler(protocol.ID(config.ProtocolID), handleStream)

	// Start a DHT, for use in peer discovery. We can't just make a new DHT
	// client because we want each peer to maintain its own local copy of the
	// DHT, so that the bootstrapping node of the DHT can go down without
	// inhibiting future peer discovery.
	ctx := context.Background()
	kademliaDHT, err := dht.New(ctx, host)
	if err != nil {
		panic(err)
	}

	// Bootstrap the DHT. In the default configuration, this spawns a Background
	// thread that will refresh the peer table every five minutes.
	logger.Debug("Bootstrapping the DHT")
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}

	// Let's connect to the bootstrap nodes first. They will tell us about the
	// other nodes in the network.
	var wg sync.WaitGroup

	for _, peerAddr := range config.BootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := host.Connect(ctx, *peerinfo); err != nil {
				logger.Warning(err)
			} else {
				logger.Info("Connection established with bootstrap node:", *peerinfo)
			}
		}()
	}
	wg.Wait()
	logger.Info(config, ipfsAbePara)

	// We use a synchronization point "meet me here" to announce our location.
	// This is like telling your friends to meet you at the Eiffel Tower.
	logger.Info("Announcing ourselves...")
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, config.RendezvousString)
	logger.Debug("Successfully announced!")

	// Now, look for others who have announced
	// This is like your friend telling you the location to meet you.
	logger.Debug("Searching for other peers...")
	peerChan, err := routingDiscovery.FindPeers(ctx, config.RendezvousString)
	if err != nil {
		panic(err)
	}

	for peer := range peerChan {
		if peer.ID == host.ID() {

			continue
		}
		logger.Debug("Found peer:", peer)

		logger.Debug("Connecting to:", peer)
		stream, err := host.NewStream(ctx, peer.ID, protocol.ID(config.ProtocolID))

		if err != nil {
			// logger.Warning("Connection failed:", err)
			continue
		} else {
			rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

			go Send(rw)
			go Receive(rw)
		}

		logger.Info("Connected to:", peer)
	}

	select {}
}
