#!/bin/bash

# Variables
NAMESPACE="consul"
CONSUL_HELM_RELEASE="consul"
CONSUL_PORT=8500
CONSUL_HELM_CHART="hashicorp/consul"

# Step 1: Add HashiCorp Helm repository
echo "Adding HashiCorp Helm repository..."
helm repo add hashicorp https://helm.releases.hashicorp.com
helm repo update

# Step 2: Create a Namespace for Consul
echo "Creating Kubernetes namespace for Consul..."
kubectl create namespace $NAMESPACE

# Step 3: Install Consul on Kubernetes
echo "Installing Consul via Helm chart..."
helm install $CONSUL_HELM_RELEASE $CONSUL_HELM_CHART --set global.name=consul --namespace $NAMESPACE

# Step 4: Wait for Consul to be up and running
echo "Waiting for Consul pods to be ready..."
kubectl wait --for=condition=ready pod -l app=consul -n $NAMESPACE --timeout=300s

# Step 5: Port-forward to access the Consul UI
echo "Setting up port-forwarding for Consul UI..."
kubectl port-forward service/${CONSUL_HELM_RELEASE}-server $CONSUL_PORT:$CONSUL_PORT -n $NAMESPACE &

# Step 6: Check the status of Consul installation
echo "Checking Consul installation status..."
kubectl get all -n $NAMESPACE

# Step 7: Print access instructions for Consul UI
echo "========================================="
echo "Consul has been installed!"
echo "You can access the Consul UI at http://localhost:$CONSUL_PORT"
echo "========================================="

# End of script
