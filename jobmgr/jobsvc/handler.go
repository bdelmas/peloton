package jobsvc

import (
	"context"
	"fmt"
	"time"

	"github.com/pborman/uuid"
	er "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uber-go/tally"

	"go.uber.org/yarpc"

	"code.uber.internal/infra/peloton/.gen/peloton/api/errors"
	"code.uber.internal/infra/peloton/.gen/peloton/api/job"
	"code.uber.internal/infra/peloton/.gen/peloton/api/peloton"
	"code.uber.internal/infra/peloton/.gen/peloton/api/query"
	"code.uber.internal/infra/peloton/.gen/peloton/api/respool"
	"code.uber.internal/infra/peloton/.gen/peloton/api/task"
	"code.uber.internal/infra/peloton/.gen/peloton/private/resmgrsvc"

	jobmgr_job "code.uber.internal/infra/peloton/jobmgr/job"
	"code.uber.internal/infra/peloton/jobmgr/job/updater"
	jobmgr_task "code.uber.internal/infra/peloton/jobmgr/task"
	task_config "code.uber.internal/infra/peloton/jobmgr/task/config"
	"code.uber.internal/infra/peloton/jobmgr/tracked"
	"code.uber.internal/infra/peloton/storage"
)

const (
	_defaultRPCTimeout = 10 * time.Second
)

// InitServiceHandler initalizes the job manager
func InitServiceHandler(
	d *yarpc.Dispatcher,
	parent tally.Scope,
	jobStore storage.JobStore,
	taskStore storage.TaskStore,
	trackedManager tracked.Manager,
	runtimeUpdater *jobmgr_job.RuntimeUpdater,
	clientName string) {

	handler := &serviceHandler{
		jobStore:       jobStore,
		taskStore:      taskStore,
		respoolClient:  respool.NewResourceManagerYARPCClient(d.ClientConfig(clientName)),
		resmgrClient:   resmgrsvc.NewResourceManagerServiceYARPCClient(d.ClientConfig(clientName)),
		trackedManager: trackedManager,
		runtimeUpdater: runtimeUpdater,
		metrics:        NewMetrics(parent.SubScope("jobmgr").SubScope("job")),
	}

	d.Register(job.BuildJobManagerYARPCProcedures(handler))
}

// serviceHandler implements peloton.api.job.JobManager
type serviceHandler struct {
	jobStore       storage.JobStore
	taskStore      storage.TaskStore
	respoolClient  respool.ResourceManagerYARPCClient
	resmgrClient   resmgrsvc.ResourceManagerServiceYARPCClient
	trackedManager tracked.Manager
	runtimeUpdater *jobmgr_job.RuntimeUpdater
	metrics        *Metrics
}

// Create creates a job object for a given job configuration and
// enqueues the tasks for scheduling
func (h *serviceHandler) Create(
	ctx context.Context,
	req *job.CreateRequest) (*job.CreateResponse, error) {

	log.WithField("request", req).Debug("JobManager.Create called")

	h.metrics.JobAPICreate.Inc(1)

	jobID := req.Id
	// It is possible that jobId is nil since protobuf doesn't enforce it
	if jobID == nil {
		jobID = &peloton.JobID{Value: ""}
	}

	if len(jobID.Value) == 0 {
		jobID.Value = uuid.New()
		log.WithField("jobID", jobID).Info("Genarating UUID ID for empty job ID")
	} else {
		if uuid.Parse(jobID.Value) == nil {
			log.WithField("job_id", jobID.Value).Warn("JobID is not valid UUID")
			return &job.CreateResponse{
				Error: &job.CreateResponse_Error{
					InvalidJobId: &job.InvalidJobId{
						Id:      req.Id,
						Message: "JobID must be valid UUID",
					},
				},
			}, nil
		}
	}
	jobConfig := req.Config

	err := h.validateResourcePool(ctx, jobConfig.RespoolID)
	if err != nil {
		return &job.CreateResponse{
			Error: &job.CreateResponse_Error{
				InvalidConfig: &job.InvalidJobConfig{
					Id:      req.Id,
					Message: err.Error(),
				},
			},
		}, nil
	}

	log.WithField("config", jobConfig).Infof("JobManager.Create called")

	// Validate job config with default task configs
	err = task_config.ValidateTaskConfig(jobConfig)
	if err != nil {
		return &job.CreateResponse{
			Error: &job.CreateResponse_Error{
				InvalidConfig: &job.InvalidJobConfig{
					Id:      req.Id,
					Message: err.Error(),
				},
			},
		}, nil
	}

	// First persist the job configuration, to get a unique version.
	if err := h.jobStore.CreateJobConfig(ctx, jobID, jobConfig); err != nil {
		h.metrics.JobCreateFail.Inc(1)
		return &job.CreateResponse{
			Error: &job.CreateResponse_Error{
				AlreadyExists: &job.JobAlreadyExists{
					Id:      req.Id,
					Message: err.Error(),
				},
			},
		}, nil
	}

	// Create the runtime, pointing to the newly create config.
	runtime := &job.RuntimeInfo{
		State:        job.JobState_INITIALIZED,
		CreationTime: time.Now().UTC().Format(time.RFC3339Nano),
		// Init the task stats to reflect that all tasks are in initialized state.
		TaskStats:     map[string]uint32{task.TaskState_INITIALIZED.String(): jobConfig.InstanceCount},
		ConfigVersion: int64(jobConfig.GetRevision().GetVersion()),
	}
	if h.jobStore.CreateJobRuntime(ctx, jobID, runtime, jobConfig); err != nil {
		h.metrics.JobCreateFail.Inc(1)
		return &job.CreateResponse{
			Error: &job.CreateResponse_Error{
				AlreadyExists: &job.JobAlreadyExists{
					Id:      req.Id,
					Message: err.Error(),
				},
			},
		}, nil
	}
	h.metrics.JobCreate.Inc(1)

	// Detach a new goroutine, for creating tasks and enqueue, as the job is now
	// fully persisted in the store and there are no reason for blocking the
	// client.
	// Note the use of a new context, as we no longer want to honor the request
	// deadline.
	go h.createAndEnqueueTasks(context.Background(), jobID, jobConfig)

	return &job.CreateResponse{
		JobId: jobID,
	}, nil
}

// TODO: Merge with recovery, such that it will use same path for creating/
// recovering.
func (h *serviceHandler) createAndEnqueueTasks(
	ctx context.Context,
	jobID *peloton.JobID,
	jobConfig *job.JobConfig) error {
	instances := jobConfig.InstanceCount
	startAddTaskTime := time.Now()

	if err := h.taskStore.CreateTaskConfigs(ctx, jobID, jobConfig); err != nil {
		log.WithError(err).
			WithField("job_id", jobID.Value).
			Error("Failed to create task configs")
		return err
	}

	runtimes := make([]*task.RuntimeInfo, instances)
	for i := uint32(0); i < instances; i++ {
		runtimes[i] = jobmgr_task.CreateInitializingTask(jobID, i, jobConfig)
	}

	// TODO: use the username of current session for createBy param
	err := h.taskStore.CreateTaskRuntimes(ctx, jobID, runtimes, "peloton")
	nTasks := int64(len(runtimes))
	if err != nil {
		log.Errorf("Failed to create tasks (%d) for job %v: %v",
			nTasks, jobID.Value, err)
		h.metrics.TaskCreateFail.Inc(nTasks)
		return err
	}
	h.metrics.TaskCreate.Inc(nTasks)

	for i := uint32(0); i < instances; i++ {
		h.trackedManager.SetTask(jobID, i, runtimes[i])
	}

	jobRuntime, err := h.jobStore.GetJobRuntime(ctx, jobID)
	if err != nil {
		log.WithError(err).
			WithField("job_id", jobID.Value).
			Error("Failed to GetJobRuntime")
		return err
	}
	jobRuntime.State = job.JobState_PENDING
	err = h.jobStore.UpdateJobRuntime(ctx, jobID, jobRuntime, nil)
	if err != nil {
		log.WithError(err).
			WithField("job_id", jobID.Value).
			Error("Failed to UpdateJobRuntime")
		return err
	}

	log.Infof("Job %v all %v tasks created, time spent: %v",
		jobID.Value, instances, time.Since(startAddTaskTime))

	return nil
}

// Update updates a job object for a given job configuration and
// performs the appropriate action based on the change
func (h *serviceHandler) Update(
	ctx context.Context,
	req *job.UpdateRequest) (*job.UpdateResponse, error) {

	h.metrics.JobAPIUpdate.Inc(1)

	jobID := req.Id
	info, err := h.jobStore.GetJob(ctx, jobID)
	if err != nil {
		log.WithError(err).
			WithField("job_id", jobID.Value).
			Error("Failed to GetJob")
		h.metrics.JobUpdateFail.Inc(1)
		return nil, err
	}

	if !jobmgr_job.NonTerminatedStates[info.Runtime.State] {
		msg := fmt.Sprintf("Job is in a terminal state:%s", info.Runtime.State)
		h.metrics.JobUpdateFail.Inc(1)
		return &job.UpdateResponse{
			Error: &job.UpdateResponse_Error{
				InvalidJobId: &job.InvalidJobId{
					Id:      req.Id,
					Message: msg,
				},
			},
		}, nil
	}

	newConfig := req.Config
	oldConfig := info.Config

	if newConfig.RespoolID == nil {
		newConfig.RespoolID = oldConfig.RespoolID
	}

	diff, err := updater.CalculateJobDiff(jobID, oldConfig, newConfig)
	if err != nil {
		h.metrics.JobUpdateFail.Inc(1)
		return &job.UpdateResponse{
			Error: &job.UpdateResponse_Error{
				InvalidConfig: &job.InvalidJobConfig{
					Id:      jobID,
					Message: err.Error(),
				},
			},
		}, nil
	}

	if diff.IsNoop() {
		log.WithField("job_id", jobID.Value).
			Info("update is a noop")
		return nil, nil
	}

	if err = h.jobStore.CreateJobConfig(ctx, jobID, newConfig); err != nil {
		h.metrics.JobUpdateFail.Inc(1)
		return &job.UpdateResponse{
			Error: &job.UpdateResponse_Error{
				JobNotFound: &job.JobNotFound{
					Id:      req.Id,
					Message: err.Error(),
				},
			},
		}, nil
	}
	info.Runtime.ConfigVersion = int64(newConfig.GetRevision().GetVersion())

	if err = h.jobStore.UpdateJobRuntime(ctx, jobID, info.Runtime, newConfig); err != nil {
		h.metrics.JobUpdateFail.Inc(1)
		return &job.UpdateResponse{
			Error: &job.UpdateResponse_Error{
				JobNotFound: &job.JobNotFound{
					Id:      req.Id,
					Message: err.Error(),
				},
			},
		}, nil
	}

	log.WithField("job_id", jobID.Value).
		Infof("adding %d instances", len(diff.InstancesToAdd))

	if err := h.taskStore.CreateTaskConfigs(ctx, jobID, newConfig); err != nil {
		h.metrics.JobUpdateFail.Inc(1)
		return nil, err
	}

	// TODO: Update goal state version of existing tasks to the new version.
	for id, runtime := range diff.InstancesToAdd {
		if err := h.taskStore.CreateTaskRuntime(ctx, jobID, id, runtime, "peloton"); err != nil {
			log.Errorf("Failed to create task for job %v: %v", jobID.Value, err)
			h.metrics.TaskCreateFail.Inc(1)
			// FIXME: Add a new Error type for this
			return nil, err
		}

		h.metrics.TaskCreate.Inc(1)
		h.trackedManager.SetTask(jobID, id, runtime)
	}

	err = h.runtimeUpdater.UpdateJob(ctx, jobID)
	if err != nil {
		log.WithError(err).
			WithField("job_id", jobID.Value).
			Error("Failed to update job runtime")
		h.metrics.JobUpdateFail.Inc(1)
		return nil, err
	}

	h.metrics.JobUpdate.Inc(1)
	msg := fmt.Sprintf("added %d instances", len(diff.InstancesToAdd))
	return &job.UpdateResponse{
		Id:      jobID,
		Message: msg,
	}, nil
}

// Get returns a job config for a given job ID
func (h *serviceHandler) Get(
	ctx context.Context,
	req *job.GetRequest) (*job.GetResponse, error) {

	log.WithField("request", req).Debug("JobManager.Get called")
	h.metrics.JobAPIGet.Inc(1)

	info, err := h.jobStore.GetJob(ctx, req.Id)
	if err != nil {
		h.metrics.JobGetFail.Inc(1)
		log.WithError(err).
			WithField("job_id", req.Id.Value).
			Info("GetJob failed")
		return &job.GetResponse{
			Error: &job.GetResponse_Error{
				NotFound: &errors.JobNotFound{
					Id:      req.Id,
					Message: err.Error(),
				},
			},
		}, nil
	}

	resp := &job.GetResponse{
		JobInfo: info,
	}
	h.metrics.JobGet.Inc(1)
	log.WithField("response", resp).Debug("JobManager.Get returned")
	return resp, nil
}

// Query returns a list of jobs matching the given query
func (h *serviceHandler) Query(ctx context.Context, req *job.QueryRequest) (*job.QueryResponse, error) {
	log.WithField("request", req).Info("JobManager.Query called")
	h.metrics.JobAPIQuery.Inc(1)

	jobConfigs, total, err := h.jobStore.QueryJobs(ctx, req.GetRespoolID(), req.GetSpec())
	if err != nil {
		h.metrics.JobQueryFail.Inc(1)
		log.WithError(err).Error("Query job failed with error")
		return &job.QueryResponse{
			Error: &job.QueryResponse_Error{
				Err: &errors.UnknownError{
					Message: err.Error(),
				},
			},
		}, nil
	}

	h.metrics.JobQuery.Inc(1)
	resp := &job.QueryResponse{
		Records: jobConfigs,
		Pagination: &query.Pagination{
			Offset: req.GetSpec().GetPagination().GetOffset(),
			Limit:  req.GetSpec().GetPagination().GetLimit(),
			Total:  total,
		},
	}
	log.WithField("response", resp).Debug("JobManager.Query returned")
	return resp, nil
}

// Delete kills all running tasks in a job
func (h *serviceHandler) Delete(
	ctx context.Context,
	req *job.DeleteRequest) (*job.DeleteResponse, error) {

	h.metrics.JobAPIDelete.Inc(1)

	jobRuntime, err := h.jobStore.GetJobRuntime(ctx, req.GetId())
	if err != nil {
		log.WithError(err).
			WithField("job_id", req.GetId().GetValue()).
			Error("Failed to GetJobRuntime")
		h.metrics.JobUpdateFail.Inc(1)
		return nil, err
	}

	if jobmgr_job.NonTerminatedStates[jobRuntime.State] {
		h.metrics.JobUpdateFail.Inc(1)
		return nil, fmt.Errorf("Job is not in a terminal state: %s", jobRuntime.State)
	}

	if err := h.jobStore.DeleteJob(ctx, req.Id); err != nil {
		h.metrics.JobDeleteFail.Inc(1)
		log.Errorf("Delete job failed with error %v", err)
		return &job.DeleteResponse{
			Error: &job.DeleteResponse_Error{
				NotFound: &errors.JobNotFound{
					Id:      req.Id,
					Message: err.Error(),
				},
			},
		}, nil
	}

	h.metrics.JobDelete.Inc(1)
	return &job.DeleteResponse{}, nil
}

// validateResourcePool validates the resource pool before submitting job
func (h *serviceHandler) validateResourcePool(
	ctx context.Context,
	respoolID *peloton.ResourcePoolID,
) error {
	ctx, cancelFunc := context.WithTimeout(ctx, _defaultRPCTimeout)
	defer cancelFunc()
	if respoolID == nil {
		return er.New("Resource Pool Id is null")
	}

	var request = &respool.GetRequest{
		Id: respoolID,
	}
	response, err := h.respoolClient.GetResourcePool(ctx, request)
	if err != nil {
		log.WithFields(log.Fields{
			"error":     err,
			"respoolID": respoolID.Value,
		}).Error("Failed to get Resource Pool")
		return err
	}
	if response.GetError() != nil {
		log.WithFields(log.Fields{
			"error":     err,
			"respoolID": respoolID.Value,
		}).Info("Resource Pool Not Found")
		return er.New(response.Error.String())
	}

	if response.GetPoolinfo() != nil && response.GetPoolinfo().Id != nil {
		if response.GetPoolinfo().Id.Value != respoolID.Value {
			return er.New("Resource Pool Not Found")
		}
	} else {
		return er.New("Resource Pool Not Found")
	}

	return nil
}
