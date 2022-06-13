package prom2click

import (
	"time"

	"github.com/gotomicro/ego/core/elog"
)

// Option 可选项
type Option func(c *Container)

// WithHost 设置host
func WithHost(host string) Option {
	return func(c *Container) {
		c.config.Host = host
	}
}

// WithPort 设置port
func WithPort(port int) Option {
	return func(c *Container) {
		c.config.Port = port
	}
}

// WithNetwork 设置network
func WithNetwork(network string) Option {
	return func(c *Container) {
		c.config.Network = network
	}
}

// WithTrustedPlatform 信任的Header头，获取客户端IP地址
func WithTrustedPlatform(trustedPlatform string) Option {
	return func(c *Container) {
		c.config.TrustedPlatform = trustedPlatform
	}
}

// WithLogger 信任的Header头，获取客户端IP地址
func WithLogger(logger *elog.Component) Option {
	return func(c *Container) {
		c.logger = logger
	}
}

// WithServerReadTimeout 设置超时时间
func WithServerReadTimeout(timeout time.Duration) Option {
	return func(c *Container) {
		c.config.ServerReadTimeout = timeout
	}
}

// WithServerReadHeaderTimeout 设置超时时间
func WithServerReadHeaderTimeout(timeout time.Duration) Option {
	return func(c *Container) {
		c.config.ServerReadHeaderTimeout = timeout
	}
}

// WithServerWriteTimeout 设置超时时间
func WithServerWriteTimeout(timeout time.Duration) Option {
	return func(c *Container) {
		c.config.ServerWriteTimeout = timeout
	}
}

// WithContextTimeout 设置port
func WithContextTimeout(timeout time.Duration) Option {
	return func(c *Container) {
		c.config.ContextTimeout = timeout
	}
}
