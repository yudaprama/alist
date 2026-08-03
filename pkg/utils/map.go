package utils

import "maps"

func MergeMap(mObj ...map[string]interface{}) map[string]interface{} {
	newObj := map[string]interface{}{}
	for _, m := range mObj {
		maps.Copy(newObj, m)
	}
	return newObj
}
