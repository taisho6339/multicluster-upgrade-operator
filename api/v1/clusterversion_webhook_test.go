package v1_test

import (
	"errors"
	"fmt"
	. "github.com/onsi/gomega"
	v1 "github.com/taisho6339/multicluster-upgrade-operator/api/v1"
	"testing"
)

func makeClusterVersion(namespace, name string) *v1.ClusterVersion {
	mc := &v1.ClusterVersion{}
	mc.Name = name
	mc.Namespace = namespace
	mc.Spec.RequiredAvailableCount = 1
	mc.Spec.Clusters = []v1.Cluster{
		{
			ID:      fmt.Sprintf("%s/cluster-1", name),
			Version: "1.16.13-gke.404",
		},
		{
			ID:      fmt.Sprintf("%s/cluster-2", name),
			Version: "1.16.13-gke.404",
		},
	}
	return mc
}

func makeClusterVersionWithOperation(namespace, name string) *v1.ClusterVersion {
	mc := makeClusterVersion(namespace, name)
	mc.Status.OperationID = "dummy-id"
	mc.Status.OperationType = "DUMMY_OPERATION"
	mc.Status.ClusterID = mc.Spec.Clusters[0].ID
	return mc
}

func makeClusterVersionWithDuplicate(namespace, name string) *v1.ClusterVersion {
	mc := makeClusterVersion(namespace, name)
	mc.Spec.Clusters[1] = mc.Spec.Clusters[0]
	return mc
}

func TestClusterVersion_ValidateCreate(t *testing.T) {
	tc := []struct {
		name     string
		in       *v1.ClusterVersion
		expected error
	}{
		{
			name:     "work as success",
			in:       makeClusterVersion("default", "success-clusters"),
			expected: nil,
		},
		{
			name:     "work as duplicate error",
			in:       makeClusterVersionWithDuplicate("default", "duplicate-clusters"),
			expected: errors.New("ClusterVersion.multicluster-ops.io \"duplicate-clusters\" is invalid: spec.clusters: Invalid value: \"duplicate-clusters/cluster-1\": duplicate cluster id"),
		},
	}
	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			ret := c.in.ValidateCreate()
			if c.expected == nil {
				g.Expect(ret).Should(BeNil())
			} else {
				g.Expect(ret).ShouldNot(BeNil())
				g.Expect(ret.Error()).Should(Equal(c.expected.Error()))
			}
		})
	}
}

func TestClusterVersion_ValidateUpdate(t *testing.T) {
	tc := []struct {
		name     string
		in       *v1.ClusterVersion
		expected error
	}{
		{
			name:     "work as success",
			in:       makeClusterVersion("default", "success-clusters"),
			expected: nil,
		},
		{
			name:     "work as duplicate error",
			in:       makeClusterVersionWithDuplicate("default", "duplicate-clusters"),
			expected: errors.New("ClusterVersion.multicluster-ops.io \"duplicate-clusters\" is invalid: spec.clusters: Invalid value: \"duplicate-clusters/cluster-1\": duplicate cluster id"),
		},
	}
	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			ret := c.in.ValidateUpdate(nil)
			if c.expected == nil {
				g.Expect(ret).Should(BeNil())
			} else {
				g.Expect(ret).ShouldNot(BeNil())
				g.Expect(ret.Error()).Should(Equal(c.expected.Error()))
			}
		})
	}
}

func TestClusterVersion_ValidateDelete(t *testing.T) {
	tc := []struct {
		name     string
		in       *v1.ClusterVersion
		expected error
	}{
		{
			name:     "work as success",
			in:       makeClusterVersion("default", "success-clusters"),
			expected: nil,
		},
		{
			name:     "work as duplicate error",
			in:       makeClusterVersionWithOperation("default", "duplicate-clusters"),
			expected: errors.New("mustn't delete while operation is running"),
		},
	}
	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			g := NewGomegaWithT(t)
			ret := c.in.ValidateDelete()
			if c.expected == nil {
				g.Expect(ret).Should(BeNil())
			} else {
				g.Expect(ret).ShouldNot(BeNil())
				g.Expect(ret.Error()).Should(Equal(c.expected.Error()))
			}
		})
	}
}
