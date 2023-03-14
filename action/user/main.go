package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/fentec-project/gofe/abe"
	"github.com/joho/godotenv"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
)

var config Config

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

func main() {
	config, _ = ParseFlags()
	ctx := context.Background()
	ipfsAbePara = getAbePara()

	h, err := libp2p.New(libp2p.ListenAddrs([]multiaddr.Multiaddr(config.ListenAddresses)...))
	if err != nil {
		panic(err)
	}
	go discoverPeers(ctx, h)

	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		panic(err)
	}
	topic, err := ps.Join(config.TopicString)
	if err != nil {
		panic(err)
	}
	go streamConsoleTo(ctx, topic)

	sub, err := topic.Subscribe()
	if err != nil {
		panic(err)
	}
	MessagesReceive(ctx, sub)
}

func initDHT(ctx context.Context, h host.Host) *dht.IpfsDHT {
	// Start a DHT, for use in peer discovery. We can't just make a new DHT
	// client because we want each peer to maintain its own local copy of the
	// DHT, so that the bootstrapping node of the DHT can go down without
	// inhibiting future peer discovery.
	kademliaDHT, err := dht.New(ctx, h)
	if err != nil {
		panic(err)
	}
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}
	var wg sync.WaitGroup
	for _, peerAddr := range config.BootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.Connect(ctx, *peerinfo); err != nil {
				fmt.Println("Bootstrap warning:", err)
			}
		}()
	}
	wg.Wait()

	return kademliaDHT
}

func discoverPeers(ctx context.Context, h host.Host) {
	kademliaDHT := initDHT(ctx, h)
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, config.TopicString)

	// Look for others who have announced and attempt to connect to them
	anyConnected := false
	for !anyConnected {
		fmt.Println("Searching for peers...")

		peerChan, err := routingDiscovery.FindPeers(ctx, config.TopicString)
		if err != nil {
			panic(err)
		}
		for peer := range peerChan {
			if peer.ID == h.ID() {
				continue // No self connection
			}
			err := h.Connect(ctx, peer)
			if err != nil {
				// fmt.Println("Failed connecting to ", peer.ID.Pretty(), ", error:", err)
			} else {
				fmt.Println("Connected to:", peer.ID.Pretty())
				anyConnected = true
			}
		}
	}
	fmt.Println("Peer discovery complete")
}

func streamConsoleTo(ctx context.Context, topic *pubsub.Topic) {
	reader := bufio.NewReader(os.Stdin)
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		if err := topic.Publish(ctx, []byte(s)); err != nil {
			fmt.Println("### Publish error:", err)
		}
	}
}

func MessagesReceive(ctx context.Context, sub *pubsub.Subscription) {
	prefix := `{"Ct0":[{"P":`
	var key *[32]byte
	for {
		m, err := sub.Next(ctx)
		if err != nil {
			panic(err)
		}

		dataGot := string(m.Message.Data)

		if len(dataGot) >= len(prefix) && dataGot[:len(prefix)] == prefix {
			var cipher *abe.FAMECipher
			err = json.Unmarshal([]byte(dataGot), &cipher)
			if err != nil {
				panic(err)
			}
			dec, _ := ipfsAbePara.abefame.Decrypt(cipher, ipfsAbePara.keys, ipfsAbePara.mpk)
			fmt.Printf("\x1b[32m%s\x1b[0m> ", dec)

			var arr [32]byte

			copy(arr[:], []byte(dec))
			key = &arr

			fmt.Printf("Key: %x\n", key)
		} else {
			// message, err := cryptopasta.Decrypt([]byte(dataGot), key)
			// if err != nil {
			// 	panic(err)
			// }
			fmt.Println(m.ReceivedFrom, ": ", dataGot)
		}

	}
}
