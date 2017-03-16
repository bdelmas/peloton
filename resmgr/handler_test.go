package resmgr

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
	"github.com/uber-go/tally"

	"code.uber.internal/infra/peloton/common/queue"
	"code.uber.internal/infra/peloton/resmgr/respool"
	rm_task "code.uber.internal/infra/peloton/resmgr/task"
	store_mocks "code.uber.internal/infra/peloton/storage/mocks"

	"peloton/api/job"
	pb_respool "peloton/api/respool"
	"peloton/api/task"
	"peloton/private/resmgr"
	"peloton/private/resmgrsvc"
)

type HandlerTestSuite struct {
	suite.Suite
	handler  *serviceHandler
	context  context.Context
	resPools map[string]*pb_respool.ResourcePoolConfig
	ctrl     *gomock.Controller
}

func (suite *HandlerTestSuite) SetupTest() {
	suite.context = context.Background()

	// FIXME after making the respool tree and task scheduler re-entryable
	// suite.ctrl = gomock.NewController(suite.T())

	// mockResPoolStore := store_mocks.NewMockResourcePoolStore(suite.ctrl)
	// gomock.InOrder(
	//      mockResPoolStore.EXPECT().
	//      GetAllResourcePools().Return(suite.getResPools(), nil),
	// )
	// respool.InitTree(tally.NoopScope, mockResPoolStore)
	// err := respool.GetTree().Start()
	// suite.NoError(err)

	// rm_task.InitScheduler(1 * time.Second)
	// err = rm_task.GetScheduler().Start()
	// suite.NoError(err)

	// suite.handler = &serviceHandler{
	//	metrics:  NewMetrics(tally.NoopScope),
	//	resPoolTree: respool.GetTree(),
	//	placements: queue.NewQueue(
	//		"placement-queue",
	//		reflect.TypeOf(&resmgr.Placement{}),
	//		maxPlacementQueueSize,
	//	),
	// }
}

func (suite *HandlerTestSuite) TearDownTest() {
	log.Info("tearing down")

	// FIXME after making the respool tree and task scheduler re-entryable
	// err := respool.GetTree().Stop()
	// suite.NoError(err)

	// err = rm_task.GetScheduler().Stop()
	// suite.NoError(err)

	// suite.ctrl.Finish()
}

func TestResManagerHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}

func (suite *HandlerTestSuite) getResourceConfig() []*pb_respool.ResourceConfig {

	resConfigs := []*pb_respool.ResourceConfig{
		{
			Share:       1,
			Kind:        "cpu",
			Reservation: 100,
			Limit:       1000,
		},
		{
			Share:       1,
			Kind:        "memory",
			Reservation: 100,
			Limit:       1000,
		},
		{
			Share:       1,
			Kind:        "disk",
			Reservation: 100,
			Limit:       1000,
		},
		{
			Share:       1,
			Kind:        "gpu",
			Reservation: 2,
			Limit:       4,
		},
	}
	return resConfigs
}

func (suite *HandlerTestSuite) getResPools() map[string]*pb_respool.ResourcePoolConfig {

	rootID := pb_respool.ResourcePoolID{Value: "root"}
	policy := pb_respool.SchedulingPolicy_PriorityFIFO

	return map[string]*pb_respool.ResourcePoolConfig{
		"root": {
			Name:      "root",
			Parent:    nil,
			Resources: suite.getResourceConfig(),
			Policy:    policy,
		},
		"respool1": {
			Name:      "respool1",
			Parent:    &rootID,
			Resources: suite.getResourceConfig(),
			Policy:    policy,
		},
		"respool2": {
			Name:      "respool2",
			Parent:    &rootID,
			Resources: suite.getResourceConfig(),
			Policy:    policy,
		},
		"respool3": {
			Name:      "respool3",
			Parent:    &rootID,
			Resources: suite.getResourceConfig(),
			Policy:    policy,
		},
		"respool11": {
			Name:      "respool11",
			Parent:    &pb_respool.ResourcePoolID{Value: "respool1"},
			Resources: suite.getResourceConfig(),
			Policy:    policy,
		},
		"respool12": {
			Name:      "respool12",
			Parent:    &pb_respool.ResourcePoolID{Value: "respool1"},
			Resources: suite.getResourceConfig(),
			Policy:    policy,
		},
		"respool21": {
			Name:      "respool21",
			Parent:    &pb_respool.ResourcePoolID{Value: "respool2"},
			Resources: suite.getResourceConfig(),
			Policy:    policy,
		},
		"respool22": {
			Name:      "respool22",
			Parent:    &pb_respool.ResourcePoolID{Value: "respool2"},
			Resources: suite.getResourceConfig(),
			Policy:    policy,
		},
	}
}

func (suite *HandlerTestSuite) pendingTasks() []*resmgr.Task {
	return []*resmgr.Task{
		{
			Name:     "job1-1",
			Priority: 0,
			JobId:    &job.JobID{Value: "job1"},
			Id:       &task.TaskID{Value: "job1-1"},
		},
		{
			Name:     "job1-1",
			Priority: 1,
			JobId:    &job.JobID{Value: "job1"},
			Id:       &task.TaskID{Value: "job1-2"},
		},
		{
			Name:     "job2-1",
			Priority: 2,
			JobId:    &job.JobID{Value: "job2"},
			Id:       &task.TaskID{Value: "job2-1"},
		},
		{
			Name:     "job2-2",
			Priority: 2,
			JobId:    &job.JobID{Value: "job2"},
			Id:       &task.TaskID{Value: "job2-2"},
		},
	}
}

func (suite *HandlerTestSuite) expectedTasks() []*resmgr.Task {
	return []*resmgr.Task{
		{
			Name:     "job2-1",
			Priority: 2,
			JobId:    &job.JobID{Value: "job2"},
			Id:       &task.TaskID{Value: "job2-1"},
		},
		{
			Name:     "job2-2",
			Priority: 2,
			JobId:    &job.JobID{Value: "job2"},
			Id:       &task.TaskID{Value: "job2-2"},
		},
		{
			Name:     "job1-1",
			Priority: 1,
			JobId:    &job.JobID{Value: "job1"},
			Id:       &task.TaskID{Value: "job1-2"},
		},
		{
			Name:     "job1-1",
			Priority: 0,
			JobId:    &job.JobID{Value: "job1"},
			Id:       &task.TaskID{Value: "job1-1"},
		},
	}
}

func (suite *HandlerTestSuite) TestEnqueueDequeueTasksOneResPool() {
	log.Info("TestEnqueueDequeueTasksOneResPool called")

	// Load respool tree from mocked respool store
	ctrl := gomock.NewController(suite.T())
	defer ctrl.Finish()

	mockResPoolStore := store_mocks.NewMockResourcePoolStore(ctrl)
	gomock.InOrder(
		mockResPoolStore.EXPECT().
			GetAllResourcePools().Return(suite.getResPools(), nil),
	)
	respool.InitTree(tally.NoopScope, mockResPoolStore)
	err := respool.GetTree().Start()
	suite.NoError(err)

	rm_task.InitScheduler(1 * time.Second)
	err = rm_task.GetScheduler().Start()
	suite.NoError(err)

	handler := &serviceHandler{
		metrics:     NewMetrics(tally.NoopScope),
		resPoolTree: respool.GetTree(),
		placements:  nil,
	}

	enqReq := &resmgrsvc.EnqueueTasksRequest{
		ResPool: &pb_respool.ResourcePoolID{Value: "respool3"},
		Tasks:   suite.pendingTasks(),
	}
	enqResp, _, err := handler.EnqueueTasks(suite.context, nil, enqReq)
	suite.NoError(err)
	suite.Nil(enqResp.GetError())

	deqReq := &resmgrsvc.DequeueTasksRequest{
		Limit:   10,
		Timeout: 2 * 1000, // 2 sec
	}
	deqResp, _, err := handler.DequeueTasks(suite.context, nil, deqReq)
	suite.NoError(err)
	suite.Nil(deqResp.GetError())
	suite.Equal(suite.expectedTasks(), deqResp.GetTasks())

	log.Info("TestEnqueueDequeueTasksOneResPool returned")
}

func (suite *HandlerTestSuite) TestEnqueueTasksResPoolNotFound() {
	log.Info("TestEnqueueTasksResPoolNotFound called")
	respool.InitTree(tally.NoopScope, nil)

	handler := &serviceHandler{
		metrics:     NewMetrics(tally.NoopScope),
		resPoolTree: respool.GetTree(),
		placements:  nil,
	}

	respoolID := &pb_respool.ResourcePoolID{Value: "respool10"}
	enqReq := &resmgrsvc.EnqueueTasksRequest{
		ResPool: respoolID,
		Tasks:   suite.pendingTasks(),
	}
	enqResp, _, err := handler.EnqueueTasks(suite.context, nil, enqReq)
	suite.NoError(err)
	log.Infof("%v", enqResp)
	notFound := &resmgrsvc.ResourcePoolNotFound{
		Id:      respoolID,
		Message: "Resource pool (respool10) not found",
	}
	suite.Equal(notFound, enqResp.GetError().GetNotFound())
	log.Info("TestEnqueueTasksResPoolNotFound returned")
}

func (suite *HandlerTestSuite) TestEnqueueTasksFailure() {
	// TODO: Mock ResPool.Enqueue task to simulate task enqueue failures
	suite.True(true)
}

func (suite *HandlerTestSuite) getPlacements() []*resmgr.Placement {
	var placements []*resmgr.Placement
	for i := 0; i < 10; i++ {
		var tasks []*task.TaskID
		for j := 0; j < 5; j++ {
			task := &task.TaskID{
				Value: fmt.Sprintf("task-%d-%d", i, j),
			}
			tasks = append(tasks, task)
		}
		placement := &resmgr.Placement{
			Tasks:    tasks,
			Hostname: fmt.Sprintf("host-%d", i),
		}
		placements = append(placements, placement)
	}
	return placements
}

func (suite *HandlerTestSuite) TestSetAndGetPlacementsSuccess() {
	handler := &serviceHandler{
		metrics:     NewMetrics(tally.NoopScope),
		resPoolTree: nil,
		placements: queue.NewQueue(
			"placement-queue",
			reflect.TypeOf(resmgr.Placement{}),
			maxPlacementQueueSize,
		),
	}

	setReq := &resmgrsvc.SetPlacementsRequest{
		Placements: suite.getPlacements(),
	}
	setResp, _, err := handler.SetPlacements(suite.context, nil, setReq)
	suite.NoError(err)
	suite.Nil(setResp.GetError())

	getReq := &resmgrsvc.GetPlacementsRequest{
		Limit:   10,
		Timeout: 1 * 1000, // 1 sec
	}
	getResp, _, err := handler.GetPlacements(suite.context, nil, getReq)
	suite.NoError(err)
	suite.Nil(getResp.GetError())
	suite.Equal(suite.getPlacements(), getResp.GetPlacements())
}