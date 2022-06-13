package prom2click

import (
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/eflag"
	"github.com/gotomicro/ego/core/util/xtime"
)

// config HTTP config
type config struct {
	Host                       string // IP地址，默认0.0.0.0
	Port                       int    // PORT端口，默认9001
	Mode                       string // gin的模式，默认是release模式
	Network                    string
	ClickhouseDSN              string
	ClickhouseDB               string
	ClickhouseTable            string
	ClickhouseBatch            int
	ClickhouseMaxSamples       int
	ClickhouseMinPeriod        int
	ClickhouseQuantile         float64
	ClickhouseHTTPWritePath    string
	ClickhouseHTTPReadPath     string
	ClickhouseChanSize         int
	ServerReadTimeout          time.Duration // 服务端，用于读取io报文过慢的timeout，通常用于互联网网络收包过慢，如果你的go在最外层，可以使用他，默认不启用。
	ServerReadHeaderTimeout    time.Duration // 服务端，用于读取io报文过慢的timeout，通常用于互联网网络收包过慢，如果你的go在最外层，可以使用他，默认不启用。
	ServerWriteTimeout         time.Duration // 服务端，用于读取io报文过慢的timeout，通常用于互联网网络收包过慢，如果你的go在最外层，可以使用他，默认不启用。
	ContextTimeout             time.Duration // 只能用于IO操作，才能触发，默认不启用
	EnableMetricInterceptor    bool          // 是否开启监控，默认开启
	SlowLogThreshold           time.Duration // 服务慢日志，默认500ms
	EnableAccessInterceptor    bool          // 是否开启，记录请求数据
	EnableAccessInterceptorReq bool          // 是否开启记录请求参数，默认不开启
	EnableAccessInterceptorRes bool          // 是否开启记录响应参数，默认不开启
	EnableTrustedCustomHeader  bool          // 是否开启自定义header头，记录数据往链路后传递，默认不开启
	TrustedPlatform            string        // 需要用户换成自己的CDN名字，获取客户端IP地址
	mu                         sync.RWMutex  // mutex for EnableAccessInterceptorReq、EnableAccessInterceptorRes、AccessInterceptorReqResFilter、aiReqResCelPrg
}

// DefaultConfig ...
func DefaultConfig() *config {
	return &config{
		Host:                    eflag.String("host"),
		Port:                    9201,
		Mode:                    gin.ReleaseMode,
		Network:                 "tcp",
		ClickhouseDSN:           "",
		ClickhouseDB:            "metrics",
		ClickhouseTable:         "samples",
		ClickhouseBatch:         8192,
		ClickhouseMaxSamples:    8192,
		ClickhouseMinPeriod:     10,
		ClickhouseQuantile:      0.75,
		ClickhouseHTTPWritePath: "/write",
		ClickhouseHTTPReadPath:  "/read",
		ClickhouseChanSize:      8192,
		EnableMetricInterceptor: true,
		SlowLogThreshold:        xtime.Duration("500ms"),
		EnableAccessInterceptor: true,
		mu:                      sync.RWMutex{},
	}
}

// Address ...
func (config *config) Address() string {
	return fmt.Sprintf("%s:%d", config.Host, config.Port)
}
