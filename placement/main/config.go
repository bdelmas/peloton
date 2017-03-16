package main

import (
	"code.uber.internal/infra/peloton/common/metrics"
	"code.uber.internal/infra/peloton/hostmgr/mesos"
	"code.uber.internal/infra/peloton/leader"
	"code.uber.internal/infra/peloton/placement"
)

// Config holds all configs to run a placement engine.
type Config struct {
	Metrics   metrics.Config        `yaml:"metrics"`
	Placement placement.Config      `yaml:"placement"`
	Election  leader.ElectionConfig `yaml:"election"`
	Mesos     mesos.Config          `yaml:"mesos"`
}