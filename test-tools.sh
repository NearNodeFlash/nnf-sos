#!/usr/bin/env bash

function prefix_webhook_names {
    SOURCE_DIR=$1
    DEST_DIR=$2

    mkdir -p $DEST_DIR
    cp $SOURCE_DIR/manifests.yaml $DEST_DIR
    sed -i.bak -e 's/validating-webhook-configuration/nnf-validating-webhook-configuration/' $DEST_DIR/manifests.yaml
    rm $DEST_DIR/manifests.yaml.bak

    cp $SOURCE_DIR/service.yaml $DEST_DIR
    sed -i.bak -e 's/webhook-service/nnf-webhook-service/' $DEST_DIR/service.yaml
    rm $DEST_DIR/service.yaml.bak
}
