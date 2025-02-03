//
// Copyright 2023 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package resources

import "bytes"

func Contains[T comparable](s []T, e T) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func UnorderedEqualSlice[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}

	elementMap := make(map[T]int, len(a))
	for _, elem := range a {
		elementMap[elem]++
	}

	for _, elem := range b {
		if elementMap[elem] == 0 {
			return false
		}
		elementMap[elem]--
	}

	return true
}

func UnorderedContainsSliceWithHashFunc[T comparable](checked, allNeededFields []T, hashFunc func(T) string) bool {
	neededMap := make(map[string]bool)

	for _, elem := range allNeededFields {
		neededMap[hashFunc(elem)] = false
	}

	for _, elem := range checked {
		if _, ok := neededMap[hashFunc(elem)]; ok {
			neededMap[hashFunc(elem)] = true
		}
	}

	for _, found := range neededMap {
		if !found {
			return false
		}
	}

	return true
}

func MapHasAllPairsFromOther[K, V comparable](checked, allNeededPairs map[K]V) bool {
	for key, value := range allNeededPairs {
		if foundValue, ok := checked[key]; !ok || foundValue != value {
			return false
		}
	}
	return true
}

func MapHasAllKeysFromOther[K comparable, V1, V2 any](checked map[K]V1, mapWithNeededKeys map[K]V2) bool {
	for key := range mapWithNeededKeys {
		if _, ok := checked[key]; !ok {
			return false
		}
	}
	return true
}

func MapHasAllPairsFromOtherBytes[K comparable, V1, V2 []byte](checked map[K]V1, mapWithNeededKeys map[K]V2) bool {
	for key, value := range mapWithNeededKeys {
		if foundValue, ok := checked[key]; !ok || !bytes.Equal(value, foundValue) {
			return false
		}
	}
	return true
}

func UnorderedContainsSlice[T comparable](checked, allNeededFields []T) bool {
	neededMap := make(map[T]bool)

	for _, elem := range allNeededFields {
		neededMap[elem] = false
	}

	for _, elem := range checked {
		if _, ok := neededMap[elem]; ok {
			neededMap[elem] = true
		}
	}

	for _, found := range neededMap {
		if !found {
			return false
		}
	}

	return true
}
