package pipelines

import (
	"fmt"
	"strings"
)

type Pipeline interface {
	Name() string
	Run() error
	Stop()
	Close()
}

////

const DATA_BUCKET_FOR_ALL_PLUGIN_INSTANCE = "*"

type PluginPreparationFunc func()

type PipelineContext interface {
	PipelineName() string
	PluginNames() []string
	Parallelism() uint16
	Statistics() PipelineStatistics
	DataBucket(pluginName, pluginInstanceId string) PipelineContextDataBucket
	DeleteBucket(pluginName, pluginInstanceId string) PipelineContextDataBucket
	PreparePlugin(pluginName string, fun PluginPreparationFunc)
	Close()
}

////

type DefaultValueFunc func() interface{}

type PipelineContextDataBucket interface {
	BindData(key, value interface{}) (interface{}, error)
	QueryData(key interface{}) interface{}
	QueryDataWithBindDefault(key interface{}, defaultValueFunc DefaultValueFunc) (interface{}, error)
	UnbindData(key interface{}) interface{}
}

////

type StatisticsKind string

const (
	SuccessStatistics StatisticsKind = "SuccessStatistics"
	FailureStatistics StatisticsKind = "FailureStatistics"
	AllStatistics     StatisticsKind = "AllStatistics"

	STATISTICS_INDICATOR_FOR_ALL_PLUGIN_INSTANCE = "*"
)

type StatisticsIndicatorEvaluator func(name, indicatorName string) (interface{}, error)

type PipelineThroughputRateUpdated func(name string, latestStatistics PipelineStatistics)
type PipelineExecutionSampleUpdated func(name string, latestStatistics PipelineStatistics)
type PluginThroughputRateUpdated func(name string, latestStatistics PipelineStatistics, kind StatisticsKind)
type PluginExecutionSampleUpdated func(name string, latestStatistics PipelineStatistics, kind StatisticsKind)

type PipelineStatistics interface {
	PipelineThroughputRate1() (float64, error)
	PipelineThroughputRate5() (float64, error)
	PipelineThroughputRate15() (float64, error)
	PipelineExecutionCount() (int64, error)
	PipelineExecutionTimeMax() (int64, error)
	PipelineExecutionTimeMin() (int64, error)
	PipelineExecutionTimePercentile(percentile float64) (float64, error)
	PipelineExecutionTimeStdDev() (float64, error)
	PipelineExecutionTimeVariance() (float64, error)
	PipelineExecutionTimeSum() (int64, error)

	PluginThroughputRate1(pluginName string, kind StatisticsKind) (float64, error)
	PluginThroughputRate5(pluginName string, kind StatisticsKind) (float64, error)
	PluginThroughputRate15(pluginName string, kind StatisticsKind) (float64, error)
	PluginExecutionCount(pluginName string, kind StatisticsKind) (int64, error)
	PluginExecutionTimeMax(pluginName string, kind StatisticsKind) (int64, error)
	PluginExecutionTimeMin(pluginName string, kind StatisticsKind) (int64, error)
	PluginExecutionTimePercentile(
		pluginName string, kind StatisticsKind, percentile float64) (float64, error)
	PluginExecutionTimeStdDev(pluginName string, kind StatisticsKind) (float64, error)
	PluginExecutionTimeVariance(pluginName string, kind StatisticsKind) (float64, error)
	PluginExecutionTimeSum(pluginName string, kind StatisticsKind) (int64, error)

	TaskExecutionCount(kind StatisticsKind) (uint64, error)

	PipelineIndicatorNames() []string
	PipelineIndicatorValue(indicatorName string) (interface{}, error)
	PluginIndicatorNames(pluginName string) []string
	PluginIndicatorValue(pluginName, indicatorName string) (interface{}, error)
	TaskIndicatorNames() []string
	TaskIndicatorValue(indicatorName string) (interface{}, error)

	AddPipelineThroughputRateUpdatedCallback(name string, callback PipelineThroughputRateUpdated,
		overwrite bool) (PipelineThroughputRateUpdated, bool)
	DeletePipelineThroughputRateUpdatedCallback(name string)
	DeletePipelineThroughputRateUpdatedCallbackAfterPluginDelete(name string, pluginName string)
	DeletePipelineThroughputRateUpdatedCallbackAfterPluginUpdate(name string, pluginName string)
	AddPipelineExecutionSampleUpdatedCallback(name string, callback PipelineExecutionSampleUpdated,
		overwrite bool) (PipelineExecutionSampleUpdated, bool)
	DeletePipelineExecutionSampleUpdatedCallback(name string)
	DeletePipelineExecutionSampleUpdatedCallbackAfterPluginDelete(name string, pluginName string)
	DeletePipelineExecutionSampleUpdatedCallbackAfterPluginUpdate(name string, pluginName string)
	AddPluginThroughputRateUpdatedCallback(name string, callback PluginThroughputRateUpdated,
		overwrite bool) (PluginThroughputRateUpdated, bool)
	DeletePluginThroughputRateUpdatedCallback(name string)
	DeletePluginThroughputRateUpdatedCallbackAfterPluginDelete(name string, pluginName string)
	DeletePluginThroughputRateUpdatedCallbackAfterPluginUpdate(name string, pluginName string)
	AddPluginExecutionSampleUpdatedCallback(name string, callback PluginExecutionSampleUpdated,
		overwrite bool) (PluginExecutionSampleUpdated, bool)
	DeletePluginExecutionSampleUpdatedCallback(name string)
	DeletePluginExecutionSampleUpdatedCallbackAfterPluginDelete(name string, pluginName string)
	DeletePluginExecutionSampleUpdatedCallbackAfterPluginUpdate(name string, pluginName string)

	RegisterPluginIndicator(pluginName, pluginInstanceId, indicatorName, desc string,
		evaluator StatisticsIndicatorEvaluator) (bool, error)
	UnregisterPluginIndicator(pluginName, pluginInstanceId, indicatorName string)
	UnregisterPluginIndicatorAfterPluginDelete(pluginName, pluginInstanceId, indicatorName string)
	UnregisterPluginIndicatorAfterPluginUpdate(pluginName, pluginInstanceId, indicatorName string)
}

////

type Config interface {
	PipelineName() string
	PluginNames() []string
	Parallelism() uint16
	Prepare() error
}

////

type CommonConfig struct {
	Name             string   `json:"pipeline_name"`
	Plugins          []string `json:"plugin_names"`
	ParallelismCount uint16   `json:"parallelism"` // up to 65535
}

func (c *CommonConfig) PipelineName() string {
	return c.Name
}

func (c *CommonConfig) PluginNames() []string {
	return c.Plugins
}

func (c *CommonConfig) Parallelism() uint16 {
	return c.ParallelismCount
}

func (c *CommonConfig) Prepare() error {
	if len(strings.TrimSpace(c.PipelineName())) == 0 {
		return fmt.Errorf("invalid pipeline name")
	}

	if len(c.PluginNames()) == 0 {
		return fmt.Errorf("pipeline is empty")
	}

	if c.Parallelism() < 1 {
		return fmt.Errorf("invalid pipeline parallelism")
	}

	return nil
}

// Pipelines register authority

var (
	PIPELINE_TYPES = map[string]interface{}{
		"LinearPipeline": nil,
	}
)

func ValidType(t string) bool {
	_, exists := PIPELINE_TYPES[t]
	return exists
}

func GetAllTypes() []string {
	types := make([]string, 0)
	for t := range PIPELINE_TYPES {
		types = append(types, t)
	}
	return types
}