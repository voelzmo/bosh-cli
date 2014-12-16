package agentclient

import (
	"fmt"
	"time"

	bosherr "github.com/cloudfoundry/bosh-agent/errors"
	boshlog "github.com/cloudfoundry/bosh-agent/logger"
	bmas "github.com/cloudfoundry/bosh-micro-cli/deployment/applyspec"
	bmhttpclient "github.com/cloudfoundry/bosh-micro-cli/deployment/httpclient"
	bmretrystrategy "github.com/cloudfoundry/bosh-micro-cli/deployment/retrystrategy"
)

type AgentClient interface {
	Ping() (string, error)
	Stop() error
	Apply(bmas.ApplySpec) error
	Start() error
	GetState() (State, error)
	MountDisk(string) error
	UnmountDisk(string) error
	ListDisk() ([]string, error)
	MigrateDisk() error
}

type agentClient struct {
	agentRequest agentRequest
	getTaskDelay time.Duration
	logger       boshlog.Logger
	logTag       string
}

func NewAgentClient(
	endpoint string,
	uuid string,
	getTaskDelay time.Duration,
	httpClient bmhttpclient.HTTPClient,
	logger boshlog.Logger,
) AgentClient {
	agentEndpoint := fmt.Sprintf("%s/agent", endpoint)
	agentRequest := NewAgentRequest(agentEndpoint, httpClient, uuid)

	return &agentClient{
		agentRequest: agentRequest,
		getTaskDelay: getTaskDelay,
		logger:       logger,
		logTag:       "agentClient",
	}
}

func (c *agentClient) Ping() (string, error) {
	var response SimpleTaskResponse
	err := c.agentRequest.Send("ping", []interface{}{}, &response)
	if err != nil {
		return "", bosherr.WrapError(err, "Sending ping to the agent")
	}

	return response.Value, nil
}

func (c *agentClient) Stop() error {
	return c.sendAsyncTaskMessage("stop", []interface{}{})
}

func (c *agentClient) Apply(spec bmas.ApplySpec) error {
	return c.sendAsyncTaskMessage("apply", []interface{}{spec})
}

func (c *agentClient) Start() error {
	var response SimpleTaskResponse
	err := c.agentRequest.Send("start", []interface{}{}, &response)
	if err != nil {
		return bosherr.WrapError(err, "Starting agent services")
	}

	if response.Value != "started" {
		return bosherr.Errorf("Failed to start agent services with response: '%s'", response)
	}

	return nil
}

func (c *agentClient) GetState() (State, error) {
	var response StateResponse
	err := c.agentRequest.Send("get_state", []interface{}{}, &response)
	if err != nil {
		return State{}, bosherr.WrapError(err, "Sending get_state to the agent")
	}

	return response.Value, nil
}

func (c *agentClient) ListDisk() ([]string, error) {
	var response ListResponse
	err := c.agentRequest.Send("list_disk", []interface{}{}, &response)
	if err != nil {
		return []string{}, bosherr.WrapError(err, "Sending 'list_disk' to the agent")
	}

	return response.Value, nil
}

func (c *agentClient) MountDisk(diskCID string) error {
	return c.sendAsyncTaskMessage("mount_disk", []interface{}{diskCID})
}

func (c *agentClient) UnmountDisk(diskCID string) error {
	return c.sendAsyncTaskMessage("unmount_disk", []interface{}{diskCID})
}

func (c *agentClient) MigrateDisk() error {
	return c.sendAsyncTaskMessage("migrate_disk", []interface{}{})
}

func (c *agentClient) sendAsyncTaskMessage(method string, arguments []interface{}) error {
	var response TaskResponse
	err := c.agentRequest.Send(method, arguments, &response)
	if err != nil {
		return bosherr.WrapErrorf(err, "Sending '%s' to the agent", method)
	}

	agentTaskID, err := response.TaskID()
	if err != nil {
		return bosherr.WrapError(err, "Getting agent task id")
	}

	getTaskRetryable := bmretrystrategy.NewRetryable(func() (bool, error) {
		var response TaskResponse
		err = c.agentRequest.Send("get_task", []interface{}{agentTaskID}, &response)
		if err != nil {
			return false, bosherr.WrapError(err, "Sending 'get_task' to the agent")
		}

		taskState, err := response.TaskState()
		if err != nil {
			return false, bosherr.WrapError(err, "Getting task state")
		}

		if taskState != "running" {
			return true, nil
		}

		return true, bosherr.Errorf("Task %s is still running", method)
	})

	getTaskRetryStrategy := bmretrystrategy.NewUnlimitedRetryStrategy(c.getTaskDelay, getTaskRetryable, c.logger)
	return getTaskRetryStrategy.Try()
}