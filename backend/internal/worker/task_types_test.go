package worker

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMonitorAgentTask(t *testing.T) {
	payload := MonitorAgentPayload{
		CaseID: "case-1", MonitorBlockID: "block-1", Symbol: "005930",
	}
	task, err := NewMonitorAgentTask(payload)
	require.NoError(t, err)
	assert.Equal(t, TypeMonitorAgent, task.Type())
	var decoded MonitorAgentPayload
	require.NoError(t, json.Unmarshal(task.Payload(), &decoded))
	assert.Equal(t, "005930", decoded.Symbol)
}

func TestNewDSLPollerTask(t *testing.T) {
	task := NewDSLPollerTask()
	assert.Equal(t, TypeDSLPoller, task.Type())
}

func TestNewLifecycleTask(t *testing.T) {
	payload := LifecyclePayload{CaseID: "case-1", Action: "CLOSE_SUCCESS", Reason: "목표가 도달"}
	task, err := NewLifecycleTask(payload)
	require.NoError(t, err)
	assert.Equal(t, TypeMonitorLifecycle, task.Type())
}
