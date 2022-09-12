#!/bin/bash

# Copyright 2021, 2022 Hewlett Packard Enterprise Development LP
# Other additional copyright holders may be indicated within.
#
# The entirety of this work is licensed under the Apache License,
# Version 2.0 (the "License"); you may not use this file except
# in compliance with the License.
#
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#
# nnf-ec makes lvm calls within the nnf-manager container. Udev operations in this case are problematic.
# Refer to: https://serverfault.com/questions/802766/calling-lvcreate-from-inside-the-container-hangs
#
# Disable lvm udev option:
#   `udev_sync`
# to allow nnf-ec running within a pod to execute `lvm` commands without udev interfering
disable_lvm_udev_processing ()
{
    echo "Disable udev_sync and udev_rules on each rabbit in the cluster"
    RABBIT_NODES=$(kubectl get nodes --selector=cray.nnf.node=true --no-headers | awk '{print $1}')

    for RABBIT in $RABBIT_NODES; do
        echo "$RABBIT"
        ssh -t "$RABBIT" "sed -e 's|udev_sync = 1|udev_sync = 0|g' /etc/lvm/lvm.conf > /etc/lvm/lvm.confnew; mv -f /etc/lvm/lvm.confnew /etc/lvm/lvm.conf"
        ssh -t "$RABBIT" grep udev_[r:s] /etc/lvm/lvm.conf
        ssh -
    done
}

disable_lvm_udev_processing
