package ops

import (
	"context"
	opsv1 "github.com/taisho6339/multicluster-upgrade-operator/api/v1"
)

// Operator requests for the operation server to perform the cluster operations.
type Operator interface {
	// GetOperationStatus gets operations status.
	GetOperationStatus(ctx context.Context, obj opsv1.ClusterVersion) (OperationStatus, error)
	// GetClusterVersion gets current versions of the clusters.
	GetClusterVersion(ctx context.Context, obj opsv1.ClusterVersion, cluster opsv1.Cluster) (*ClusterVersion, error)
	// GetClusterStatus gets the cluster status.
	GetClusterStatus(ctx context.Context, obj opsv1.ClusterVersion, cluster opsv1.Cluster) (*ClusterStatus, error)
	// ServiceIn requests the operation for servicein the cluster.
	ServiceIn(ctx context.Context, obj opsv1.ClusterVersion, cluster opsv1.Cluster) (*OperationResult, error)
	// ServiceOut requests the operation for serviceout the cluster.
	ServiceOut(ctx context.Context, obj opsv1.ClusterVersion, cluster opsv1.Cluster) (*OperationResult, error)
	// UpgradeMaster requests the operation for upgrading the master of the cluster.
	UpgradeMaster(ctx context.Context, obj opsv1.ClusterVersion, cluster opsv1.Cluster) (*OperationResult, error)
	// UpgradeNodePool requests the operation for upgrading the node pool of the cluster.
	UpgradeNodePool(ctx context.Context, obj opsv1.ClusterVersion, cluster opsv1.Cluster, nodePoolID string) (*OperationResult, error)
}

// ClusterStatus shows cluster status.
type ClusterStatus struct {
	Type      ClusterStatusType
	Available bool
}

// OperationResult shows running operation.
type OperationResult struct {
	OperationID   string
	OperationType string
}

// OperationStatus shows operation status.
type OperationStatus string

const (
	// OperationStatusDone shows the operation has done.
	OperationStatusDone OperationStatus = "Done"
	// OperationStatusRunning shows the operation is running.
	OperationStatusRunning OperationStatus = "Running"
	// OperationStatusFailed shows the operation has failed.
	OperationStatusFailed OperationStatus = "Failed"
	// OperationStatusUnknown show the operation status is unknown.
	OperationStatusUnknown OperationStatus = "Unknown"
)

// ClusterStatusType shows the cluster status.
type ClusterStatusType string

const (
	// ClusterStatusServiceUnknown shows the cluster status is unknown.
	ClusterStatusServiceUnknown ClusterStatusType = "Unknown"
	// ClusterStatusServiceIn shows the cluster status is servicein.
	ClusterStatusServiceIn ClusterStatusType = "ServiceIn"
	// ClusterStatusServiceOut shows the cluster status is serviceout.
	ClusterStatusServiceOut ClusterStatusType = "ServiceOut"
)

// ClusterVersion shows the cluster version.
type ClusterVersion struct {
	Master    MasterVersion
	NodePools []NodePoolVersion
}

// MasterVersion shows the version of the master of specified cluster.
type MasterVersion struct {
	ClusterID string
	Version   string
}

// NodePoolVersion shows the version of specified node pool.
type NodePoolVersion struct {
	NodePoolID string
	Version    string
}
