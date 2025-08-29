#!/bin/bash

set -e
set -o pipefail

fluxdir=$(mktemp -d /tmp/flux-role.XXXX)
cd "$fluxdir"

CURRENT_CONTEXT=$(yq -rM .current-context ~/.kube/config)
# shellcheck disable=SC2049
if [[ $CURRENT_CONTEXT =~ *kind* ]]
then
    echo "This tool expects a basic KIND environment."
    exit 1
fi

SERVER_ADDRESS=$(yq -rM '.clusters[]|select(.name=="'"$CURRENT_CONTEXT"'")|.cluster.server'  ~/.kube/config)

docker exec kind-control-plane cat /etc/kubernetes/pki/ca.crt > ca.crt
docker exec kind-control-plane cat /etc/kubernetes/pki/ca.key > ca.key

export USERNAME=flux
export CLUSTER_NAME=$CURRENT_CONTEXT

# generate a new key
openssl genrsa -out rabbit.key 2048

# create a certificate signing request for this user
openssl req -new -key rabbit.key -out rabbit.csr -subj "/CN=$USERNAME"

# generate a certificate using the certificate authority on the k8s cluster. This certificate lasts 500 days
openssl x509 -req -in rabbit.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out rabbit.crt -days 500

# create a new kubeconfig with the server information
kubectl config set-cluster "$CLUSTER_NAME" --kubeconfig=rabbit.conf --server="$SERVER_ADDRESS" --certificate-authority=ca.crt --embed-certs=true

# add the key and cert for this user to the config
kubectl config set-credentials "$USERNAME" --kubeconfig=rabbit.conf --client-certificate=rabbit.crt --client-key=rabbit.key --embed-certs=true

# add a context
kubectl config set-context "$USERNAME" --kubeconfig=rabbit.conf --cluster="$CLUSTER_NAME" --user="$USERNAME"

# activate the context
kubectl config use-context "$USERNAME" --kubeconfig=rabbit.conf

echo "The flux kubeconfig is $fluxdir/rabbit.conf"

