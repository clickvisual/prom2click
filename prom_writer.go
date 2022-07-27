package prom2click

import (
	"database/sql"
	"fmt"
	"sort"
	"sync"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"
)

type promRequest struct {
	name string
	tags []string
	val  float64
	ts   time.Time
}

var insertSQL = `INSERT INTO %s.%s
	(date, name, tags, val, ts)
	VALUES	(?, ?, ?, ?, ?)`

type promWriter struct {
	config   *config
	requests chan *promRequest
	wg       sync.WaitGroup
	db       *sql.DB
	tx       prometheus.Counter
	ko       prometheus.Counter
	test     prometheus.Counter
	timings  prometheus.Histogram
	rx       prometheus.Counter
}

func NewWriter(conf *config) (*promWriter, error) {
	var err error
	w := new(promWriter)
	w.config = conf
	w.requests = make(chan *promRequest, conf.ClickhouseChanSize)
	w.db, err = sql.Open("clickhouse", w.config.ClickhouseDSN)
	if err != nil {
		fmt.Printf("Error connecting to clickhouse: %s\n", err.Error())
		return w, err
	}

	w.tx = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sent_samples_total",
			Help: "Total number of processed samples sent to remote storage.",
		},
	)

	w.ko = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "failed_samples_total",
			Help: "Total number of processed samples which failed on send to remote storage.",
		},
	)

	w.test = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "prometheus_remote_storage_sent_batch_duration_seconds_bucket_test",
			Help: "Test metric to ensure backfilled metrics are readable via prometheus.",
		},
	)

	w.timings = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "sent_batch_duration_seconds",
			Help:    "Duration of sample batch send calls to the remote storage.",
			Buckets: prometheus.DefBuckets,
		},
	)

	w.rx = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "received_samples_total",
			Help: "Total number of received samples.",
		},
	)

	prometheus.MustRegister(w.rx)
	prometheus.MustRegister(w.tx)
	prometheus.MustRegister(w.ko)
	prometheus.MustRegister(w.test)
	prometheus.MustRegister(w.timings)
	return w, nil
}

func (w *promWriter) process(req *prompb.WriteRequest) {
	for _, series := range req.Timeseries {
		w.rx.Add(float64(len(series.Samples)))
		var (
			name string
			tags []string
		)

		for _, label := range series.Labels {
			if model.LabelName(label.Name) == model.MetricNameLabel {
				name = label.Value
			}
			// store tags in <key>=<value> format
			// allows for has(tags, "key=val") searches
			// probably impossible/difficult to do regex searches on tags
			t := fmt.Sprintf("%s=%s", label.Name, label.Value)
			tags = append(tags, t)
		}

		for _, sample := range series.Samples {
			p2c := new(promRequest)
			p2c.name = name
			p2c.ts = time.Unix(sample.Timestamp/1000, 0)
			p2c.val = sample.Value
			p2c.tags = tags
			w.requests <- p2c
		}

	}
}

func (w *promWriter) Start() {

	go func() {
		w.wg.Add(1)
		fmt.Println("Writer starting..")
		sql := fmt.Sprintf(insertSQL, w.config.ClickhouseDB, w.config.ClickhouseTable)
		ok := true
		for ok {
			w.test.Add(1)
			// get next batch of requests
			var reqs []*promRequest

			tstart := time.Now()
			for i := 0; i < w.config.ClickhouseBatch; i++ {
				var req *promRequest
				// get requet and also check if channel is closed
				req, ok = <-w.requests
				if !ok {
					fmt.Println("Writer stopping..")
					break
				}
				reqs = append(reqs, req)
			}

			// ensure we have something to send..
			nmetrics := len(reqs)
			if nmetrics < 1 {
				continue
			}

			// post them to db all at once
			tx, err := w.db.Begin()
			if err != nil {
				fmt.Printf("Error: begin transaction: %s\n", err.Error())
				w.ko.Add(1.0)
				continue
			}

			// build statements
			smt, err := tx.Prepare(sql)
			for _, req := range reqs {
				if err != nil {
					fmt.Printf("Error: prepare statement: %s\n", err.Error())
					w.ko.Add(1.0)
					continue
				}

				// ensure tags are inserted in the same order each time
				// possibly/probably impacts indexing?
				sort.Strings(req.tags)
				_, err = smt.Exec(req.ts, req.name, req.tags, req.val, req.ts)

				if err != nil {
					fmt.Printf("Error: statement exec: %s\n", err.Error())
					w.ko.Add(1.0)
				}
			}

			// commit and record metrics
			if err = tx.Commit(); err != nil {
				fmt.Printf("Error: commit failed: %s\n", err.Error())
				w.ko.Add(1.0)
			} else {
				w.tx.Add(float64(nmetrics))
				w.timings.Observe(float64(time.Since(tstart)))
			}

		}
		fmt.Println("Writer stopped..")
		w.wg.Done()
	}()
}

func (w *promWriter) Wait() {
	w.wg.Wait()
}
