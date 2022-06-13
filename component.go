package prom2click

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/constant"
	"github.com/gotomicro/ego/core/elog"
	"github.com/gotomicro/ego/server"
	"github.com/prometheus/prometheus/prompb"
	"github.com/prometheus/prometheus/storage/remote"
)

// PackageName 包名
const PackageName = "server.prom2click"

// Component ...
type Component struct {
	*gin.Engine
	mu       sync.Mutex
	name     string
	config   *config
	logger   *elog.Component
	Server   *http.Server
	listener net.Listener
	writer   *promWriter
	reader   *promReader
}

func newComponent(name string, config *config, logger *elog.Component) *Component {
	var err error
	gin.SetMode(config.Mode)
	comp := &Component{
		name:     name,
		config:   config,
		logger:   logger,
		Engine:   gin.New(),
		listener: nil,
	}
	comp.writer, err = NewWriter(config)
	if err != nil {
		elog.Panic("p2c writer fail", elog.FieldErr(err))
		return nil
	}
	comp.reader, err = NewReader(config)
	if err != nil {
		elog.Panic("p2c reader fail", elog.FieldErr(err))
		return nil
	}
	// 设置信任的header头
	comp.Engine.TrustedPlatform = config.TrustedPlatform

	comp.route()
	return comp
}

func (c *Component) route() {
	c.Engine.Any(c.config.ClickhouseHTTPWritePath, func(ctx *gin.Context) {
		prompbReq, err := remote.DecodeWriteRequest(ctx.Request.Body)
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}
		c.writer.process(prompbReq)
	})

	c.Engine.POST(c.config.ClickhouseHTTPReadPath, func(ctx *gin.Context) {
		prompbReq, err := remote.DecodeReadRequest(ctx.Request)
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}
		var resp *prompb.ReadResponse
		resp, err = c.reader.Read(prompbReq)
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}
		// ctx.Header("Content-Type", "application/x-protobuf")
		// ctx.Header("Content-Encoding", "snappy")
		err = remote.EncodeReadResponse(resp, ctx.Writer)
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}
	})
}

// Name 配置名称
func (c *Component) Name() string {
	return c.name
}

// PackageName 包名
func (c *Component) PackageName() string {
	return PackageName
}

// Init 初始化
func (c *Component) Init() error {
	var err error
	c.listener, err = net.Listen(c.config.Network, c.config.Address())
	if err != nil {
		c.logger.Panic("new prom2click server err", elog.FieldErrKind("listen err"), elog.FieldErr(err))
	}
	c.config.Port = c.listener.Addr().(*net.TCPAddr).Port
	return nil
}

// Start implements server.Component interface.
func (c *Component) Start() error {
	for _, route := range c.Engine.Routes() {
		c.logger.Info("add route", elog.FieldMethod(route.Method), elog.String("path", route.Path))
	}

	// 因为start和stop在多个goroutine里，需要对Server上写锁
	c.mu.Lock()
	c.Server = &http.Server{
		Addr:              c.config.Address(),
		Handler:           c,
		ReadHeaderTimeout: c.config.ServerReadHeaderTimeout,
		ReadTimeout:       c.config.ServerReadTimeout,
		WriteTimeout:      c.config.ServerWriteTimeout,
	}
	c.mu.Unlock()

	c.writer.Start()

	var err error
	err = c.Server.Serve(c.listener)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// Stop implements server.Component interface
// it will terminate gin server immediately
func (c *Component) Stop() error {
	close(c.writer.requests)
	c.writer.Wait()

	wchan := make(chan struct{})
	go func() {
		c.writer.Wait()
		close(wchan)
	}()

	select {
	case <-wchan:
		c.logger.Info("Prom2Click shutdown cleanly..")
	// All done!
	case <-time.After(10 * time.Second):
		c.logger.Info("Prom2Click shutdown timed out, samples will be lost..")
	}

	c.mu.Lock()
	err := c.Server.Close()
	c.mu.Unlock()
	return err
}

// GracefulStop implements server.Component interface
// it will stop gin server gracefully
func (c *Component) GracefulStop(ctx context.Context) error {
	close(c.writer.requests)
	c.writer.Wait()

	wchan := make(chan struct{})
	go func() {
		c.writer.Wait()
		close(wchan)
	}()

	select {
	case <-wchan:
		c.logger.Info("Prom2Click shutdown cleanly..")
	// All done!
	case <-time.After(10 * time.Second):
		c.logger.Info("Prom2Click shutdown timed out, samples will be lost..")
	}

	c.mu.Lock()
	err := c.Server.Shutdown(ctx)
	c.mu.Unlock()
	return err
}

// Info returns server info, used by governor and consumer balancer
func (c *Component) Info() *server.ServiceInfo {
	info := server.ApplyOptions(
		server.WithScheme("http"),
		server.WithAddress(c.listener.Addr().String()),
		server.WithKind(constant.ServiceProvider),
	)
	return &info
}

func commentUniqKey(method, path string) string {
	return fmt.Sprintf("%s@%s", strings.ToLower(method), path)
}
