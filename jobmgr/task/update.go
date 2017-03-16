package task

import (
	"code.uber.internal/infra/peloton/common"
	"code.uber.internal/infra/peloton/common/eventstream"
	"code.uber.internal/infra/peloton/storage"
	"code.uber.internal/infra/peloton/util"
	log "github.com/Sirupsen/logrus"
	"go.uber.org/yarpc"
	mesos "mesos/v1"
	p_task "peloton/api/task"
	pb_eventstream "peloton/private/eventstream"
)

// StatusUpdate reads and processes the task state change events from HM
type StatusUpdate struct {
	taskStore   storage.TaskStore
	eventClient *eventstream.Client
	applier     *asyncEventProcessor
}

// InitTaskStatusUpdate creates a StatusUpdate
// TODO: add shutdown method to stop the eventClient
// In case the current jobmgr lost leadership
func InitTaskStatusUpdate(
	d yarpc.Dispatcher,
	server string,
	taskStore storage.TaskStore) *StatusUpdate {

	statusUpdate := &StatusUpdate{
		taskStore: taskStore,
	}
	// TODO: add config for BucketEventProcessor
	statusUpdate.applier = newBucketEventProcessor(statusUpdate, 100, 10000)

	eventClient := eventstream.NewEventStreamClient(d, common.PelotonJobManager, server, statusUpdate)
	statusUpdate.eventClient = eventClient

	return statusUpdate
}

// OnEvent is the callback function notifying an event
func (p *StatusUpdate) OnEvent(event *pb_eventstream.Event) {
	log.WithField("event_offset", event.Offset).Debug("JobMgr receiving event")
	p.applier.addEvent(event)
}

// GetEventProgress returns the progress of the event progressing
func (p *StatusUpdate) GetEventProgress() uint64 {
	return p.applier.GetEventProgress()
}

// ProcessStatusUpdate processes the actual task status
func (p *StatusUpdate) ProcessStatusUpdate(taskStatus *mesos.TaskStatus) error {
	mesosTaskID := taskStatus.GetTaskId().GetValue()
	taskID, err := util.ParseTaskIDFromMesosTaskID(mesosTaskID)
	if err != nil {
		log.WithError(err).
			WithField("task_id", mesosTaskID).
			Error("Fail to parse taskID for mesostaskID")
		return err
	}
	taskInfo, err := p.taskStore.GetTaskByID(taskID)
	if err != nil {
		log.WithError(err).
			WithField("task_id", taskID).
			Error("Fail to find taskInfo for taskID")
		return err
	}
	state := util.MesosStateToPelotonState(taskStatus.GetState())

	// TODO: figure out on what cases state updates should not be persisted

	// TODO: depends on the state, may need to put the task back to
	// the queue, or clear the pending task record from taskqueue
	taskInfo.GetRuntime().State = state

	// persist error message to help end user figure out root cause
	if isUnexpected(state) {
		taskInfo.GetRuntime().Message = taskStatus.GetMessage()
		taskInfo.GetRuntime().Reason = taskStatus.GetReason().String()
		// TODO: Add metrics for unexpected task updates
		log.WithFields(log.Fields{
			"task_id": taskID,
			"state":   state,
			"message": taskStatus.GetMessage(),
			"reason":  taskStatus.GetReason().String()}).
			Debug("Received unexpected update for task")
	}

	err = p.taskStore.UpdateTask(taskInfo)
	if err != nil {
		log.WithError(err).
			WithFields(log.Fields{
				"task_id": taskID,
				"State":   state}).
			Error("Fail to update taskInfo for taskID")
		return err
	}
	return nil
}

// isUnexpected tells if taskState is unexpected or not
func isUnexpected(taskState p_task.RuntimeInfo_TaskState) bool {
	switch taskState {
	case p_task.RuntimeInfo_FAILED,
		p_task.RuntimeInfo_LOST:
		return true
	default:
		// TODO: we may want to treat unknown state as error
		return false
	}
}

// OnEvents is the callback function notifying a batch of events
func (p *StatusUpdate) OnEvents(events []*pb_eventstream.Event) {
	// Not implemented
}