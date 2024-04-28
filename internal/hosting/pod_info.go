package hosting

import (
	"fmt"
	"os"
)

type PodInfo struct {
	Network      string
	PodName      string
	PodNamespace string
}

func ParsePodInfo() *PodInfo {
	return &PodInfo{
		Network:      os.Getenv("CSMC_NETWORK"),
		PodName:      os.Getenv("POD_NAME"),
		PodNamespace: os.Getenv("POD_NAMESPACE"),
	}
}

func (p PodInfo) RPCBaseSubject() string {
	return fmt.Sprintf("csmc.%s.%s", p.PodNamespace, p.Network)
}

func (p PodInfo) DebugString() string {
	return fmt.Sprintf("PodInfo{Network: %s, PodName: %s, PodNamespace: %s}", p.Network, p.PodName, p.PodNamespace)
}

func (p PodInfo) KVNetworkKey() string {
	return fmt.Sprintf("csmc_%s_%s", p.PodNamespace, p.Network)
}

func (p PodInfo) KVGamemodesKey() string {
	return fmt.Sprintf("%s_gamemodes", p.KVNetworkKey())
}

// csmc_<namespace>_<network>_instances<Container hostname, InstanceInfo>
func (p PodInfo) KVInstancesKey() string {
	return fmt.Sprintf("%s_instances", p.KVNetworkKey())
}

type InstanceInfo struct {
	Gamemode string `json:"gamemode"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
}
