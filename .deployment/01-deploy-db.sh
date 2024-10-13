
# Create a keyfile for the MongoDB cluster as a kubernetes shared secret
TEMPFILE=$(mktemp)
/usr/bin/openssl rand -base64 741 > $TEMPFILE
kubectl create secret generic shared-bootstrap-data --from-file=internal-auth-mongodb-keyfile=$TEMPFILE
rm $TEMPFILE

# create mongodb service with mongodb sateful-set
kubectl apply -f ./mongo-deployment.yml --validate=false
sleep 5

kubectl get all
kubectl get persistent-volumes

echo
echo "Keep running the follwing command until all 'mongodb-n' pods are shown as running: kubectl getall"
echo