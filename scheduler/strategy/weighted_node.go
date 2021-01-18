package strategy

import (
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
)

// WeightedNode represents a node in the cluster with a given weight, typically used for sorting
// purposes.
type weightedNode struct {
	Node *node.Node
	// Weight is the inherent value of this node.
	Weight int64
	// Containers is the number of containers running on this node
	Containers int
}

type weightedNodeList []*weightedNode

func (n weightedNodeList) Len() int {
	return len(n)
}

func (n weightedNodeList) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

func (n weightedNodeList) Less(i, j int) bool {
	var (
		ip = n[i]
		jp = n[j]
	)

	// If the nodes have the same weight sort them out by number of containers.
	if ip.Weight != jp.Weight {
		return ip.Weight < jp.Weight
	}
	if ip.Containers != jp.Containers {
		return ip.Containers < jp.Containers
	}

	// If there are as many containers, sort by free CPUs and free RAMs
	ipFreeCpus := ip.Node.TotalCpus - ip.Node.UsedCpus
	jpFreeCpus := jp.Node.TotalCpus - jp.Node.UsedCpus
	if ipFreeCpus != jpFreeCpus {
		return ipFreeCpus > jpFreeCpus
	}
	return ip.Node.TotalMemory-ip.Node.UsedMemory > jp.Node.TotalMemory-jp.Node.UsedMemory
}

func weighNodes(config *cluster.ContainerConfig, nodes []*node.Node, healthinessFactor int64) (weightedNodeList, error) {
	weightedNodes := weightedNodeList{}

	for _, node := range nodes {
		nodeMemory := node.TotalMemory
		nodeCpus := node.TotalCpus

		// Skip nodes that are smaller than the requested resources.
		if nodeMemory < int64(config.HostConfig.Memory) || nodeCpus < config.HostConfig.CPUShares {
			continue
		}

		var (
			cpuScore    int64 = 100
			memoryScore int64 = 100
		)

		if config.HostConfig.CPUShares > 0 {
			cpuScore = (node.UsedCpus + config.HostConfig.CPUShares) * 100 / nodeCpus
		}
		if config.HostConfig.Memory > 0 {
			memoryScore = (node.UsedMemory + config.HostConfig.Memory) * 100 / nodeMemory
		}

		if cpuScore <= 100 && memoryScore <= 100 {
			weightedNodes = append(weightedNodes, &weightedNode{Node: node,
				Weight:     cpuScore + memoryScore + healthinessFactor*node.HealthIndicator,
				Containers: len(node.Containers)})
		}
	}

	if len(weightedNodes) == 0 {
		return nil, ErrNoResourcesAvailable
	}

	return weightedNodes, nil
}

func weightNodesByCapacity(config *cluster.ContainerConfig, nodes []*node.Node, healthinessFactor int64) (weightedNodeList, error) {
	weightedNodes := weightedNodeList{}

	weight := JenkinsWeight(config)
	for _, node := range nodes {
		totalWeight := weight
		for _, container := range node.Containers {
			totalWeight += JenkinsWeight(container.Config)
		}
		// Assign lower score for highest 'speed', e.g. estimated available core
		// This score is not normalized, to properly distribute according to capacity of each node
		cpuScore := -(node.TotalCpus - node.UsedCpus) / int64(totalWeight)

		if cpuScore != 0 {
			weightedNodes = append(weightedNodes, &weightedNode{Node: node,
				Weight:     cpuScore + healthinessFactor*node.HealthIndicator,
				Containers: totalWeight})
		}
	}

	if len(weightedNodes) == 0 {
		return nil, ErrNoResourcesAvailable
	}

	return weightedNodes, nil
}
