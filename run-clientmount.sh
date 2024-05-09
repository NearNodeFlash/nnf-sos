#!/bin/bash

[[ -z $KUBECONFIG ]] && export KUBECONFIG=~/.kube/config

CLUSTER=$(kubectl config get-contexts | grep \* | awk '{print $3}')
SERVER_URL=$(kubectl config view -o json | jq -Mr '.clusters[] | select(.name == "'"$CLUSTER"'") | .cluster.server')
SERVER_PORT=$(echo "$SERVER_URL" | sed -e 's/^http.*:\/\///')

SRVR=$(echo "$SERVER_PORT" | awk -F: '{print $1}')
PORT=$(echo "$SERVER_PORT" | awk -F: '{print $2}')

#echo "CONF=$KUBECONFIG"
#echo "SERVER=$SRVR"
#echo "PORT=$PORT"

NODENAME=dean-ns
NS=nnf-system
SECRET=nnf-clientmount

if [[ ! -f ca.crt || ! -s ca.crt ]]; then
    if CRT=$(kubectl get secret -n $NS $SECRET -o json | jq -Mr '.data."ca.crt"' | base64 --decode); then
        echo "$CRT" > ca.crt
    fi
fi

if [[ ! -f token || ! -s token ]]; then
    if TOK=$(kubectl get secret -n $NS $SECRET -o json | jq -Mr .data.token | base64 --decode); then
        echo "$TOK" > token
    fi
fi

# Tell library funcs we are in KIND.
export ENVIRONMENT=kind

exec bin/clientmounter --node-name $NODENAME --kubernetes-service-host=$SRVR --kubernetes-service-port=$PORT --service-cert-file ca.crt --service-token-file token --requeue-delay 1s

