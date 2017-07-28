package resmgr

import (
	"time"

	"code.uber.internal/infra/peloton/resmgr/task"
)

// Config is Resource Manager specific configuration
type Config struct {
	Port               int `yaml:"port"`
	DbWriteConcurrency int `yaml:"db_write_concurrency"`
	// Period to run task scheduling in seconds
	TaskSchedulingPeriod time.Duration `yaml:"task_scheduling_period"`
	// Period to run entitlement calculator
	EntitlementCaculationPeriod time.Duration `yaml:"entitlement_calculation_period"`
	// RM Task Config
	RmTaskConfig *task.Config `yaml:"task"`
}
