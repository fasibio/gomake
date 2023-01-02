package nearfinder

import "github.com/schollz/closestmatch"

func GetKeysOfMap[K comparable, V any](mapValue map[K]V) []K {
	keys := make([]K, len(mapValue))

	i := 0
	for k := range mapValue {
		keys[i] = k
		i++
	}
	return keys
}

func ClosestMatch(match string, checkList []string, subsetSize int) string {
	bagSizes := []int{subsetSize}
	cm := closestmatch.New(checkList, bagSizes)
	return cm.Closest(match)
}
