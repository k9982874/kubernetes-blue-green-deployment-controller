package util

import (
	"fmt"
	"hash/fnv"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/dump"
)

// ComputeHash returns a hash value calculated from pod template to avoid hash collision.
func ComputeHash(template *v1.PodSpec) uint32 {
	hasher := fnv.New32a()
	hasher.Reset()
	fmt.Fprintf(hasher, "%v", dump.ForHash(*template))
	return hasher.Sum32()
}
