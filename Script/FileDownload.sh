#!/bin/zsh
if [ $# -lt 1 ]
then
  echo "Usage: $0 arg1"
  exit 1
fi

export ABE_PROPERTIES=$(echo -n "{\"id\":\"praabe\",\"numMagnitude\":$1,\"AttributesNeeded\":3}" | base64 | tr -d \\n)

peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C mychannel -n private -c '{"function":"PrepareAbe","Args":[]}' --transient "{\"abe_properties\":\"$ABE_PROPERTIES\"}"

peer chaincode query -C mychannel -n private -c '{"function":"PrepareFileTest","Args":["http://kdd.ics.uci.edu/databases/kddcup99/kddcup.newtestdata_10_percent_unlabeled.gz","'1'"]}'
export FILE_PROPERTIES=$(echo -n "{\"paraID\":\"niceab1\"}" | base64 | tr -d \\n)
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C mychannel -n private -c '{"function":"UploadFile","Args":["praabe","192.168.2.101:5001"]}' --transient "{\"file_properties\":\"$FILE_PROPERTIES\"}"
sleep 3
echo "45MB===========45MB"
for i in {1..21}
do
peer chaincode query -C mychannel -n private -c '{"function":"GetFileFromIPFS","Args":["Org1MSPPrivateCollection","MemberKey1","192.168.2.101:5001","niceab1"]}'
done

peer chaincode query -C mychannel -n private -c '{"function":"PrepareFileTest","Args":["http://kdd.ics.uci.edu/databases/kddcup99/kddcup.newtestdata_10_percent_unlabeled.gz","'2'"]}'
export FILE_PROPERTIES=$(echo -n "{\"paraID\":\"niceab2\"}" | base64 | tr -d \\n)
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C mychannel -n private -c '{"function":"UploadFile","Args":["praabe","192.168.2.101:5001"]}' --transient "{\"file_properties\":\"$FILE_PROPERTIES\"}"
echo "90MB===========90MB"
sleep 3
for i in {1..21}
do
peer chaincode query -C mychannel -n private -c '{"function":"GetFileFromIPFS","Args":["Org1MSPPrivateCollection","MemberKey1","192.168.2.101:5001","niceab2"]}'
done

peer chaincode query -C mychannel -n private -c '{"function":"PrepareFileTest","Args":["http://kdd.ics.uci.edu/databases/kddcup99/kddcup.newtestdata_10_percent_unlabeled.gz","'4'"]}'
export FILE_PROPERTIES=$(echo -n "{\"paraID\":\"niceab3\"}" | base64 | tr -d \\n)
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C mychannel -n private -c '{"function":"UploadFile","Args":["praabe","192.168.2.101:5001"]}' --transient "{\"file_properties\":\"$FILE_PROPERTIES\"}"
echo "180MB===========180MB"
sleep 3
for i in {1..21}
do
peer chaincode query -C mychannel -n private -c '{"function":"GetFileFromIPFS","Args":["Org1MSPPrivateCollection","MemberKey1","192.168.2.101:5001","niceab3"]}'
done

peer chaincode query -C mychannel -n private -c '{"function":"PrepareFileTest","Args":["http://kdd.ics.uci.edu/databases/kddcup99/kddcup.newtestdata_10_percent_unlabeled.gz","'8'"]}'
export FILE_PROPERTIES=$(echo -n "{\"paraID\":\"niceab4\"}" | base64 | tr -d \\n)
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C mychannel -n private -c '{"function":"UploadFile","Args":["praabe","192.168.2.101:5001"]}' --transient "{\"file_properties\":\"$FILE_PROPERTIES\"}"
echo "360MB===========360MB"
sleep 3
for i in {1..21}
do
peer chaincode query -C mychannel -n private -c '{"function":"GetFileFromIPFS","Args":["Org1MSPPrivateCollection","MemberKey1","192.168.2.101:5001","niceab4"]}'
done