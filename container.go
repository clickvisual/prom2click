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
		cfg := configs[index]
		if cfg.Host != "" {
			c.config.Host = cfg.Host
		}
		if cfg.Port != 0 {
			c.config.Port = cfg.Port
		}
		if cfg.ClickhouseDSN != "" {
			c.config.ClickhouseDSN = cfg.ClickhouseDSN
		}
		if cfg.ClickhouseDB != "" {
			c.config.ClickhouseDB = cfg.ClickhouseDB
		}
		if cfg.ClickhouseTable != "" {
			c.config.ClickhouseTable = cfg.ClickhouseTable
		}
		if cfg.ClickhouseBatch != 0 {
			c.config.ClickhouseBatch = cfg.ClickhouseBatch
		}
		if cfg.ClickhouseMaxSamples != 0 {
			c.config.ClickhouseMaxSamples = cfg.ClickhouseMaxSamples
		}
		if cfg.ClickhouseMinPeriod != 0 {
			c.config.ClickhouseMinPeriod = cfg.ClickhouseMinPeriod
		}
		if cfg.ClickhouseQuantile != 0 {
			c.config.ClickhouseQuantile = cfg.ClickhouseQuantile
		}
		if cfg.ClickhouseHTTPWritePath != "" {
			c.config.ClickhouseHTTPWritePath = cfg.ClickhouseHTTPWritePath
		}
		if cfg.ClickhouseHTTPReadPath != "" {
			c.config.ClickhouseHTTPReadPath = cfg.ClickhouseHTTPReadPath
		}
		if cfg.ClickhouseChanSize != 0 {
			c.config.ClickhouseChanSize = cfg.ClickhouseChanSize
		}
		if cfg.ServerReadTimeout != 0 {
			c.config.ServerReadTimeout = cfg.ServerReadTimeout
		}
		if cfg.ServerReadHeaderTimeout != 0 {
			c.config.ServerReadHeaderTimeout = cfg.ServerReadHeaderTimeout
		}
		if cfg.ServerWriteTimeout != 0 {
			c.config.ServerWriteTimeout = cfg.ServerWriteTimeout
		}
		if cfg.ContextTimeout != 0 {
			c.config.ContextTimeout = cfg.ContextTimeout
		}
		if cfg.EnableMetricInterceptor != nil {
			c.config.EnableMetricInterceptor = cfg.EnableMetricInterceptor
		}
		if cfg.SlowLogThreshold != 0 {
			c.config.SlowLogThreshold = cfg.SlowLogThreshold
		}
		if cfg.EnableAccessInterceptor != nil {
			c.config.EnableAccessInterceptor = cfg.EnableAccessInterceptor
		}
		if cfg.EnableAccessInterceptorReq != nil {
			c.config.EnableAccessInterceptorReq = cfg.EnableAccessInterceptorReq
		}
		if cfg.EnableAccessInterceptorRes != nil {
			c.config.EnableAccessInterceptorRes = cfg.EnableAccessInterceptorRes
		}
		if cfg.EnableTrustedCustomHeader != nil {
			c.config.EnableTrustedCustomHeader = cfg.EnableTrustedCustomHeader
		}
		if cfg.TrustedPlatform != "" {
			c.config.TrustedPlatform = cfg.TrustedPlatform
		}

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
	if c.config.EnableMetricInterceptor != nil && *c.config.EnableMetricInterceptor {
		server.Use(metricServerInterceptor())
	}
	return server
}
