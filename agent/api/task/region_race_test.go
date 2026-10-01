//go:build unit
// +build unit

// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//	http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package task

import (
	"encoding/json"
	"sync"
	"testing"

	apicontainer "github.com/aws/amazon-ecs-agent/agent/api/container"
)

// Task save marshals every container while createContainer injects region env
// into one of them; run with -race to catch an unlocked write.
func TestApplyRegionToContainerConcurrentWithTaskMarshal(t *testing.T) {
	for i := 0; i < 2000; i++ {
		c1 := &apicontainer.Container{Name: "c1", Environment: map[string]string{"A": "1"}}
		c2 := &apicontainer.Container{Name: "c2", Environment: map[string]string{"B": "2"}}
		tsk := &Task{Arn: "arn:aws:ecs:us-east-1:123456789012:task/c/abc", Containers: []*apicontainer.Container{c1, c2}}

		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); json.Marshal(tsk) }()
		go func() { defer wg.Done(); tsk.ApplyRegionToContainer(c2, "us-east-1", nil) }()
		wg.Wait()
	}
}
