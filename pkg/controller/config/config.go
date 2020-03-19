package config

import (
	"time"

	"github.com/lyft/flytestdlib/config"
	"k8s.io/apimachinery/pkg/types"
)

//go:generate pflags Config --default-var=defaultConfig

const configSectionKey = "propeller"

var (
	configSection = config.MustRegisterSection(configSectionKey, defaultConfig)

	defaultConfig = &Config{
		MaxWorkflowRetries:  5,
		MaxDatasetSizeBytes: 10 * 1024 * 1024,
		Queue: CompositeQueueConfig{
			Type: CompositeQueueSimple,
			Queue: WorkqueueConfig{
				Type:      WorkqueueTypeDefault,
				BaseDelay: config.Duration{Duration: time.Second * 10},
				MaxDelay:  config.Duration{Duration: time.Second * 10},
				Rate:      10,
				Capacity:  100,
			},
		},
		KubeConfig: KubeClientConfig{
			QPS:     5,
			Burst:   10,
			Timeout: config.Duration{Duration: 0},
		},
		LeaderElection: LeaderElectionConfig{
			Enabled:       false,
			LeaseDuration: config.Duration{Duration: time.Second * 15},
			RenewDeadline: config.Duration{Duration: time.Second * 10},
			RetryPeriod:   config.Duration{Duration: time.Second * 2},
		},
		NodeConfig: NodeConfig{
			DefaultDeadlines: DefaultDeadlines{
				DefaultNodeExecutionDeadline:  config.Duration{Duration: time.Hour * 48},
				DefaultNodeActiveDeadline:     config.Duration{Duration: time.Hour * 48},
				DefaultWorkflowActiveDeadline: config.Duration{Duration: time.Hour * 72},
			},
			MaxNodeRetriesOnSystemFailures: 3,
			InterruptibleFailureThreshold:  1,
		},
	}
)

// NOTE: when adding new fields, do not mark them as "omitempty" if it's desirable to read the value from env variables.
// Config that uses the flytestdlib Config module to generate commandline and load config files. This configuration is
// the base configuration to start propeller
type Config struct {
	KubeConfigPath      string               `json:"kube-config" pflag:",Path to kubernetes client config file."`
	MasterURL           string               `json:"master"`
	Workers             int                  `json:"workers" pflag:"2,Number of threads to process workflows"`
	WorkflowReEval      config.Duration      `json:"workflow-reeval-duration" pflag:"\"30s\",Frequency of re-evaluating workflows"`
	DownstreamEval      config.Duration      `json:"downstream-eval-duration" pflag:"\"60s\",Frequency of re-evaluating downstream tasks"`
	LimitNamespace      string               `json:"limit-namespace" pflag:"\"all\",Namespaces to watch for this propeller"`
	ProfilerPort        config.Port          `json:"prof-port" pflag:"\"10254\",Profiler port"`
	MetadataPrefix      string               `json:"metadata-prefix,omitempty" pflag:",MetadataPrefix should be used if all the metadata for Flyte executions should be stored under a specific prefix in CloudStorage. If not specified, the data will be stored in the base container directly."`
	Queue               CompositeQueueConfig `json:"queue,omitempty" pflag:",Workflow workqueue configuration, affects the way the work is consumed from the queue."`
	MetricsPrefix       string               `json:"metrics-prefix" pflag:"\"flyte:\",An optional prefix for all published metrics."`
	EnableAdminLauncher bool                 `json:"enable-admin-launcher" pflag:"false, Enable remote Workflow launcher to Admin"`
	MaxWorkflowRetries  int                  `json:"max-workflow-retries" pflag:"50,Maximum number of retries per workflow"`
	MaxTTLInHours       int                  `json:"max-ttl-hours" pflag:"23,Maximum number of hours a completed workflow should be retained. Number between 1-23 hours"`
	GCInterval          config.Duration      `json:"gc-interval" pflag:"\"30m\",Run periodic GC every 30 minutes"`
	LeaderElection      LeaderElectionConfig `json:"leader-election,omitempty" pflag:",Config for leader election."`
	PublishK8sEvents    bool                 `json:"publish-k8s-events" pflag:",Enable events publishing to K8s events API."`
	MaxDatasetSizeBytes int64                `json:"max-output-size-bytes" pflag:",Maximum size of outputs per task"`
	KubeConfig          KubeClientConfig     `json:"kube-client-config" pflag:",Configuration to control the Kubernetes client"`
	NodeConfig          NodeConfig           `json:"node-config,omitempty" pflag:",config for a workflow node"`
}

type KubeClientConfig struct {
	// QPS indicates the maximum QPS to the master from this client.
	// If it's zero, the created RESTClient will use DefaultQPS: 5
	QPS float32 `json:"qps" pflag:"-,Max QPS to the master for requests to KubeAPI. 0 defaults to 5."`
	// Maximum burst for throttle.
	// If it's zero, the created RESTClient will use DefaultBurst: 10.
	Burst int `json:"burst" pflag:",Max burst rate for throttle. 0 defaults to 10"`
	// The maximum length of time to wait before giving up on a server request. A value of zero means no timeout.
	Timeout config.Duration `json:"timeout" pflag:",Max duration allowed for every request to KubeAPI before giving up. 0 implies no timeout."`
}

type CompositeQueueType = string

const (
	CompositeQueueSimple CompositeQueueType = "simple"
	CompositeQueueBatch  CompositeQueueType = "batch"
)

type CompositeQueueConfig struct {
	Type             CompositeQueueType `json:"type" pflag:",Type of composite queue to use for the WorkQueue"`
	Queue            WorkqueueConfig    `json:"queue,omitempty" pflag:",Workflow workqueue configuration, affects the way the work is consumed from the queue."`
	Sub              WorkqueueConfig    `json:"sub-queue,omitempty" pflag:",SubQueue configuration, affects the way the nodes cause the top-level Work to be re-evaluated."`
	BatchingInterval config.Duration    `json:"batching-interval" pflag:",Duration for which downstream updates are buffered"`
	BatchSize        int                `json:"batch-size" pflag:"-1,Number of downstream triggered top-level objects to re-enqueue every duration. -1 indicates all available."`
}

type WorkqueueType = string

const (
	WorkqueueTypeDefault                       WorkqueueType = "default"
	WorkqueueTypeBucketRateLimiter             WorkqueueType = "bucket"
	WorkqueueTypeExponentialFailureRateLimiter WorkqueueType = "expfailure"
	WorkqueueTypeMaxOfRateLimiter              WorkqueueType = "maxof"
)

// prototypical configuration to configure a workqueue. We may want to generalize this in a package like k8sutils
type WorkqueueConfig struct {
	// Refer to https://github.com/kubernetes/client-go/tree/master/util/workqueue
	Type      WorkqueueType   `json:"type" pflag:",Type of RateLimiter to use for the WorkQueue"`
	BaseDelay config.Duration `json:"base-delay" pflag:",base backoff delay for failure"`
	MaxDelay  config.Duration `json:"max-delay" pflag:",Max backoff delay for failure"`
	Rate      int64           `json:"rate" pflag:",Bucket Refill rate per second"`
	Capacity  int             `json:"capacity" pflag:",Bucket capacity as number of items"`
}

// configuration for a node
type NodeConfig struct {
	DefaultDeadlines               DefaultDeadlines `json:"default-deadlines,omitempty" pflag:",Default value for timeouts"`
	MaxNodeRetriesOnSystemFailures int64            `json:"max-node-retries-system-failures" pflag:"2,Maximum number of retries per node for node failure due to infra issues"`
	InterruptibleFailureThreshold  int64            `json:"interruptible-failure-threshold" pflag:"1,number of failures for a node to be still considered interruptible'"`
}

// Contains default values for timeouts
type DefaultDeadlines struct {
	DefaultNodeExecutionDeadline  config.Duration `json:"node-execution-deadline" pflag:",Default value of node execution timeout"`
	DefaultNodeActiveDeadline     config.Duration `json:"node-active-deadline" pflag:",Default value of node timeout"`
	DefaultWorkflowActiveDeadline config.Duration `json:"workflow-active-deadline" pflag:",Default value of workflow timeout"`
}

// Contains leader election configuration.
type LeaderElectionConfig struct {
	// Enable or disable leader election.
	Enabled bool `json:"enabled" pflag:",Enables/Disables leader election."`

	// Determines the name of the configmap that leader election will use for holding the leader lock.
	LockConfigMap types.NamespacedName `json:"lock-config-map" pflag:",ConfigMap namespace/name to use for resource lock."`

	// Duration that non-leader candidates will wait to force acquire leadership. This is measured against time of last
	// observed ack
	LeaseDuration config.Duration `json:"lease-duration" pflag:",Duration that non-leader candidates will wait to force acquire leadership. This is measured against time of last observed ack."`

	// RenewDeadline is the duration that the acting master will retry refreshing leadership before giving up.
	RenewDeadline config.Duration `json:"renew-deadline" pflag:",Duration that the acting master will retry refreshing leadership before giving up."`

	// RetryPeriod is the duration the LeaderElector clients should wait between tries of actions.
	RetryPeriod config.Duration `json:"retry-period" pflag:",Duration the LeaderElector clients should wait between tries of actions."`
}

// Extracts the Configuration from the global config module in flytestdlib and returns the corresponding type-casted object.
// TODO What if the type is incorrect?
func GetConfig() *Config {
	return configSection.GetConfig().(*Config)
}

func MustRegisterSubSection(subSectionKey string, section config.Config) config.Section {
	return configSection.MustRegisterSection(subSectionKey, section)
}
