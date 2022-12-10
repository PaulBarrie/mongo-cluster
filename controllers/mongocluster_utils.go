package controllers

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/rand"
)

func getResourceGenericName(prefix, suffix string) string {
	return fmt.Sprintf(MONGO_RESOURCE_FORMAT, prefix, suffix)
}

func getRandomPort() int32 {
	return int32(rand.IntnRange(30000, 32767))
}

type ClusterMembers struct {
	Id   int    `json:"Id"`
	Host string `json:"Host"`
}

func getClusterMembers(resourceName string, clusterSize int) string {
	var members []ClusterMembers
	for i := 0; i < clusterSize; i++ {
		member := ClusterMembers{
			Id:   i,
			Host: getResourceGenericName(resourceName, fmt.Sprintf("%d", i)),
		}
		members = append(members, member)
	}
	membersBytes, err := json.Marshal(members)
	if err != nil {
		return ""
	}
	return string(membersBytes[:])
}
