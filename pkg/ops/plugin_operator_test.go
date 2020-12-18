package ops

import (
	"context"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	"github.com/taisho6339/multicluster-upgrade-operator-proto/go/plugin"
	v1 "github.com/taisho6339/multicluster-upgrade-operator/api/v1"
	"testing"
)

func makeClusterVersionResource() *v1.ClusterVersion {
	return &v1.ClusterVersion{
		Spec: v1.ClusterVersionSpec{
			Clusters: []v1.Cluster{
				{
					ID:      "test-cluster",
					Version: "1.0.0",
				},
			},
			OpsEndpoint: v1.OpsEndpoint{
				Endpoint: "test.example.com",
				Insecure: true,
			},
		},
	}
}

func TestPluginOperator_GetClusterStatus(t *testing.T) {
	testCases := []struct {
		name           string
		ret            *plugin.ClusterStatus
		expected       *ClusterStatus
		expectedHasErr bool
	}{
		{
			name: "ret service in",
			ret: &plugin.ClusterStatus{
				Status:      plugin.ClusterStatusType_STATUS_SERVICE_IN,
				IsAvailable: true,
			},
			expected: &ClusterStatus{
				Type:      ClusterStatusServiceIn,
				Available: true,
			},
		},
		{
			name: "ret service out",
			ret: &plugin.ClusterStatus{
				Status:      plugin.ClusterStatusType_STATUS_SERVICE_OUT,
				IsAvailable: true,
			},
			expected: &ClusterStatus{
				Type:      ClusterStatusServiceOut,
				Available: true,
			},
		},
		{
			name: "ret unknown",
			ret: &plugin.ClusterStatus{
				Status:      -1, // illegal value
				IsAvailable: false,
			},
			expected: &ClusterStatus{
				Type:      ClusterStatusServiceUnknown,
				Available: false,
			},
			expectedHasErr: true,
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var (
				g   = NewGomegaWithT(t)
				ctx = context.Background()
				obj = makeClusterVersionResource()
			)
			c := plugin.NewMockClusterClient(ctrl)
			operator := NewPluginOperator(func(obj v1.ClusterVersion) (plugin.ClusterClient, func(), error) {
				return c, func() {}, nil
			})
			req := &plugin.GetClusterStatusRequest{
				ClusterID: obj.Spec.Clusters[0].ID,
			}
			c.EXPECT().GetClusterStatus(gomock.Any(), gomock.Eq(req)).Return(testCase.ret, nil).Times(1)

			status, err := operator.GetClusterStatus(ctx, *obj, obj.Spec.Clusters[0])
			if testCase.expectedHasErr {
				g.Expect(err).ShouldNot(BeNil())
			} else {
				g.Expect(err).Should(BeNil())
			}
			g.Expect(testCase.expected).Should(Equal(status))
		})
	}
}

func TestPluginOperator_GetOperationStatus(t *testing.T) {
	testCases := []struct {
		name           string
		ret            *plugin.OperationStatus
		expected       OperationStatus
		expectedHasErr bool
	}{
		{
			name: "ret done",
			ret: &plugin.OperationStatus{
				Status: plugin.OperationStatusType_DONE,
			},
			expected: OperationStatusDone,
		},
		{
			name: "ret running",
			ret: &plugin.OperationStatus{
				Status: plugin.OperationStatusType_RUNNING,
			},
			expected: OperationStatusRunning,
		},
		{
			name: "ret failed",
			ret: &plugin.OperationStatus{
				Status: plugin.OperationStatusType_FAILED,
			},
			expected: OperationStatusFailed,
		},
		{
			name: "ret unknown",
			ret: &plugin.OperationStatus{
				Status: plugin.OperationStatusType_UNKNOWN,
			},
			expected: OperationStatusUnknown,
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var (
				g   = NewGomegaWithT(t)
				ctx = context.Background()
				obj = makeClusterVersionResource()
			)
			c := plugin.NewMockClusterClient(ctrl)
			operator := NewPluginOperator(func(obj v1.ClusterVersion) (plugin.ClusterClient, func(), error) {
				return c, func() {}, nil
			})
			obj.Status.ClusterID = obj.Spec.Clusters[0].ID
			obj.Status.OperationID = "dummy"
			obj.Status.OperationType = "dummy"
			req := &plugin.GetOperationStatusRequest{
				ClusterID:   obj.Spec.Clusters[0].ID,
				OperationID: obj.Status.OperationID,
				Type:        obj.Status.OperationType,
			}
			c.EXPECT().GetOperationStatus(gomock.Any(), gomock.Eq(req)).Return(testCase.ret, nil).Times(1)

			status, err := operator.GetOperationStatus(ctx, *obj)
			if testCase.expectedHasErr {
				g.Expect(err).ShouldNot(BeNil())
			} else {
				g.Expect(err).Should(BeNil())
			}
			g.Expect(testCase.expected).Should(Equal(status))
		})
	}
}

func TestPluginOperator_GetClusterVersion(t *testing.T) {
	testCases := []struct {
		name           string
		ret            *plugin.ClusterVersion
		expected       *ClusterVersion
		expectedHasErr bool
	}{
		{
			name: "work as expected",
			ret: &plugin.ClusterVersion{
				Master: &plugin.MasterVersion{
					ClusterID: "test-cluster",
					Version:   "0.1.0",
				},
				NodePools: []*plugin.NodePoolVersion{
					{
						ClusterID:  "test-cluster",
						NodePoolID: "test-cluster-node-pool-1",
						Version:    "0.1.0",
					},
					{
						ClusterID:  "test-cluster",
						NodePoolID: "test-cluster-node-pool-2",
						Version:    "0.1.0",
					},
				},
			},
			expected: &ClusterVersion{
				Master: MasterVersion{
					ClusterID: "test-cluster",
					Version:   "0.1.0",
				},
				NodePools: []NodePoolVersion{
					{
						NodePoolID: "test-cluster-node-pool-1",
						Version:    "0.1.0",
					},
					{
						NodePoolID: "test-cluster-node-pool-2",
						Version:    "0.1.0",
					},
				},
			},
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var (
				g   = NewGomegaWithT(t)
				ctx = context.Background()
				obj = makeClusterVersionResource()
			)
			c := plugin.NewMockClusterClient(ctrl)
			operator := NewPluginOperator(func(obj v1.ClusterVersion) (plugin.ClusterClient, func(), error) {
				return c, func() {}, nil
			})
			req := &plugin.GetVersionRequest{
				ClusterID: obj.Spec.Clusters[0].ID,
			}
			c.EXPECT().GetVersion(gomock.Any(), gomock.Eq(req)).Return(testCase.ret, nil).Times(1)

			version, err := operator.GetClusterVersion(ctx, *obj, obj.Spec.Clusters[0])
			if testCase.expectedHasErr {
				g.Expect(err).ShouldNot(BeNil())
			} else {
				g.Expect(err).Should(BeNil())
			}
			g.Expect(testCase.expected).Should(Equal(version))
		})
	}
}

func TestPluginOperator_ServiceIn(t *testing.T) {
	testCases := []struct {
		name           string
		ret            *plugin.Operation
		expected       *OperationResult
		expectedHasErr bool
	}{
		{
			name: "work as expected",
			ret: &plugin.Operation{
				OperationID: "dummy_operation",
				Type:        "SERVICE_IN",
			},
			expected: &OperationResult{
				OperationID:   "dummy_operation",
				OperationType: "SERVICE_IN",
			},
			expectedHasErr: false,
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var (
				g   = NewGomegaWithT(t)
				ctx = context.Background()
				obj = makeClusterVersionResource()
			)
			c := plugin.NewMockClusterClient(ctrl)
			operator := NewPluginOperator(func(obj v1.ClusterVersion) (plugin.ClusterClient, func(), error) {
				return c, func() {}, nil
			})
			req := &plugin.ServiceInRequest{
				ClusterID: obj.Spec.Clusters[0].ID,
			}
			c.EXPECT().ServiceIn(gomock.Any(), gomock.Eq(req)).Return(testCase.ret, nil).Times(1)

			operation, err := operator.ServiceIn(ctx, *obj, obj.Spec.Clusters[0])
			if testCase.expectedHasErr {
				g.Expect(err).ShouldNot(BeNil())
			} else {
				g.Expect(err).Should(BeNil())
			}
			g.Expect(testCase.expected).Should(Equal(operation))
		})
	}
}

func TestPluginOperator_ServiceOut(t *testing.T) {
	testCases := []struct {
		name           string
		ret            *plugin.Operation
		expected       *OperationResult
		expectedHasErr bool
	}{
		{
			name: "work as expected",
			ret: &plugin.Operation{
				OperationID: "dummy_operation",
				Type:        "SERVICE_OUT",
			},
			expected: &OperationResult{
				OperationID:   "dummy_operation",
				OperationType: "SERVICE_OUT",
			},
			expectedHasErr: false,
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var (
				g   = NewGomegaWithT(t)
				ctx = context.Background()
				obj = makeClusterVersionResource()
			)
			c := plugin.NewMockClusterClient(ctrl)
			operator := NewPluginOperator(func(obj v1.ClusterVersion) (plugin.ClusterClient, func(), error) {
				return c, func() {}, nil
			})
			req := &plugin.ServiceOutRequest{
				ClusterID: obj.Spec.Clusters[0].ID,
			}
			c.EXPECT().ServiceOut(gomock.Any(), gomock.Eq(req)).Return(testCase.ret, nil).Times(1)

			operation, err := operator.ServiceOut(ctx, *obj, obj.Spec.Clusters[0])
			if testCase.expectedHasErr {
				g.Expect(err).ShouldNot(BeNil())
			} else {
				g.Expect(err).Should(BeNil())
			}
			g.Expect(testCase.expected).Should(Equal(operation))
		})
	}
}

func TestPluginOperator_UpgradeMaster(t *testing.T) {
	testCases := []struct {
		name           string
		ret            *plugin.Operation
		expected       *OperationResult
		expectedHasErr bool
	}{
		{
			name: "work as expected",
			ret: &plugin.Operation{
				OperationID: "dummy_operation",
				Type:        "UPGRADE_MASTER",
			},
			expected: &OperationResult{
				OperationID:   "dummy_operation",
				OperationType: "UPGRADE_MASTER",
			},
			expectedHasErr: false,
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var (
				g   = NewGomegaWithT(t)
				ctx = context.Background()
				obj = makeClusterVersionResource()
			)
			c := plugin.NewMockClusterClient(ctrl)
			operator := NewPluginOperator(func(obj v1.ClusterVersion) (plugin.ClusterClient, func(), error) {
				return c, func() {}, nil
			})
			req := &plugin.MasterVersion{
				ClusterID: obj.Spec.Clusters[0].ID,
				Version:   "1.0.0",
			}
			c.EXPECT().UpgradeMaster(gomock.Any(), gomock.Eq(req)).Return(testCase.ret, nil).Times(1)

			operation, err := operator.UpgradeMaster(ctx, *obj, obj.Spec.Clusters[0])
			if testCase.expectedHasErr {
				g.Expect(err).ShouldNot(BeNil())
			} else {
				g.Expect(err).Should(BeNil())
			}
			g.Expect(testCase.expected).Should(Equal(operation))
		})
	}
}

func TestPluginOperator_UpgradeNodePool(t *testing.T) {
	testCases := []struct {
		name           string
		ret            *plugin.Operation
		expected       *OperationResult
		expectedHasErr bool
	}{
		{
			name: "work as expected",
			ret: &plugin.Operation{
				OperationID: "dummy_operation",
				Type:        "UPGRADE_NODE_POOL",
			},
			expected: &OperationResult{
				OperationID:   "dummy_operation",
				OperationType: "UPGRADE_NODE_POOL",
			},
			expectedHasErr: false,
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var (
				g   = NewGomegaWithT(t)
				ctx = context.Background()
				obj = makeClusterVersionResource()
			)
			c := plugin.NewMockClusterClient(ctrl)
			operator := NewPluginOperator(func(obj v1.ClusterVersion) (plugin.ClusterClient, func(), error) {
				return c, func() {}, nil
			})
			req := &plugin.NodePoolVersion{
				ClusterID:  obj.Spec.Clusters[0].ID,
				NodePoolID: "nodepool-1",
				Version:    "1.0.0",
			}
			c.EXPECT().UpgradeNodePool(gomock.Any(), gomock.Eq(req)).Return(testCase.ret, nil).Times(1)

			operation, err := operator.UpgradeNodePool(ctx, *obj, obj.Spec.Clusters[0], "nodepool-1")
			if testCase.expectedHasErr {
				g.Expect(err).ShouldNot(BeNil())
			} else {
				g.Expect(err).Should(BeNil())
			}
			g.Expect(testCase.expected).Should(Equal(operation))
		})
	}
}
