package prom2click

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/constant"
	"github.com/gotomicro/ego/core/elog"
	"github.com/stretchr/testify/assert"
)

func TestContextClientIP(t *testing.T) {
	router := DefaultContainer().Build(WithTrustedPlatform("X-Forwarded-For"))
	router.GET("/", func(c *gin.Context) {
		assert.Equal(t, "10.10.10.11", c.ClientIP())
	})

	performRequest(router, "GET", "/", header{
		Key:   "X-Forwarded-For",
		Value: "10.10.10.11",
	})

	router3 := DefaultContainer().Build(WithTrustedPlatform("X-Forwarded-For"))
	router3.GET("/", func(c *gin.Context) {
		assert.NotEqual(t, "10.10.10.12", c.ClientIP())
	})

	performRequest(router3, "GET", "/", header{
		Key:   "X-Forwarded-For",
		Value: "10.10.10.11,10.10.10.12",
	})

	router2 := DefaultContainer().Build(WithTrustedPlatform("X-CUSTOM-CDN-IP"))
	router2.GET("/", func(c *gin.Context) {
		assert.Equal(t, "10.10.10.12", c.ClientIP())
	})

	performRequest(router2, "GET", "/", header{
		Key:   "X-CUSTOM-CDN-IP",
		Value: "10.10.10.12",
	})
}

func TestNewComponent(t *testing.T) {
	cfg := config{
		Host:    "0.0.0.0",
		Port:    9006,
		Network: "tcp",
	}
	cmp := newComponent("test-cmp", &cfg, elog.DefaultLogger)
	assert.Equal(t, "test-cmp", cmp.Name())
	assert.Equal(t, "server.prom2click", cmp.PackageName())
	assert.Equal(t, "0.0.0.0:9006", cmp.config.Address())

	assert.NoError(t, cmp.Init())

	info := cmp.Info()
	assert.NotEmpty(t, info.Name)
	assert.Equal(t, "http", info.Scheme)
	assert.Equal(t, "[::]:9006", info.Address)
	assert.Equal(t, constant.ServiceProvider, info.Kind)

	// err = cmp.Start()
	go func() {
		assert.NoError(t, cmp.Start())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	<-ctx.Done()
	assert.NoError(t, cmp.Stop())
}

func loadConfig(t *testing.T, loadClientCert bool) *tls.Config {
	pool := x509.NewCertPool()
	ca, err := os.ReadFile("./testdata/egoServer/ca.pem")
	assert.Nil(t, err)
	assert.True(t, pool.AppendCertsFromPEM(ca))
	cf := &tls.Config{}
	cf.RootCAs = pool
	if loadClientCert {
		serverCert, err := tls.LoadX509KeyPair("./testdata/egoClient/anyClient.pem", "./testdata/egoClient/anyClient-key.pem")
		assert.Nil(t, err)
		cf.Certificates = []tls.Certificate{serverCert}
	}
	return cf
}

func TestServerTimeouts(t *testing.T) {
	timeout := 2 * time.Second
	err := testServerTimeouts(timeout)
	if err == nil {
		return
	}
	t.Logf("failed at %v: %v", timeout, err)
}

func testServerTimeouts(timeout time.Duration) error {
	reqNum := 0
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		reqNum++
		fmt.Fprintf(res, "req=%d", reqNum)
	}))
	ts.Config.ReadTimeout = timeout
	ts.Config.WriteTimeout = timeout
	ts.Start()
	defer ts.Close()

	// Hit the HTTP server successfully.
	c := ts.Client()
	t0 := time.Now()
	r, err := c.Get(ts.URL)
	if err != nil {
		return fmt.Errorf("http Get #1: %v", err)
	}
	got, err := io.ReadAll(r.Body)
	latency := time.Since(t0)
	fmt.Printf("got--------------->"+"%+v\n", got)
	fmt.Printf("latency--------------->"+"%+v\n", latency)

	expected := "req=1"
	if string(got) != expected || err != nil {
		return fmt.Errorf("Unexpected response for request #1; got %q ,%v; expected %q, nil",
			string(got), err, expected)
	}

	// Slow client that should timeout.
	t1 := time.Now()
	conn, err := net.Dial("tcp", ts.Listener.Addr().String())
	if err != nil {
		return fmt.Errorf("Dial: %v", err)
	}
	buf := make([]byte, 1)
	n, err := conn.Read(buf)
	conn.Close()
	latency = time.Since(t1)
	fmt.Printf("latency--------------->"+"%+v\n", latency)
	if n != 0 || err != io.EOF {
		return fmt.Errorf("Read = %v, %v, wanted %v, %v", n, err, 0, io.EOF)
	}
	minLatency := timeout / 5 * 4
	if latency < minLatency {
		return fmt.Errorf("got EOF after %s, want >= %s", latency, minLatency)
	}

	// Hit the HTTP server successfully again, verifying that the
	// previous slow connection didn't run our handler.  (that we
	// get "req=2", not "req=3")
	r, err = c.Get(ts.URL)
	if err != nil {
		return fmt.Errorf("http Get #2: %v", err)
	}
	got, err = io.ReadAll(r.Body)
	r.Body.Close()
	expected = "req=2"

	if string(got) != expected || err != nil {
		return fmt.Errorf("Get #2 got %q, %v, want %q, nil", string(got), err, expected)
	}
	return nil
}
