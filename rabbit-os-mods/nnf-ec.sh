#!/bin/bash

#
# nnf-ec makes lvm calls within the nnf-manager container. Udev operations in this case are problematic.
# Refer to: https://serverfault.com/questions/802766/calling-lvcreate-from-inside-the-container-hangs
#
# Disable lvm udev options:
#   `udev_sync`
#   `udev_rules`
# to allow nnf-ec running within a pod to execute `lvm` commands without udev interfering
disable_lvm_udev_processing ()
{
    echo "Disable udev_sync and udev_rules on each rabbit in the cluster"
    RABBIT_NODES=$(kubectl get nodes --selector=cray.nnf.node=true --no-headers | awk '{print $1}')

    for RABBIT in $RABBIT_NODES; do
        echo "$RABBIT"
        ssh -t "$RABBIT" "sed -e 's|udev_sync = 1|udev_sync = 0|g' -e 's|udev_rules = 1|udev_rules = 0|g' /etc/lvm/lvm.conf > /etc/lvm/lvm.confnew; mv -f /etc/lvm/lvm.confnew /etc/lvm/lvm.conf"
        ssh -t "$RABBIT" grep udev_[r:s] /etc/lvm/lvm.conf
        ssh -
    done
}

disable_lvm_udev_processing
