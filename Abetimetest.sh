#!/bin/zsh

if [ $# -lt 2 ]
then
  echo "Usage: $0 arg1 arg2"
  exit 1
fi

for i in {1..21}
do
    peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C mychannel -n private -c '{"function":"PrepareAbeTest","Args":["'$1'","'$2'"]}'
done
