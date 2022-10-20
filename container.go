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
