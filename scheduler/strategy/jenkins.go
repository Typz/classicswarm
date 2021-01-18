package strategy

import (
	"sort"
	"strconv"

	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
)

// JenkinsPlacementStrategy places the container on the node with the most available capacity
type JenkinsPlacementStrategy struct {
}

// Initialize a RandomPlacementStrategy.
func (p *JenkinsPlacementStrategy) Initialize() error {
	return nil
}

// Name returns the name of the strategy.
func (p *JenkinsPlacementStrategy) Name() string {
	return "jenkins"
}

// JenkinsWeight returns the relative weight of jenkins slave container. 0 if this is not a jenkins
// slave or if it uses memory or CPU reservation ; otherwise the value of the `weight` label, with
// a default value of 1.
func JenkinsWeight(c *cluster.ContainerConfig) int {
	_, ok := c.Labels["com.nirima.jenkins.plugins.docker.JenkinsId"]
	if !ok || c.HostConfig.CPUShares > 0 || c.HostConfig.Memory > 0 {
		return 0
	}
	weightStr, ok := c.Labels[cluster.SwarmLabelNamespace+".weight"]
	if !ok {
		return 1
	}
	weight, error := strconv.Atoi(weightStr)
	if error != nil || weight <= 0 {
		return 1
	}
	return weight
}

// RankAndSort randomly sorts the list of nodes.
func (p *JenkinsPlacementStrategy) RankAndSort(config *cluster.ContainerConfig, nodes []*node.Node) ([]*node.Node, error) {
	if config.HostConfig.CPUShares > 0 && config.HostConfig.Memory > 0 {
		binpack := BinpackPlacementStrategy{}
		return binpack.RankAndSort(config, nodes)
	}
	_, isJenkinsSlave := config.Labels["com.nirima.jenkins.plugins.docker.JenkinsId"]
	if !isJenkinsSlave {
		spread := SpreadPlacementStrategy{}
		return spread.RankAndSort(config, nodes)
	}

	// for jenkins, a healthy node should decrease its weight to increase its chance of being selected
	const healthFactor int64 = -5
	weightedNodes, err := weightNodesByCapacity(config, nodes, healthFactor)
	if err != nil {
		return nil, err
	}

	sort.Sort(weightedNodes)
	output := make([]*node.Node, len(weightedNodes))
	for i, n := range weightedNodes {
		output[i] = n.Node
	}
	return output, nil
}
