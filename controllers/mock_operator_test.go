package controllers

import (
	"context"
	"errors"
	opsv1 "github.com/taisho6339/multicluster-upgrade-operator/api/v1"
	. "github.com/taisho6339/multicluster-upgrade-operator/pkg/ops"
	"k8s.io/apimachinery/pkg/util/uuid"
	"sync"
	"time"
)

const (
	operationWaitTime = time.Millisecond * 10
)

type mockOperator struct {
	clusterVersionMap  map[string]*ClusterVersion
	clusterStatusMap   map[string]*ClusterStatus
	operationStatusMap map[string]OperationStatus
	executedOperations map[string][]*OperationResult

	lock sync.RWMutex
}

var _ Operator = &mockOperator{}

func newMockOperator() *mockOperator {
	return &mockOperator{
		clusterVersionMap:  map[string]*ClusterVersion{},
		clusterStatusMap:   map[string]*ClusterStatus{},
		operationStatusMap: map[string]OperationStatus{},
		executedOperations: map[string][]*OperationResult{},
	}
}

func (m *mockOperator) AddClusterVersion(clusters ...*ClusterVersion) {
	m.lock.Lock()
	defer m.lock.Unlock()
	for _, cl := range clusters {
		m.clusterVersionMap[cl.Master.ClusterID] = cl
		m.clusterStatusMap[cl.Master.ClusterID] = &ClusterStatus{
			Type:      ClusterStatusServiceIn,
			Available: true,
		}
	}
}

func (m *mockOperator) LastExecutedOperationIs(operationType string, resourceName string) func() bool {
	return func() bool {
		m.lock.RLock()
		defer m.lock.RUnlock()

		index := len(m.executedOperations[resourceName]) - 1
		if index < 0 {
			return false
		}
		result := m.executedOperations[resourceName][index]
		return result.OperationType == operationType
	}
}

func (m *mockOperator) HasExecutedAt(at int, operationType string, resourceName string) func() bool {
	return func() bool {
		m.lock.RLock()
		defer m.lock.RUnlock()

		if at < 0 {
			return false
		}
		if (len(m.executedOperations[resourceName]) - 1) < at {
			return false
		}
		result := m.executedOperations[resourceName][at]
		return result.OperationType == operationType
	}
}

func (m *mockOperator) HasServiceIn(clusterID string) func() bool {
	return func() bool {
		m.lock.RLock()
		defer m.lock.RUnlock()

		v := m.clusterStatusMap[clusterID]
		return v != nil && v.Type == ClusterStatusServiceIn && v.Available
	}
}

func (m *mockOperator) HasServiceOut(clusterID string) func() bool {
	return func() bool {
		m.lock.RLock()
		defer m.lock.RUnlock()

		v := m.clusterStatusMap[clusterID]
		return v != nil && v.Type == ClusterStatusServiceOut && v.Available
	}
}

func (m *mockOperator) ChangeAvailability(clusterID string, available bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.clusterStatusMap[clusterID].Available = available
}

func (m *mockOperator) GetClusterVersion(_ context.Context, _ opsv1.ClusterVersion, cluster opsv1.Cluster) (*ClusterVersion, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	v, ok := m.clusterVersionMap[cluster.ID]
	if !ok {
		return nil, errors.New("not found")
	}
	return v, nil
}

func (m *mockOperator) GetClusterStatus(_ context.Context, _ opsv1.ClusterVersion, cluster opsv1.Cluster) (*ClusterStatus, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	v, ok := m.clusterStatusMap[cluster.ID]
	if !ok {
		return &ClusterStatus{
			Type:      ClusterStatusServiceUnknown,
			Available: false,
		}, errors.New("not found")
	}
	return v, nil
}

func (m *mockOperator) GetOperationStatus(_ context.Context, obj opsv1.ClusterVersion) (OperationStatus, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	v, ok := m.operationStatusMap[obj.Status.OperationID]
	if !ok {
		return OperationStatusUnknown, errors.New("not found")
	}
	return v, nil
}

func (m *mockOperator) ServiceIn(_ context.Context, obj opsv1.ClusterVersion, cluster opsv1.Cluster) (*OperationResult, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	id := string(uuid.NewUUID())
	m.operationStatusMap[id] = OperationStatusRunning

	time.AfterFunc(operationWaitTime, func() {
		m.lock.Lock()
		defer m.lock.Unlock()

		m.operationStatusMap[id] = OperationStatusDone
		m.clusterStatusMap[cluster.ID] = &ClusterStatus{
			Type:      ClusterStatusServiceIn,
			Available: m.clusterStatusMap[cluster.ID].Available,
		}
	})

	or := &OperationResult{
		OperationID:   id,
		OperationType: "SERVICE_IN",
	}
	results := m.executedOperations[obj.Name]
	m.executedOperations[obj.Name] = append(results, or)
	return or, nil
}

func (m *mockOperator) ServiceOut(_ context.Context, obj opsv1.ClusterVersion, cluster opsv1.Cluster) (*OperationResult, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	id := string(uuid.NewUUID())
	m.operationStatusMap[id] = OperationStatusRunning

	time.AfterFunc(operationWaitTime, func() {
		m.lock.Lock()
		defer m.lock.Unlock()

		m.operationStatusMap[id] = OperationStatusDone
		m.clusterStatusMap[cluster.ID] = &ClusterStatus{
			Type:      ClusterStatusServiceOut,
			Available: m.clusterStatusMap[cluster.ID].Available,
		}
	})

	or := &OperationResult{
		OperationID:   id,
		OperationType: "SERVICE_OUT",
	}
	results := m.executedOperations[obj.Name]
	m.executedOperations[obj.Name] = append(results, or)
	return or, nil
}

func (m *mockOperator) UpgradeMaster(_ context.Context, obj opsv1.ClusterVersion, cluster opsv1.Cluster) (*OperationResult, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	id := string(uuid.NewUUID())
	m.operationStatusMap[id] = OperationStatusRunning

	time.AfterFunc(operationWaitTime, func() {
		m.lock.Lock()
		defer m.lock.Unlock()

		current, ok := m.clusterVersionMap[cluster.ID]
		if !ok {
			return
		}

		var desiredVersion string
		for _, cl := range obj.Spec.Clusters {
			if cl.ID == cluster.ID {
				desiredVersion = cl.Version
			}
		}
		current.Master.Version = desiredVersion
		m.operationStatusMap[id] = OperationStatusDone
	})

	or := &OperationResult{
		OperationID:   id,
		OperationType: "UPGRADE_MASTER",
	}
	results := m.executedOperations[obj.Name]
	m.executedOperations[obj.Name] = append(results, or)
	return or, nil
}

func (m *mockOperator) UpgradeNodePool(_ context.Context, obj opsv1.ClusterVersion, cluster opsv1.Cluster, nodePoolID string) (*OperationResult, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	id := string(uuid.NewUUID())
	m.operationStatusMap[id] = OperationStatusRunning

	time.AfterFunc(operationWaitTime, func() {
		m.lock.Lock()
		defer m.lock.Unlock()

		current, ok := m.clusterVersionMap[cluster.ID]
		if !ok {
			return
		}
		var desiredVersion string
		for _, cl := range obj.Spec.Clusters {
			if cl.ID == cluster.ID {
				desiredVersion = cl.Version
			}
		}
		for i, np := range current.NodePools {
			if np.NodePoolID == nodePoolID {
				current.NodePools[i].Version = desiredVersion
			}
		}
		m.operationStatusMap[id] = OperationStatusDone
	})

	or := &OperationResult{
		OperationID:   id,
		OperationType: "UPGRADE_NODE_POOL",
	}
	results := m.executedOperations[obj.Name]
	m.executedOperations[obj.Name] = append(results, or)
	return or, nil
}
