package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	NnfWorkflowReconcilesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nnf_workflow_reconciles_total",
			Help: "Number of total reconciles in nnf_workflow controller",
		},
	)

	NnfDirectiveBreakdownReconcilesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nnf_directive_breakdown_reconciles_total",
			Help: "Number of total reconciles in directive_breakdown controller",
		},
	)

	NnfAccessReconcilesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nnf_access_reconciles_total",
			Help: "Number of total reconciles in nnf_access controller",
		},
	)

	NnfClientMountReconcilesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nnf_clientmount_reconciles_total",
			Help: "Number of total reconciles in nnf_clientmount controller",
		},
	)

	NnfNodeReconcilesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nnf_node_reconciles_total",
			Help: "Number of total reconciles in nnf_node controller",
		},
	)

	NnfNodeECDataReconcilesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nnf_node_ec_data_reconciles_total",
			Help: "Number of total reconciles in nnf_node_ec_data controller",
		},
	)

	NnfNodeStorageReconcilesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nnf_node_storage_reconciles_total",
			Help: "Number of total reconciles in nnf_node_storage controller",
		},
	)

	NnfNodeBlockStorageReconcilesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nnf_node_block_storage_reconciles_total",
			Help: "Number of total reconciles in nnf_node_block_storage controller",
		},
	)

	NnfPersistentStorageReconcilesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nnf_persistent_storage_reconciles_total",
			Help: "Number of total reconciles in nnf_persistentstorage controller",
		},
	)

	NnfServersReconcilesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nnf_servers_reconciles_total",
			Help: "Number of total reconciles in nnf_servers controller",
		},
	)

	NnfStorageReconcilesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nnf_storage_reconciles_total",
			Help: "Number of total reconciles in nnf_storage controller",
		},
	)

	NnfLustreMGTReconcilesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nnf_lustre_mgt_reconciles_total",
			Help: "Number of total reconciles in nnf_lustre_mgt controller",
		},
	)

	NnfSystemConfigurationReconcilesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nnf_system_configuration_reconciles_total",
			Help: "Number of total reconciles in nnf_systemconfiguration controller",
		},
	)

	NnfSystemStorageReconcilesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nnf_system_storage_reconciles_total",
			Help: "Number of total reconciles in nnf_systemstorage controller",
		},
	)

	Gfs2FenceRequestsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gfs2_fence_requests_total",
			Help: "Number of GFS2 fence requests detected by the file watcher",
		},
	)
)

func init() {
	metrics.Registry.MustRegister(NnfWorkflowReconcilesTotal)
	metrics.Registry.MustRegister(NnfDirectiveBreakdownReconcilesTotal)
	metrics.Registry.MustRegister(NnfAccessReconcilesTotal)
	metrics.Registry.MustRegister(NnfClientMountReconcilesTotal)
	metrics.Registry.MustRegister(NnfNodeReconcilesTotal)
	metrics.Registry.MustRegister(NnfNodeECDataReconcilesTotal)
	metrics.Registry.MustRegister(NnfNodeStorageReconcilesTotal)
	metrics.Registry.MustRegister(NnfNodeBlockStorageReconcilesTotal)
	metrics.Registry.MustRegister(NnfPersistentStorageReconcilesTotal)
	metrics.Registry.MustRegister(NnfServersReconcilesTotal)
	metrics.Registry.MustRegister(NnfStorageReconcilesTotal)
	metrics.Registry.MustRegister(NnfLustreMGTReconcilesTotal)
	metrics.Registry.MustRegister(NnfSystemConfigurationReconcilesTotal)
	metrics.Registry.MustRegister(NnfSystemStorageReconcilesTotal)
	metrics.Registry.MustRegister(Gfs2FenceRequestsTotal)
}
