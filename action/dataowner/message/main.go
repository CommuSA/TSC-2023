package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/gtank/cryptopasta"

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

type AbePeerPara struct {
	abefame *abe.FAME
	mpk     *abe.FAMEPubKey
	msk     *abe.FAMESecKey
	msp     *abe.MSP
}

var peerAbePara AbePeerPara

func getAbePara() AbePeerPara {
	var abepara AbePeerPara
	err := godotenv.Load("../AbeSetup/.env")
	if err != nil {
		panic(err)
	}
	abefameJSON := os.Getenv("ABEFAME")
	err = json.Unmarshal([]byte(abefameJSON), &abepara.abefame)
	if err != nil {
		panic(err)
	}

	mpkJSON := os.Getenv("MPK")
	err = json.Unmarshal([]byte(mpkJSON), &abepara.mpk)
	if err != nil {
		panic(err)
	}

	mskJSON := os.Getenv("MSK")
	err = json.Unmarshal([]byte(mskJSON), &abepara.msk)
	if err != nil {
		panic(err)
	}

	mspJSON := os.Getenv("MSP")
	err = json.Unmarshal([]byte(mspJSON), &abepara.msp)
	if err != nil {
		panic(err)
	}
	return abepara
}

func main() {
	config, _ = ParseFlags()
	ctx := context.Background()
	peerAbePara = getAbePara()
	absKey := cryptopasta.NewEncryptionKey()

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
	go streamConsoleTo(ctx, topic, absKey)

	sub, err := topic.Subscribe()
	if err != nil {
		panic(err)
	}
	printMessagesFrom(ctx, sub)
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

func sendKey(ctx context.Context, topic *pubsub.Topic, key *[32]byte) {

	sendData, _ := peerAbePara.abefame.Encrypt(string(key[:]), peerAbePara.msp, peerAbePara.mpk)
	sendDataJson, _ := json.Marshal(sendData)
	sendDatastr := string(sendDataJson)

	err := topic.Publish(ctx, []byte(sendDatastr))
	if err != nil {
		fmt.Println("### Publish error:", err)
	}
}

func streamSend(ctx context.Context, topic *pubsub.Topic, key *[32]byte) {

	dir, err := os.Open(config.MessageDirPath)
	if err != nil {
		panic(err)
		fmt.Println(config.MessageDirPath)
	}
	defer dir.Close()

	files, err := dir.Readdir(0)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		filePath := config.MessageDirPath + "/" + file.Name()
		file, err := os.Open(filePath)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		reader := bufio.NewReader(file)
		sendDataGot, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}

		sendDatastr, err := cryptopasta.Encrypt([]byte(sendDataGot), key)
		if err != nil {
			panic(err)
		}
		if err = topic.Publish(ctx, []byte(sendDatastr)); err != nil {
			fmt.Println("### Publish error:", err)
		}
	}
}

func streamConsoleTo(ctx context.Context, topic *pubsub.Topic, key *[32]byte) {
	reader := bufio.NewReader(os.Stdin)
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		if s == "send AES Key\n" {
			sendKey(ctx, topic, key)
		} else if s == "send msg kdd\n" {
			streamSend(ctx, topic, key)
		} else {
			// sendData, _ := peerAbePara.abefame.Encrypt(s, peerAbePara.msp, peerAbePara.mpk)
			// sendDataJson, _ := json.Marshal(sendData)
			// sendDatastr := string(sendDataJson)
			sendDatastr, err := cryptopasta.Encrypt([]byte(s), key)
			if err != nil {
				panic(err)
			}
			err = topic.Publish(ctx, []byte(sendDatastr))
			if err != nil {
				fmt.Println("### Publish error:", err)
			}
		}
	}
}

func printMessagesFrom(ctx context.Context, sub *pubsub.Subscription) {
	for {
		m, err := sub.Next(ctx)
		if err != nil {
			panic(err)
		}
		fmt.Println(m.ReceivedFrom, ": ", string(m.Message.Data))
		// inputBytes := []byte(string(m.Message.Data))

		// ioutil.WriteFile("output.txt", inputBytes, 0644)

	}
}
