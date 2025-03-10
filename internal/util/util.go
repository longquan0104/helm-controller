/*
Copyright 2020 The Flux authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"crypto/sha1"
	"fmt"
	"sort"

	goyaml "gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/release"
	"sigs.k8s.io/yaml"
)

// ValuesChecksum calculates and returns the SHA1 checksum for the
// given chartutil.Values.
func ValuesChecksum(values chartutil.Values) string {
	var s string
	if len(values) != 0 {
		s, _ = values.YAML()
	}
	return fmt.Sprintf("%x", sha1.Sum([]byte(s)))
}

// OrderedValuesChecksum sort the chartutil.Values then calculates
// and returns the SHA1 checksum for the sorted values.
func OrderedValuesChecksum(values chartutil.Values) string {
	var s []byte
	if len(values) != 0 {
		msValues := yaml.JSONObjectToYAMLObject(copyValues(values))
		SortMapSlice(msValues)
		s, _ = goyaml.Marshal(msValues)
	}
	return fmt.Sprintf("%x", sha1.Sum(s))
}

// SortMapSlice recursively sorts the given goyaml.MapSlice by key.
// This is used to ensure that the values checksum is the same regardless
// of the order of the keys in the values file.
func SortMapSlice(ms goyaml.MapSlice) {
	sort.Slice(ms, func(i, j int) bool {
		return fmt.Sprint(ms[i].Key) < fmt.Sprint(ms[j].Key)
	})
	for _, item := range ms {
		if nestedMS, ok := item.Value.(goyaml.MapSlice); ok {
			SortMapSlice(nestedMS)
		} else if _, ok := item.Value.([]interface{}); ok {
			for _, vItem := range item.Value.([]interface{}) {
				if itemMS, ok := vItem.(goyaml.MapSlice); ok {
					SortMapSlice(itemMS)
				}
			}
		}
	}
}

// cleanUpMapValue changes all instances of
// map[interface{}]interface{} to map[string]interface{}.
// This is for handling the mismatch when unmarshaling
// reference to the issue: https://github.com/go-yaml/yaml/issues/139
func cleanUpMapValue(v interface{}) interface{} {
	switch v := v.(type) {
	case []interface{}:
		return cleanUpInterfaceArray(v)
	case map[interface{}]interface{}:
		return cleanUpInterfaceMap(v)
	default:
		return v
	}
}

func cleanUpInterfaceMap(in map[interface{}]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range in {
		result[fmt.Sprintf("%v", k)] = cleanUpMapValue(v)
	}
	return result
}

func cleanUpInterfaceArray(in []interface{}) []interface{} {
	result := make([]interface{}, len(in))
	for i, v := range in {
		result[i] = cleanUpMapValue(v)
	}
	return result
}

func copyValues(in map[string]interface{}) map[string]interface{} {
	copiedValues, _ := goyaml.Marshal(in)
	newValues := make(map[string]interface{})

	_ = goyaml.Unmarshal(copiedValues, newValues)
	for i, value := range newValues {
		newValues[i] = cleanUpMapValue(value)
	}

	return newValues
}

// ReleaseRevision returns the revision of the given release.Release.
func ReleaseRevision(rel *release.Release) int {
	if rel == nil {
		return 0
	}
	return rel.Version
}
