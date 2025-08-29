#!/bin/bash

set -e
set -x

kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: flux
subjects:
- kind: User
  name: flux
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: nnf-workload-manager
  apiGroup: rbac.authorization.k8s.io
---
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: flux
  namespace: default
subjects:
- kind: User
  name: flux
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: nnf-workload-manager-coregrp
  apiGroup: ""
EOF

