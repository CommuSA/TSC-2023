# A Lightweight Authentication-Driven Trusted Management Framework for IoT Collaboration

## Abstract
Essentially, the property of IoT applications is the ability to perform tasks through the collaboration of IoT devices and entities. However, security threats from both devices and entities severely hinder reliable IoT collaboration. Uncertified IoT entities may trick devices into launching illegal tasks. The adversaries can intercept the transmitted data and then modify it or forward it to an intended receiver. Besides, the traditional centralized storage system is no longer suitable for IoT due to scalability, reliability, and privacy issues. Unfortunately, most current works focus on one of the above problems and lack a holistic approach. This paper proposes a holistic trusted management framework for IoT collaboration. We design a lightweight authentication and key negotiation mechanism to ensure the reliable initialization of IoT collaboration. Besides, edge computing is introduced to enable secure and distributed data storage. We propose a fine-grained access control mechanism to avoid unauthorized data sharing. Blockchain is deployed as a decentralized trusted protocol execution environment. The experimental evaluation and security analysis prove the feasibility and effectiveness of the proposed framework.



## Before using the repo
- **/action**:  This folder is a simple DHT network built locally based on libp2p and can be used to do local content transfer in a simple AES&ABE encrypted environment.
  - **/dataowner**：Code for running the data sender
    - **/AbeSetup**: Generate the relevant parameters for Abe, including the master key, public key, policy, and the matching user keys
    - **/message**: Accept terminal input, encrypt it, and send it to the DHT network.
  - **/user**: user node, accepts data
- **chaincode**:  The main experimental code, chaincode running on fabric, a file transfer system with full permission control based on ABE encryption
- **Docker**： I have made a docker image of a simple ABE permission-controlled DHT network, you can use the following script to test the DHT network, note that this is not the main experimental code
- **Fabric-Network-Script**： Scripts for testing under fabric network
- **test** ： Some code for unit testing



## Contribution

- Implemented a complete trusted management system for IoT collaboration based on Fabric network, and this system is not only limited to IoT
- We have built a highly secure system that uses Fabric's private data collection to control parameter access rights, abe to control content reading rights, and aes encrypted session transmission under abe protection. The usual access rates are achieved while ensuring security.





## Environment

- go version go1.19.5 
- Docker version 20.10.14
- Docker-compose version 1.29.2
- Python 3.11.2
- curl 7.64.1
- jq-1.6
- ipfs version 0.18.1





## Operation process

### Fabric environment

- It is recommended that you first create a working directory, which is the official hyperedger-fabric step, which will also allow you to have the least amount of errors due to the environment.

```
mkdir -p $HOME/go/src/github.com/<your_github_userid>
cd $HOME/go/src/github.com/<your_github_userid>
```



- Then , you can use the scirpt in my repo `install-fabric.sh` to install the fabric environment, which will take a bit longer

- You can also get the install script:

  ```
  curl -sSLO https://raw.githubusercontent.com/hyperledger/fabric/main/scripts/install-fabric.sh && chmod +x install-fabric.sh
  ```



Then run the script:

```
./install-fabric.sh
```



### Start the network

- First you need to clear your test environment, go to your fabric directory

```
cd fabric-samples/test-network
./network.sh down
```

- Then start a network with ca, channel and couchdb

```shell
./network.sh up createChannel -ca -s couchdb
```

*Open another terminal and enable the monitoring process to easily see the intermediate process output*

```shell
./monitordocker.sh 
```





### Prepare the chaincode

Go to TSC-2023/chaincode

```shell
cd TSC-2023/chaincode
go mod init 
go mod tidy
go mod vendor
```

You can test the code locally for usability

```shell
go build .
```



### Deploy chaincode



Go back to the terminal of your fabric directory

```
cd fabric-samples/test-network
```



```shell
./network.sh deployCC -ccn private -ccp $HOME/go/src/github.com/TSC-2023/chaincode -ccl go -ccep "OR('Org1MSP.peer','Org2MSP.peer')" -cccg $HOME/go/src/github.com/TSC-2023/chaincode/collections_config.json
```

The endorsement strategy here is "OR('Org1MSP.peer','Org2MSP.peer')". This is in the simplest two-org experimental scenario, when you need to add orgs to build a larger network, you need to modify the endorsement strategy.

#### Register identities

First, we need to set the following environment variables to use the Fabric CA client:

```
export PATH=${PWD}/../bin:${PWD}:$PATH
export FABRIC_CFG_PATH=$PWD/../config/
```

Use the fabric-ca-client tool to register a new owner client identity, then generate the ID book and MSP folder.

```
export FABRIC_CA_CLIENT_HOME=${PWD}/organizations/peerOrganizations/org1.example.com/
fabric-ca-client register --caname ca-org1 --id.name owner --id.secret ownerpw --id.type client --tls.certfiles "${PWD}/organizations/fabric-ca/org1/tls-cert.pem"
fabric-ca-client enroll -u https://owner:ownerpw@localhost:7054 --caname ca-org1 -M "${PWD}/organizations/peerOrganizations/org1.example.com/users/owner@org1.example.com/msp" --tls.certfiles "${PWD}/organizations/fabric-ca/org1/tls-cert.pem"
cp "${PWD}/organizations/peerOrganizations/org1.example.com/msp/config.yaml" "${PWD}/organizations/peerOrganizations/org1.example.com/users/owner@org1.example.com/msp/config.yaml"

```

A similar process uses Org2 CA to create user identities

```
export FABRIC_CA_CLIENT_HOME=${PWD}/organizations/peerOrganizations/org2.example.com/
fabric-ca-client register --caname ca-org2 --id.name user --id.secret userpw --id.type client --tls.certfiles "${PWD}/organizations/fabric-ca/org2/tls-cert.pem"
fabric-ca-client enroll -u https://user:userpw@localhost:8054 --caname ca-org2 -M "${PWD}/organizations/peerOrganizations/org2.example.com/users/user@org2.example.com/msp" --tls.certfiles "${PWD}/organizations/fabric-ca/org2/tls-cert.pem"
cp "${PWD}/organizations/peerOrganizations/org2.example.com/msp/config.yaml" "${PWD}/organizations/peerOrganizations/org2.example.com/users/user@org2.example.com/msp/config.yaml"

```



## Formal start

Now that we have created the identity of the data owner and data user. Assign the environment variable so that the terminal is currently under the data owner identity of org1.

```shell
export PATH=${PWD}/../bin:$PATH
export FABRIC_CFG_PATH=$PWD/../config/
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/owner@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051
```



#### Prepare the parameters of the abe

First initialize Abe's parameters, including the owner's master key and the user's private and public keys, which are first stored in the data owner's private couchdb from the beginning.

```shell
export ABE_PROPERTIES=$(echo -n "{\"id\":\"praabe\",\"numMagnitude\":7,\"AttributesNeeded\":3}" | base64 | tr -d \\n)
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C mychannel -n private -c '{"function":"PrepareAbe","Args":[]}' --transient "{\"abe_properties\":\"$ABE_PROPERTIES\"}"

```

- function: `PrepareAbe`
  - The `id` is the unique identifier of the master key, no one else can know it, and even if they know the parameter they cannot read the key based on the id.
  - `numMagnitude`: Specify the magnitude of the attribute, i.e. the number of attributes
  - `AttributesNeeded`: The number of required user keys, this parameter specifies the amount of data users.

**Output**

```
2023-03-28 03:22:43.772 CST 0001 INFO [chaincodeCmd] chaincodeInvokeOrQuery -> Chaincode invoke successful. result: status:200 
```



*I provide a query function to check if the setup is successful, and it is also convenient for the data owner in the system to query the parameters themselves.*

`GetDataOwnerAbePara`

```
peer chaincode query -C mychannel -n private -c '{"function":"GetPeerAbePara","Args":["Org1MSPPrivateCollection","praabe"]}'
```

`GetUserAbePara`

```
peer chaincode query -C mychannel -n private -c '{"function":"GetUserAbePara","Args":["Org1MSPPrivateCollection","MemberKey1"]}'
```



#### Simulated data generation

The iot data of kddcup was downloaded and simply processed inside the temporarily generated container as the data used for transmission, which is not readable by any other node except that one.

```shell
peer chaincode query -C mychannel -n private -c '{"function":"PrepareFile","Args":["https://archive.ics.uci.edu/ml/machine-learning-databases/kddcup99-mld/kddcup.newtestdata_10_percent_unlabeled.gz"]}'

```



#### UploadFile

- Open a new terminal and run an ipfs node

  `Output`

  ```shell
  -> ipfs daemon   
  Initializing daemon...
  Kubo version: 0.18.1
  Repo version: 13
  System version: amd64/darwin
  Golang version: go1.19.1
  
  Computing default go-libp2p Resource Manager limits based on:
      - 'Swarm.ResourceMgr.MaxMemory': "8.6 GB"
      - 'Swarm.ResourceMgr.MaxFileDescriptors': 30720
  
  Applying any user-supplied overrides on top.
  Run 'ipfs swarm limit all' to see the resulting limits.
  
  Swarm listening on /ip4/127.0.0.1/tcp/4001
  Swarm listening on /ip4/127.0.0.1/udp/4001/quic
  Swarm listening on /ip4/127.0.0.1/udp/4001/quic-v1
  Swarm listening on /ip4/127.0.0.1/udp/4001/quic-v1/webtransport/certhash/uEiC6WoG84Nl513rtFkJBzYS4BREM9R2zDne74a7Q8NIE3w/certhash/uEiBTncFzPygG2OJT0qBT-UPsT8ZlOh_k2_TertVd0SjDCA
  ```

You can check the output to get the ip and port of the ipfs api . Or you can run this command `ipfs config show ` to check the complete ipfs configuration. 



- Open ipfs node api service and access method

  **You should modify the *your-ipfs-ip* to the correct ip address **

```
ipfs config --json API.HTTPHeaders.Access-Control-Allow-Origin '["http://your-ipfs-ip:5001", "http://localhost:3000", "http://127.0.0.1:5001", "https://webui.ipfs.io"]'
ipfs config --json API.HTTPHeaders.Access-Control-Allow-Methods '["PUT", "POST"]'

```



Now you can upload the local data to ipfs. The `UploadFile` function accomplishes 3 tasks, not just simply upload the data, but also encrypt the data with aes and store the aes key after using abe encryption in the private collection of the data owner along with the cid returned by ipfs.

```shell
export FILE_PROPERTIES=$(echo -n "{\"paraID\":\"fileTransfer\"}" | base64 | tr -d \\n)
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C mychannel -n private -c '{"function":"UploadFile","Args":["praabe","your-ipfs-ip:5001"]}' --transient "{\"file_properties\":\"$FILE_PROPERTIES\"}"
```

`output`

```
2023-03-28 03:59:04.716 CST 0001 INFO [chaincodeCmd] chaincodeInvokeOrQuery -> Chaincode invoke successful. result: status:200 
```



`GetUploadKey`: Check if the upload is successful and you can get the aes key and cid of the completed upload

```
peer chaincode query -C mychannel -n private -c '{"function":"GetUploadKey","Args":["fileTransfer"]}'
```

`Output`

```
{"ID":"fileTransferq","key":"{\"Ct0\":[{\"P\":{\"X\":{\"X\":[14624952514284084633,.......1785251046713]}}}},\"Msp\":{\"P\":null,\"Mat\":[[0,-1,0,0,0,0,0,0,0,......wi\",\"otybu\",\"muqv\"]},\"SymEnc\":\"eJyfCj5d0fiKe9s/ZuW+mdsjP/yCb4mDM9Nz6Q9FgT9TRlT9/21TcYq/NWdTMbnG\",\"Iv\":\"CjKK6H7xchrqb5S6d3fjLA==\"}","cid":"QmVaerNzs23VmKCen7tfnKsjB8JTP423nvZJBR2X1AqeMT","Owner":"x509::CN=owner,OU=client,O=Hyperledger,ST=North Carolina,C=US::CN=ca.org1.example.com,O=org1.example.com,L=Durham,ST=North Carolina,C=US"}
```



#### Check if the permission management is successful

Change the environment variables and switch to a user peer in Org2

```
export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/user@org2.example.com/msp
export CORE_PEER_ADDRESS=localhost:9051
```



Run the query command provided above to check if the data can be read



```
peer chaincode query -C mychannel -n private -c '{"function":"GetPeerAbePara","Args":["Org1MSPPrivateCollection","praabe"]}'
```

Or

```
peer chaincode query -C mychannel -n private -c '{"function":"GetUserAbePara","Args":["Org1MSPPrivateCollection","MemberKey1"]}'
```



`output`

```
Error: endorsement failure during query. response: status:500 message:"failed to read from world state: GET_STATE failed: transaction ID: 5fbd37e1254c3fb5c0bd7241a5213015114d0920a89a1cc98f4f9e0c97b0c3ad: tx creator does not have read access permission on privatedata in chaincodeName:private collectionName: Org1MSPPrivateCollection" 
```



- When query the UploadKey, the parameters were successfully output. Because the UploadKey is public data.

```
peer chaincode query -C mychannel -n private -c '{"function":"GetUploadKey","Args":["fileTransfer"]}'
```



#### Request for ABE Para

* in org2-user

```
export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/user@org2.example.com/msp
export CORE_PEER_ADDRESS=localhost:9051

export PATH=${PWD}/../bin:${PWD}:$PATH
export FABRIC_CFG_PATH=$PWD/../config/
```
run

```
export APPLY_VALUE=$(echo -n "{\"paraID\":\"MemberKey1\",\"applyInfo\":\"Apply to get the Kdd data\"}" | base64 | tr -d \\n)
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C mychannel -n private -c '{"function":"ApplyForMem","Args":[]}' --transient "{\"apply_value\":\"$APPLY_VALUE\"}"
```

The user requests the user private key from the data owner to join the data sharing network. Here we omit the application form information and use a statement instead. The application information contains the user's identity, address and attribute information, and the data owner is able to determine whether the node is qualified n get the user key by this information.



#### Agree the apply

- Switch to org1-dataowner 

  ```
  export CORE_PEER_TLS_ENABLED=trueexport CORE_PEER_LOCALMSPID="Org1MSP"                                                                                             
  export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
  export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/owner@org1.example.com/msp
  export CORE_PEER_ADDRESS=localhost:7051
  ```

  

- in org1-user

```
export PARA_OWNER=$(echo -n "{\"paraID\":\"MemberKey1\"}" | base64 | tr -d \\n)
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C mychannel -n private -c '{"function":"SetUserAbePara","Args":[]}' --transient "{\"para_owner\":\"$PARA_OWNER\"}" --peerAddresses localhost:7051 --tlsRootCertFiles "${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt"
```

Within the function `SetUserAbePara`, the application form information sent by the user is read, and the identity and attributes of the user are judged to be eligible. After successful judgment, data owner puts MemberKey1 into the channel that only data owner and the applicant can access, and sets this channel to exist only for 3 block cycles, and MemberKey1 is deleted from data owner's collection. Now the applicant can read the MemberKey1 and put it into its own collection.



#### Access Control Verification

Again this command in org2-user, the terminal now returns the full parameters of memberkey.

```
peer chaincode query -C mychannel -n private -c '{"function":"GetUserAbePara","Args":["assetCollection","MemberKey1"]}'
```



#### The user download the file  

```
peer chaincode query -C mychannel -n private -c '{"function":"GetFileFromIPFS","Args":["Org1MSPPrivateCollection","MemberKey1","your-ipfs-ip:5001","fileTransfer"]}'
```

`GetFileFromIPFS` function downloads the data from ipfs to the container of user and decrypts the aes key by using abe key, then decrypts the file by using the aes key and now the read content is the plaintext of the data.



## Dataset

[KDD Cup 1999 Data](http://kdd.ics.uci.edu/databases/kddcup99/kddcup99.html)
