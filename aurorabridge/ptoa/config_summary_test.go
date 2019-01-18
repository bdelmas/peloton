// Copyright (c) 2019 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ptoa

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/uber/peloton/.gen/peloton/api/v1alpha/peloton"
	"github.com/uber/peloton/.gen/peloton/api/v1alpha/pod"
	"github.com/uber/peloton/aurorabridge/fixture"
	"github.com/uber/peloton/aurorabridge/label"
)

// Tests the success scenario to get config summary for provided
// list of pod infos
func TestNewConfigSumamry_Success(t *testing.T) {
	jobKey := fixture.AuroraJobKey()
	jobID := fixture.PelotonJobID()

	var podInfos []*pod.PodInfo
	for i := 0; i < 6; i++ {
		var entityVersion string
		if i < 2 {
			entityVersion = "1"
		} else if i < 5 {
			entityVersion = "2"
		} else {
			entityVersion = "3"
		}

		podName := fmt.Sprintf("%s-%d", jobID.GetValue(), i)
		podID := fmt.Sprintf("%s-%d", podName, 1)
		mdLabel, err := label.NewAuroraMetadata(fixture.AuroraMetadata())
		assert.NoError(t, err)

		podInfos = append(podInfos, &pod.PodInfo{
			Spec: &pod.PodSpec{
				PodName: &peloton.PodName{
					Value: podName,
				},
				Labels: []*peloton.Label{
					mdLabel,
				},
			},
			Status: &pod.PodStatus{
				PodId: &peloton.PodID{
					Value: podID,
				},
				Version: &peloton.EntityVersion{
					Value: entityVersion,
				},
			},
		})
	}

	configSummary, err := NewConfigSummary(
		jobKey,
		podInfos,
	)
	assert.NoError(t, err)
	// Creates two config groups indicating set of pods which have same entity version (same pod spec)
	assert.Equal(t, 3, len(configSummary.GetGroups()))
}

// Test the error scenario for invalid task ID for provided pod info

func TestNewConfigSummary_InvalidTaskID(t *testing.T) {
	jobKey := fixture.AuroraJobKey()
	podName := fmt.Sprintf("%s-%d", "dummy_job_id", 0)
	podID := fmt.Sprintf("%s-%d", podName, 1)
	entityVersion := fixture.PelotonEntityVersion()

	podInfos := []*pod.PodInfo{&pod.PodInfo{
		Spec: &pod.PodSpec{
			PodName: &peloton.PodName{
				Value: podName,
			},
		},
		Status: &pod.PodStatus{
			PodId: &peloton.PodID{
				Value: podID,
			},
			Version: entityVersion,
		},
	}}
	_, err := NewConfigSummary(jobKey, podInfos)
	assert.Error(t, err)
}

// Test the error scenario for incorrect config group
func TestNewConfigSummary_ConfigGroupError(t *testing.T) {
	jobKey := fixture.AuroraJobKey()
	jobID := fixture.PelotonJobID()
	podName := fmt.Sprintf("%s-%d", jobID.GetValue(), 0)
	podID := fmt.Sprintf("%s-%d", podName, 1)
	entityVersion := fixture.PelotonEntityVersion()

	podInfos := []*pod.PodInfo{&pod.PodInfo{
		Spec: &pod.PodSpec{
			PodName: &peloton.PodName{
				Value: podName,
			},
		},
		Status: &pod.PodStatus{
			PodId: &peloton.PodID{
				Value: podID,
			},
			Version: entityVersion,
		},
	}}
	_, err := NewConfigSummary(jobKey, podInfos)
	assert.Error(t, err)
}