package controllers

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	opsv1 "github.com/taisho6339/multicluster-upgrade-operator/api/v1"
	"github.com/taisho6339/multicluster-upgrade-operator/pkg/ops"
)

func makeClusterVersion(namespace, name string) *opsv1.ClusterVersion {
	mc := &opsv1.ClusterVersion{}
	mc.Name = name
	mc.Namespace = namespace
	mc.Spec.RequiredAvailableCount = 1
	mc.Spec.Clusters = []opsv1.Cluster{
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

func makeCurrentResourceDifferentState(mc opsv1.ClusterVersion) []*ops.ClusterVersion {
	ret := make([]*ops.ClusterVersion, len(mc.Spec.Clusters))
	for i, cl := range mc.Spec.Clusters {
		cv := &ops.ClusterVersion{
			Master: ops.MasterVersion{
				ClusterID: cl.ID,
				Version:   "1.16.13-gke.different",
			},
		}
		np := []ops.NodePoolVersion{
			{
				NodePoolID: fmt.Sprintf("%s/node-pool-1", cl.ID),
				Version:    "1.16.13-gke.different",
			},
			{
				NodePoolID: fmt.Sprintf("%s/node-pool-2", cl.ID),
				Version:    "1.16.13-gke.different",
			},
		}
		cv.NodePools = np
		ret[i] = cv
	}
	return ret
}

var _ = Describe("Multi Cluster Reconciler", func() {
	ctx := context.Background()

	Context("success cases", func() {
		Describe("cases about controller flow", func() {
			It("work as expected in flow", func() {
				var mcName = "success-cases-mc-1"
				var mcNamespace = "default"
				var checkOperationAt = 0
				mc := makeClusterVersion(mcNamespace, mcName)

				By("[prepare] mock operation")
				operator.AddClusterVersion(makeCurrentResourceDifferentState(*mc)...)

				By("[prepare] create a multicluster resource")
				err := k8sClient.Create(ctx, mc)
				Expect(err).ToNot(HaveOccurred())

				By("[check] start service out for first cluster")
				Eventually(operator.HasExecutedAt(checkOperationAt, "SERVICE_OUT", mcName)).Should(Equal(true))
				checkOperationAt += 1

				By("[check] start upgrade master for first cluster")
				Eventually(operator.HasExecutedAt(checkOperationAt, "UPGRADE_MASTER", mcName)).Should(Equal(true))
				checkOperationAt += 1
				Expect(operator.HasServiceOut(mc.Spec.Clusters[0].ID)()).Should(Equal(true))

				By("[check] start upgrade node pool 1 for first cluster")
				Eventually(operator.HasExecutedAt(checkOperationAt, "UPGRADE_NODE_POOL", mcName)).Should(Equal(true))
				checkOperationAt += 1
				Expect(operator.HasServiceOut(mc.Spec.Clusters[0].ID)()).Should(Equal(true))

				By("[check] start upgrade node pool 2 for first cluster")
				Eventually(operator.HasExecutedAt(checkOperationAt, "UPGRADE_NODE_POOL", mcName)).Should(Equal(true))
				checkOperationAt += 1
				Expect(operator.HasServiceOut(mc.Spec.Clusters[0].ID)()).Should(Equal(true))

				By("[check] start servicein for first cluster")
				Eventually(operator.HasExecutedAt(checkOperationAt, "SERVICE_IN", mcName)).Should(Equal(true))
				checkOperationAt += 1

				By("[check] complete servicein for first cluster")
				Eventually(operator.HasServiceIn(mc.Spec.Clusters[0].ID)).Should(Equal(true))

				By("[check] start serviceout for second cluster")
				Eventually(operator.HasExecutedAt(checkOperationAt, "SERVICE_OUT", mcName)).Should(Equal(true))
				checkOperationAt += 1

				By("[check] start upgrade master for second cluster")
				Eventually(operator.HasExecutedAt(checkOperationAt, "UPGRADE_MASTER", mcName)).Should(Equal(true))
				checkOperationAt += 1
				Expect(operator.HasServiceOut(mc.Spec.Clusters[1].ID)()).Should(Equal(true))

				By("[check] start upgrade node pool 1 for second cluster")
				Eventually(operator.HasExecutedAt(checkOperationAt, "UPGRADE_NODE_POOL", mcName)).Should(Equal(true))
				checkOperationAt += 1
				Expect(operator.HasServiceOut(mc.Spec.Clusters[1].ID)()).Should(Equal(true))

				By("[check] start upgrade node pool 2 for second cluster")
				Eventually(operator.HasExecutedAt(checkOperationAt, "UPGRADE_NODE_POOL", mcName)).Should(Equal(true))
				checkOperationAt += 1
				Expect(operator.HasServiceOut(mc.Spec.Clusters[1].ID)()).Should(Equal(true))

				By("[check] start servicein for second cluster")
				Eventually(operator.HasExecutedAt(checkOperationAt, "SERVICE_IN", mcName)).Should(Equal(true))

				By("[check] complete servicein for second cluster")
				Eventually(operator.HasServiceIn(mc.Spec.Clusters[1].ID)).Should(Equal(true))
			})
		})
	})

	Context("exception cases", func() {
		It("when the cluster is unavailable, wouldn't service in", func() {
			var mcName = "test-clusters-exception-1"
			var mcNamespace = "default"
			mc := makeClusterVersion(mcNamespace, mcName)

			By("[prepare] mock operation")
			operator.AddClusterVersion(makeCurrentResourceDifferentState(*mc)...)

			By("[prepare] create a multicluster resource")
			err := k8sClient.Create(ctx, mc)
			Expect(err).ToNot(HaveOccurred())

			By("[prepare] make first cluster be unavailable")
			operator.ChangeAvailability(mc.Spec.Clusters[0].ID, false)

			By("[prepare] wait for complete upgrade")
			Eventually(operator.LastExecutedOperationIs("UPGRADE_NODE_POOL", mcName)).Should(Equal(true))

			By("[check] stop operation when cluster is unavailable")
			Consistently(operator.LastExecutedOperationIs("UPGRADE_NODE_POOL", mcName)).Should(Equal(true))
		})

		It("available clusters less than required available count, wouldn't service out", func() {
			var mcName = "test-clusters-exception-3"
			var mcNamespace = "default"
			mc := makeClusterVersion(mcNamespace, mcName)

			By("[prepare] mock operation")
			operator.AddClusterVersion(makeCurrentResourceDifferentState(*mc)...)

			By("[prepare] make one of the clusters be unavailable")
			operator.ChangeAvailability(mc.Spec.Clusters[1].ID, false)

			By("[prepare] create a multicluster resource")
			err := k8sClient.Create(ctx, mc)
			Expect(err).ToNot(HaveOccurred())

			By("[check] operations won't start")
			Consistently(func() bool {
				return len(operator.executedOperations[mc.Name]) == 0
			}).Should(Equal(true))
		})
	})
})
