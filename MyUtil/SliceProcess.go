package MyUtil

import "blockchainProject-refactor/Data"

func MergeNodeSlice(s1 []Data.Node, s2 []Data.Node) []Data.Node {
	slice := make([]Data.Node, len(s1)+len(s2))
	copy(slice, s1)
	copy(slice[len(s1):], s2)
	return slice
}
