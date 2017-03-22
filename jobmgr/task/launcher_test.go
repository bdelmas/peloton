package task

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"

	"go.uber.org/yarpc"

	mesos "mesos/v1"
	"peloton/api/job"
	"peloton/api/task"
	"peloton/api/task/config"
	"peloton/private/hostmgr/hostsvc"
	"peloton/private/resmgr"
	"peloton/private/resmgrsvc"

	"code.uber.internal/infra/peloton/jobmgr"
	"code.uber.internal/infra/peloton/util"

	store_mocks "code.uber.internal/infra/peloton/storage/mocks"
	yarpc_mocks "code.uber.internal/infra/peloton/vendor_mocks/go.uber.org/yarpc/encoding/json/mocks"
	"github.com/uber-go/tally"
)

const (
	taskIDFmt   = "testjob-%d-abcdefgh-abcd-1234-5678-1234567890"
	testJobName = "testjob"
)

var (
	defaultResourceConfig = config.ResourceConfig{
		CpuLimit:    10,
		MemLimitMb:  10,
		DiskLimitMb: 10,
		FdLimit:     10,
	}
)

func createTestTask(instanceID int) *task.TaskInfo {
	var tid = fmt.Sprintf(taskIDFmt, instanceID)

	return &task.TaskInfo{
		JobId: &job.JobID{
			Value: testJobName,
		},
		InstanceId: uint32(instanceID),
		Config: &config.TaskConfig{
			Name:     testJobName,
			Resource: &defaultResourceConfig,
		},
		Runtime: &task.RuntimeInfo{
			TaskId: &mesos.TaskID{
				Value: &tid,
			},
		},
	}
}

func createResources(defaultMultiplier float64) []*mesos.Resource {
	values := map[string]float64{
		"cpus": defaultMultiplier * defaultResourceConfig.CpuLimit,
		"mem":  defaultMultiplier * defaultResourceConfig.MemLimitMb,
		"disk": defaultMultiplier * defaultResourceConfig.DiskLimitMb,
		"gpus": defaultMultiplier * defaultResourceConfig.GpuLimit,
	}
	return util.CreateMesosScalarResources(values, "*")
}

func createHostOffer(hostID int, resources []*mesos.Resource) *hostsvc.HostOffer {
	agentID := fmt.Sprintf("agent-%d", hostID)
	return &hostsvc.HostOffer{
		Hostname: fmt.Sprintf("hostname-%d", hostID),
		AgentId: &mesos.AgentID{
			Value: &agentID,
		},
		Resources: resources,
	}
}

// This test ensures that multiple placements returned from resmgr can be properly placed by hostmgr
func TestMultipleTasksPlaced(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRes := yarpc_mocks.NewMockClient(ctrl)
	mockHostMgr := yarpc_mocks.NewMockClient(ctrl)
	mockTaskStore := store_mocks.NewMockTaskStore(ctrl)
	testScope := tally.NewTestScope("", map[string]string{})
	metrics := NewMetrics(testScope)
	taskLauncher := launcher{
		config: &jobmgr.Config{
			PlacementDequeueLimit: 100,
		},
		resMgrClient:  mockRes,
		hostMgrClient: mockHostMgr,
		taskStore:     mockTaskStore,
		rootCtx:       context.Background(),
		metrics:       metrics,
	}

	// generate 25 test tasks
	numTasks := 1
	testTasks := make([]*task.TaskInfo, numTasks)
	taskIDs := make([]*task.TaskID, numTasks)
	placements := make([]*resmgr.Placement, numTasks)
	taskConfigs := make(map[string]*config.TaskConfig)
	taskIds := make(map[string]*task.TaskID)
	for i := 0; i < numTasks; i++ {
		tmp := createTestTask(i)
		taskID := &task.TaskID{
			Value: tmp.JobId.Value + "-" + fmt.Sprint(tmp.InstanceId),
		}
		taskIDs[i] = taskID
		testTasks[i] = tmp
		taskConfigs[tmp.GetRuntime().GetTaskId().GetValue()] = tmp.Config
		taskIds[taskID.Value] = taskID
	}

	// generate 1 host offer, each can hold 1 tasks.
	numHostOffers := 1
	rs := createResources(1)
	var hostOffers []*hostsvc.HostOffer
	for i := 0; i < numHostOffers; i++ {
		hostOffers = append(hostOffers, createHostOffer(i, rs))
	}

	// Generate Placements per host offer
	for i := 0; i < numHostOffers; i++ {
		p := createPlacements(testTasks[i], hostOffers[i])
		placements[i] = p
	}

	// Capture LaunchTasks calls
	hostsLaunchedOn := make(map[string]bool)
	launchedTasks := make(map[string]*config.TaskConfig)

	gomock.InOrder(

		mockRes.EXPECT().
			Call(
				gomock.Any(),
				gomock.Eq(yarpc.NewReqMeta().Procedure("ResourceManagerService.GetPlacements")),
				gomock.Any(),
				gomock.Any()).
			Do(func(_ context.Context, _ yarpc.CallReqMeta, _ interface{}, resBodyOut interface{}) {
				o := resBodyOut.(*resmgrsvc.GetPlacementsResponse)
				*o = resmgrsvc.GetPlacementsResponse{
					Placements: placements,
				}
			}).
			Return(nil, nil),

		// Mock Task Store GetTaskByID
		mockTaskStore.EXPECT().GetTaskByID(taskIDs[0].Value).Return(testTasks[0], nil),

		// Mock LaunchTasks call.
		mockHostMgr.EXPECT().
			Call(
				gomock.Any(),
				gomock.Eq(yarpc.NewReqMeta().Procedure("InternalHostService.LaunchTasks")),
				gomock.Any(),
				gomock.Any()).
			Do(func(_ context.Context, _ yarpc.CallReqMeta, reqBody interface{}, _ interface{}) {
				// No need to unmarksnal output: empty means success.
				// Capture call since we don't know ordering of tasks.
				req := reqBody.(*hostsvc.LaunchTasksRequest)
				hostsLaunchedOn[req.Hostname] = true
				for _, lt := range req.Tasks {
					launchedTasks[lt.TaskId.GetValue()] = lt.Config
				}
			}).
			Return(nil, nil).
			Times(1),
	)

	placements, err := taskLauncher.getPlacements()

	if err != nil {
		assert.Error(t, err)
	}
	taskLauncher.processPlacements(placements)

	expectedLaunchedHosts := map[string]bool{
		"hostname-0": true,
	}

	assert.Equal(t, expectedLaunchedHosts, hostsLaunchedOn)
	assert.Equal(t, taskConfigs, launchedTasks)
}

// createPlacements creates the placement
func createPlacements(t *task.TaskInfo,
	hostOffer *hostsvc.HostOffer) *resmgr.Placement {
	TasksIds := make([]*task.TaskID, 1)

	taskID := &task.TaskID{
		Value: t.JobId.Value + "-" + fmt.Sprint(t.InstanceId),
	}
	TasksIds[0] = taskID
	placement := &resmgr.Placement{
		AgentId:  hostOffer.AgentId,
		Hostname: hostOffer.Hostname,
		Tasks:    TasksIds,
	}

	return placement
}