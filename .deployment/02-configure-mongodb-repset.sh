
# Script to connect to the first mongodb instance running in container for the
# kubernetes stateful set, via mongo shell, to initialize a MongoDB Replica set
# and create a mongodb admin user

# Run when all the replicas are up and running - 3 stateful set mongodb pods

if [[ $# -eq 0 ]] ; then
	echo "You must provide one argument for the password of the 'admin_user' user to be created"
	echo 'usage : 02-configure-mongodb-repset.sh MyPassword123'
	ehco
	exit 1
fi

echo "Configuring the MongoDB Replica Set"
kubectl exec mongodb-0 -c mongodb-container -- mongo --eval 'rs.initiate({_id: "MainRepSet", version: 1, members: [{_id: 0, host: "mongodb-0.mongodb-service.default.svc.cluster.local:27017"}, {_id: 1, host: "mongodb-1.mongodb-service.default.svc.cluster.local:27017"}, {_id: 2, host: "mongodb-2.mongodb-service.default.svc.cluster.local:27017"} ]});'

echo "Waiting for the Replica Set to initialize..."
sleep 30
kubectl exec mongodb-0 -c mongodb-container -- mongo --eval 'rs.status();'

echo "Creating user: 'admin_user'"
kubectl exec mongodb-0 -c mongodb-container -- mongo --eval 'db.getSiblingDB("admin").createUser({user: "admin_user", pwd:"'"${1}"'", roles: [{role: "root", db: "admin"}]});'
echo