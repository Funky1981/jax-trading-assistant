package eventdecisions

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type SafetyState struct {
	RuntimeMode      string  `json:"runtimeMode"`
	AllowLiveTrading bool    `json:"allowLiveTrading"`
	ExecutionEnabled bool    `json:"executionEnabled"`
	ExecutionWorker  bool    `json:"executionWorkerEnabled"`
	BrokerExecution  bool    `json:"brokerExecutionAllowed"`
	MaximumLeverage  float64 `json:"maximumLeverage"`
}

func ReadSafetyState(lookup func(string) (string, bool)) (SafetyState, error) {
	required := []string{"JAX_RUNTIME_MODE", "ALLOW_LIVE_TRADING", "EXECUTION_ENABLED", "EXECUTION_INSTRUCTION_WORKER_ENABLED", "BROKER_EXECUTION_ALLOWED", "MAX_LEVERAGE"}
	values := map[string]string{}
	for _, key := range required {
		value, ok := lookup(key)
		if !ok || strings.TrimSpace(value) == "" {
			return SafetyState{}, fmt.Errorf("missing safety state: %s", key)
		}
		values[key] = strings.TrimSpace(value)
	}
	parseBool := func(key string) (bool, error) {
		value, err := strconv.ParseBool(values[key])
		if err != nil {
			return false, fmt.Errorf("invalid safety state %s=%q", key, values[key])
		}
		return value, nil
	}
	live, err := parseBool("ALLOW_LIVE_TRADING")
	if err != nil {
		return SafetyState{}, err
	}
	executionEnabled, err := parseBool("EXECUTION_ENABLED")
	if err != nil {
		return SafetyState{}, err
	}
	worker, err := parseBool("EXECUTION_INSTRUCTION_WORKER_ENABLED")
	if err != nil {
		return SafetyState{}, err
	}
	broker, err := parseBool("BROKER_EXECUTION_ALLOWED")
	if err != nil {
		return SafetyState{}, err
	}
	leverage, err := strconv.ParseFloat(values["MAX_LEVERAGE"], 64)
	if err != nil || leverage <= 0 {
		return SafetyState{}, fmt.Errorf("invalid safety state MAX_LEVERAGE=%q", values["MAX_LEVERAGE"])
	}
	state := SafetyState{RuntimeMode: strings.ToLower(values["JAX_RUNTIME_MODE"]), AllowLiveTrading: live, ExecutionEnabled: executionEnabled, ExecutionWorker: worker, BrokerExecution: broker, MaximumLeverage: leverage}
	if state.RuntimeMode != "paper" || state.AllowLiveTrading || state.ExecutionEnabled || state.ExecutionWorker || state.BrokerExecution || state.MaximumLeverage > 1 {
		return state, fmt.Errorf("unsafe replay state: paper mode with live, execution, worker and broker disabled and leverage <=1x is required")
	}
	return state, nil
}

func ReadEnvironmentSafetyState() (SafetyState, error) {
	return ReadSafetyState(os.LookupEnv)
}
