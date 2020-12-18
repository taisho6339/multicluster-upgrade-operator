package ops

import (
	"context"
	"fmt"
	"github.com/taisho6339/multicluster-upgrade-operator-proto/go/plugin"
	opsv1 "github.com/taisho6339/multicluster-upgrade-operator/api/v1"
	"google.golang.org/grpc"
	ctrl "sigs.k8s.io/controller-runtime"
)

type newConnFunc func(obj opsv1.ClusterVersion) (c plugin.ClusterClient, closer func(), err error)

type pluginOperator struct {
	newFunc newConnFunc
}

const (
	metricsGetClusterStatus   = "GetClusterStatus"
	metricsGetOperationStatus = "GetOperationStatus"
	metricsGetClusterVersion  = "GetClusterVersion"
	metricsServiceIn          = "ServiceIn"
	metricsServiceOut         = "ServiceOut"
	metricsUpgradeMaster      = "UpgradeMaster"
	metricsUpgradeNodePool    = "UpgradeNodePool"
)

var (
	DefaultNewFunc = func(obj opsv1.ClusterVersion) (c plugin.ClusterClient, closer func(), err error) {
		var opts []grpc.DialOption
		if obj.Spec.OpsEndpoint.Insecure {
			opts = append(opts, grpc.WithInsecure())
		}
		conn, err := grpc.Dial(obj.Spec.OpsEndpoint.Endpoint, opts...)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to dial grpc. err: %#v", err)
		}
		return plugin.NewClusterClient(conn), func() {
			if err := conn.Close(); err != nil {
				ctrl.Log.Error(err, "failed to close connection")
			}
		}, nil
	}
)

func NewPluginOperator(newFunc newConnFunc) Operator {
	return &pluginOperator{
		newFunc: newFunc,
	}
}

func (p *pluginOperator) GetClusterStatus(ctx context.Context, obj opsv1.ClusterVersion, cluster opsv1.Cluster) (*ClusterStatus, error) {
	c, closer, err := p.newFunc(obj)
	if err != nil {
		return nil, err
	}
	defer closer()
	req := &plugin.GetClusterStatusRequest{
		ClusterID: cluster.ID,
	}
	res, err := c.GetClusterStatus(ctx, req)
	if err != nil {
		addFailedPluginServerCall(metricsGetClusterStatus)
		return nil, err
	}
	addSuccessPluginServerCall(metricsGetClusterStatus)
	switch res.Status {
	case plugin.ClusterStatusType_STATUS_SERVICE_IN:
		return &ClusterStatus{
			Type:      ClusterStatusServiceIn,
			Available: res.IsAvailable,
		}, nil
	case plugin.ClusterStatusType_STATUS_SERVICE_OUT:
		return &ClusterStatus{
			Type:      ClusterStatusServiceOut,
			Available: res.IsAvailable,
		}, nil
	}
	return &ClusterStatus{
		Type:      ClusterStatusServiceUnknown,
		Available: false,
	}, fmt.Errorf("no match cluster status. status: %s", res.Status)
}

func (p *pluginOperator) GetOperationStatus(ctx context.Context, obj opsv1.ClusterVersion) (OperationStatus, error) {
	c, closer, err := p.newFunc(obj)
	if err != nil {
		return OperationStatusUnknown, err
	}
	defer closer()
	req := &plugin.GetOperationStatusRequest{
		ClusterID:   obj.Status.ClusterID,
		OperationID: obj.Status.OperationID,
		Type:        obj.Status.OperationType,
	}
	st, err := c.GetOperationStatus(ctx, req)
	if err != nil {
		addFailedPluginServerCall(metricsGetOperationStatus)
		return OperationStatusUnknown, err
	}
	addSuccessPluginServerCall(metricsGetOperationStatus)
	switch st.GetStatus() {
	case plugin.OperationStatusType_DONE:
		return OperationStatusDone, nil
	case plugin.OperationStatusType_RUNNING:
		return OperationStatusRunning, nil
	case plugin.OperationStatusType_FAILED:
		return OperationStatusFailed, nil
	case plugin.OperationStatusType_UNKNOWN:
		return OperationStatusUnknown, nil
	}
	return OperationStatusUnknown, fmt.Errorf("no match status of the operation. status: %s", st.GetStatus())
}

func (p *pluginOperator) GetClusterVersion(ctx context.Context, obj opsv1.ClusterVersion, cluster opsv1.Cluster) (*ClusterVersion, error) {
	c, closer, err := p.newFunc(obj)
	if err != nil {
		return nil, err
	}
	defer closer()
	req := &plugin.GetVersionRequest{
		ClusterID: cluster.ID,
	}
	res, err := c.GetVersion(ctx, req)
	if err != nil {
		addFailedPluginServerCall(metricsGetClusterVersion)
		return nil, err
	}
	addSuccessPluginServerCall(metricsGetClusterVersion)
	cv := &ClusterVersion{
		Master: MasterVersion{
			ClusterID: res.Master.ClusterID,
			Version:   res.Master.Version,
		},
	}
	cv.NodePools = make([]NodePoolVersion, len(res.NodePools))
	for i, np := range res.NodePools {
		cv.NodePools[i] = NodePoolVersion{
			NodePoolID: np.NodePoolID,
			Version:    np.Version,
		}
	}
	return cv, nil
}

func (p *pluginOperator) ServiceIn(ctx context.Context, obj opsv1.ClusterVersion, cluster opsv1.Cluster) (*OperationResult, error) {
	c, closer, err := p.newFunc(obj)
	if err != nil {
		return nil, err
	}
	defer closer()
	req := &plugin.ServiceInRequest{
		ClusterID: cluster.ID,
	}
	ops, err := c.ServiceIn(ctx, req)
	if err != nil {
		addFailedPluginServerCall(metricsServiceIn)
		return nil, err
	}
	addSuccessPluginServerCall(metricsServiceIn)
	return &OperationResult{
		OperationID:   ops.OperationID,
		OperationType: ops.Type,
	}, nil
}

func (p *pluginOperator) ServiceOut(ctx context.Context, obj opsv1.ClusterVersion, cluster opsv1.Cluster) (*OperationResult, error) {
	c, closer, err := p.newFunc(obj)
	if err != nil {
		return nil, err
	}
	defer closer()
	req := &plugin.ServiceOutRequest{
		ClusterID: cluster.ID,
	}
	ops, err := c.ServiceOut(ctx, req)
	if err != nil {
		addFailedPluginServerCall(metricsServiceOut)
		return nil, err
	}
	addSuccessPluginServerCall(metricsServiceOut)
	return &OperationResult{
		OperationID:   ops.OperationID,
		OperationType: ops.Type,
	}, nil
}

func (p *pluginOperator) UpgradeMaster(ctx context.Context, obj opsv1.ClusterVersion, cluster opsv1.Cluster) (*OperationResult, error) {
	c, closer, err := p.newFunc(obj)
	if err != nil {
		return nil, err
	}
	defer closer()
	req := &plugin.MasterVersion{
		ClusterID: cluster.ID,
		Version:   cluster.Version,
	}
	res, err := c.UpgradeMaster(ctx, req)
	if err != nil {
		addFailedPluginServerCall(metricsUpgradeMaster)
		return nil, err
	}
	addSuccessPluginServerCall(metricsUpgradeMaster)
	return &OperationResult{
		OperationID:   res.OperationID,
		OperationType: res.Type,
	}, nil
}

func (p *pluginOperator) UpgradeNodePool(ctx context.Context, obj opsv1.ClusterVersion, cluster opsv1.Cluster, nodePoolID string) (*OperationResult, error) {
	c, closer, err := p.newFunc(obj)
	if err != nil {
		return nil, err
	}
	defer closer()
	req := &plugin.NodePoolVersion{
		ClusterID:  cluster.ID,
		NodePoolID: nodePoolID,
		Version:    cluster.Version,
	}
	res, err := c.UpgradeNodePool(ctx, req)
	if err != nil {
		addFailedPluginServerCall(metricsUpgradeNodePool)
		return nil, err
	}
	addSuccessPluginServerCall(metricsUpgradeNodePool)
	return &OperationResult{
		OperationID:   res.OperationID,
		OperationType: res.Type,
	}, nil
}
