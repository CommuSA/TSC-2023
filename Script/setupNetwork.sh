
if [ $# != 1 ] ; then
	echo "1 args needed num"
	exit 1;
fi
# ./network.sh up createChannel -ca -s couchdb

./network.sh deployCC -ccn private -ccp $HOME/go/src/github.com/TSC-2023/chaincode -ccl go -ccep "OR('Org1MSP.peer','Org2MSP.peer')" -cccg $HOME/go/src/github.com/TSC-2023/chaincode/collections_config.json

export PATH=${PWD}/../bin:${PWD}:$PATH
export FABRIC_CFG_PATH=$PWD/../config/

export FABRIC_CA_CLIENT_HOME=${PWD}/organizations/peerOrganizations/org1.example.com/
fabric-ca-client register --caname ca-org1 --id.name owner --id.secret ownerpw --id.type client --tls.certfiles "${PWD}/organizations/fabric-ca/org1/tls-cert.pem"
fabric-ca-client enroll -u https://owner:ownerpw@localhost:7054 --caname ca-org1 -M "${PWD}/organizations/peerOrganizations/org1.example.com/users/owner@org1.example.com/msp" --tls.certfiles "${PWD}/organizations/fabric-ca/org1/tls-cert.pem"
cp "${PWD}/organizations/peerOrganizations/org1.example.com/msp/config.yaml" "${PWD}/organizations/peerOrganizations/org1.example.com/users/owner@org1.example.com/msp/config.yaml"
export FABRIC_CA_CLIENT_HOME=${PWD}/organizations/peerOrganizations/org2.example.com/
fabric-ca-client register --caname ca-org2 --id.name user --id.secret userpw --id.type client --tls.certfiles "${PWD}/organizations/fabric-ca/org2/tls-cert.pem"
fabric-ca-client enroll -u https://user:userpw@localhost:8054 --caname ca-org2 -M "${PWD}/organizations/peerOrganizations/org2.example.com/users/user@org2.example.com/msp" --tls.certfiles "${PWD}/organizations/fabric-ca/org2/tls-cert.pem"
cp "${PWD}/organizations/peerOrganizations/org2.example.com/msp/config.yaml" "${PWD}/organizations/peerOrganizations/org2.example.com/users/user@org2.example.com/msp/config.yaml"
export PATH=${PWD}/../bin:$PATH
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/owner@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051
# export ASSET_PROPERTIES=$(echo -n "{\"objectType\":\"para\",\"paraID\":\"para3\",\"color\":\"green\",\"size\":20,\"appraisedValue\":100}" | base64 | tr -d \\n)
# peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C mychannel -n private -c '{"function":"CreateAsset","Args":[]}' --transient "{\"para_properties\":\"$ASSET_PROPERTIES\"}"
# export ABE_PROPERTIES=$(echo -n "{\"id\":\"praabe\",\"numMagnitude\":7,\"AttributesNeeded\":3}" | base64 | tr -d \\n)
# peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C mychannel -n private -c '{"function":"PrepareAbe","Args":[]}' --transient "{\"abe_properties\":\"$ABE_PROPERTIES\"}"
# # peer chaincode query -C mychannel -n private -c '{"function":"AesKeyDecrypt","Args":["add"]}'
# peer chaincode query -C mychannel -n private -c '{"function":"GetFileFromIPFS","Args":["Org1MSPPrivateCollection","192.168.2.102:5001","adq"]}'
# export ABE_PROPERTIES=$(echo -n "{\"id\":\"praabe\",\"numMagnitude\":10,\"AttributesNeeded\":3}" | base64 | tr -d \\n)

# peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C mychannel -n private -c '{"function":"PrepareAbe","Args":[]}' --transient "{\"abe_properties\":\"$ABE_PROPERTIES\"}"
# peer chaincode query -C mychannel -n private -c '{"function":"PrepareFileTest","Args":["https://archive.ics.uci.edu/ml/machine-learning-databases/kddcup99-mld/kddcup.newtestdata_10_percent_unlabeled.gz","1"]}'