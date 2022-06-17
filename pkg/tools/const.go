package tools

import "time"

type ServiceTimerMode string

const (
	ServiceTimerModeCreate      ServiceTimerMode = "ServiceTimerModeCreate"
	ServiceTimerModeCreateAsync ServiceTimerMode = "ServiceTimerModeCreateAsync"
	ServiceTimerModeDelete      ServiceTimerMode = "ServiceTimerModeDelete"
	ServiceTimerModeDeleteAll   ServiceTimerMode = "ServiceTimerModeDeleteAll"
)

const (
	KubeConfigPathEnv = "KUBECONFIG"
)

type ObjectFinishTime struct {
	Name              string
	Started, Finished time.Time
}
