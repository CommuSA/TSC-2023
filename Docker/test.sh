
chmod a+X create-peer-node
chmod a+X create-master-node

./create-master-node master 4000 5000 8000
./create-peer-node ipfs-node1 4001 5001 8001
./create-peer-node ipfs-node2 4002 5002 8002
./create-peer-node ipfs-node3 4003 5003 8003
