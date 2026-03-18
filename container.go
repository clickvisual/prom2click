package prom2click

import (
	"fmt"

	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/core/elog"
)

// Container 容器
type Container struct {
	config *config
	name   string
	logger *elog.Component
}

// DefaultContainer 默认容器
func DefaultContainer() *Container {
	return &Container{
		config: DefaultConfig(),
		logger: elog.EgoLogger.With(elog.FieldComponent(PackageName)),
	}
}

// Load 加载配置key
func Load(key string) *Container {
	c := DefaultContainer()
	c.logger = c.logger.With(elog.FieldComponentName(key))
	if err := econf.UnmarshalKey(key, &c.config); err != nil {
		c.logger.Panic("parse config error", elog.FieldErr(err), elog.FieldKey(key))
		return c
	}
	c.name = key
	return c
}

func LoadBatch(key string) []*Container {
	containers := make([]*Container, 0)
	configs := make([]*config, 0)
	if err := econf.UnmarshalKey(key, &configs); err != nil {
		elog.EgoLogger.With(elog.FieldComponent(PackageName)).Panic("parse config error", elog.FieldErr(err), elog.FieldKey(key))
		return nil
	}
	for index := range configs {
		c := DefaultContainer()
		c.logger = c.logger.With(elog.FieldComponentName(key))
		c.config.Host = configs[index].Host
		c.config.Port = configs[index].Port
		c.config.ClickhouseDSN = configs[index].ClickhouseDSN
		c.config.ClickhouseDB = configs[index].ClickhouseDB
		c.config.ClickhouseTable = configs[index].ClickhouseTable
		c.config.ClickhouseBatch = configs[index].ClickhouseBatch
		c.config.ClickhouseMaxSamples = configs[index].ClickhouseMaxSamples
		c.config.ClickhouseMinPeriod = configs[index].ClickhouseMinPeriod
		c.config.ClickhouseQuantile = configs[index].ClickhouseQuantile
		c.config.ClickhouseHTTPWritePath = configs[index].ClickhouseHTTPWritePath
		c.config.ClickhouseHTTPReadPath = configs[index].ClickhouseHTTPReadPath
		c.config.ClickhouseChanSize = configs[index].ClickhouseChanSize
		c.config.ServerReadTimeout = configs[index].ServerReadTimeout
		c.config.ServerReadHeaderTimeout = configs[index].ServerReadHeaderTimeout
		c.config.ServerWriteTimeout = configs[index].ServerWriteTimeout
		c.config.ContextTimeout = configs[index].ContextTimeout
		c.config.EnableMetricInterceptor = configs[index].EnableMetricInterceptor
		c.config.SlowLogThreshold = configs[index].SlowLogThreshold
		c.config.EnableAccessInterceptor = configs[index].EnableAccessInterceptor
		c.config.EnableAccessInterceptorReq = configs[index].EnableAccessInterceptorReq
		c.config.EnableAccessInterceptorRes = configs[index].EnableAccessInterceptorRes
		c.config.EnableTrustedCustomHeader = configs[index].EnableTrustedCustomHeader
		c.config.TrustedPlatform = configs[index].TrustedPlatform

		c.name = fmt.Sprintf("%s_%d", key, index)
		containers = append(containers, c)
	}
	return containers
}

// Build 构建组件
func (c *Container) Build(options ...Option) *Component {
	for _, option := range options {
		option(c)
	}
	server := newComponent(c.name, c.config, c.logger)
	server.Use(c.defaultServerInterceptor())
	if c.config.ContextTimeout > 0 {
		server.Use(timeoutMiddleware(c.config.ContextTimeout))
	}
	if c.config.EnableMetricInterceptor {
		server.Use(metricServerInterceptor())
	}
	return server
}
