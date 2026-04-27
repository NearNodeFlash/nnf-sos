"""CRD group, version, and plural constants for NNF and DWS resources."""

# ---------------------------------------------------------------------------
# NNF (nnf.cray.hpe.com)
# ---------------------------------------------------------------------------

NNF_GROUP = "nnf.cray.hpe.com"
# Update NNF_VERSION when the NNF CRDs are bumped (e.g. v1alpha11 → v1alpha12).
NNF_VERSION = "v1alpha11"

NNF_NODE_PLURAL = "nnfnodes"
NNF_STORAGE_PLURAL = "nnfstorages"
NNF_ACCESS_PLURAL = "nnfaccesses"
NNF_DATA_MOVEMENT_PLURAL = "nnfdatamovements"
NNF_STORAGE_PROFILE_PLURAL = "nnfstorageprofiles"
NNF_DATA_MOVEMENT_PROFILE_PLURAL = "nnfdatamovementprofiles"
NNF_CONTAINER_PROFILE_PLURAL = "nnfcontainerprofiles"
NNF_SYSTEM_STORAGE_PLURAL = "nnfsystemstorages"

# ---------------------------------------------------------------------------
# DWS (dataworkflowservices.github.io)
# ---------------------------------------------------------------------------

DWS_GROUP = "dataworkflowservices.github.io"
# Update DWS_VERSION when the DWS CRDs are bumped (e.g. v1alpha7 → v1alpha8).
DWS_VERSION = "v1alpha7"

DWS_WORKFLOW_PLURAL = "workflows"
DWS_PERSISTENT_STORAGE_PLURAL = "persistentstorageinstances"
DWS_STORAGE_PLURAL = "storages"
DWS_DIRECTIVE_BREAKDOWN_PLURAL = "directivebreakdowns"
DWS_SERVERS_PLURAL = "servers"
DWS_COMPUTES_PLURAL = "computes"
DWS_SYSTEM_CONFIGURATION_PLURAL = "systemconfigurations"
