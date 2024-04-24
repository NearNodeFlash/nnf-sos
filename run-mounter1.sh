#!/bin/bash

[[ -z $KUBECONFIG ]] && export KUBECONFIG=~/.kube/config

EXTRA="$1"

CLUSTER=$(kubectl config get-contexts | grep \* | awk '{print $3}')
SERVER_URL=$(kubectl config view | yq -Mr '.clusters[] | select(.name == "'"$CLUSTER"'") | .cluster.server')
SERVER_PORT=$(echo "$SERVER_URL" | sed -e 's/^http.*:\/\///')

SRVR=$(echo "$SERVER_PORT" | awk -F: '{print $1}')
PORT=$(echo "$SERVER_PORT" | awk -F: '{print $2}')

echo "CONF=$KUBECONFIG"
echo "SERVER=$SRVR"
echo "PORT=$PORT"

set -x

NODENAME=compute-01
NS=nnf-system
SECRET=nnf-clientmount

kubectl get secret -n $NS $SECRET -o json | jq -Mr '.data."ca.crt"' | base64 --decode > mounter1-ca.crt

kubectl get secret -n $NS $SECRET -o json | jq -Mr .data.token | base64 --decode > mounter1-token

exec bin/mounter1 --kubeconfig $KUBECONFIG --node-name $NODENAME --kubernetes-service-host=$SRVR --kubernetes-service-port=$PORT --service-cert-file mounter1-ca.crt --service-token-file mounter1-token $EXTRA


