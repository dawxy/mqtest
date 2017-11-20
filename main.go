package main

import (
	"flag"
	"math"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/cmd"
	_ "github.com/micro/go-plugins/broker/kafka"
	_ "github.com/micro/go-plugins/broker/mqtt"
	_ "github.com/micro/go-plugins/broker/nats"
	_ "github.com/micro/go-plugins/broker/nsq"
	_ "github.com/micro/go-plugins/broker/rabbitmq"
	_ "github.com/micro/go-plugins/broker/redis"
	"github.com/shouyingo/logwriter"
	"github.com/vizee/echo"
)

var (
	wg  = sync.WaitGroup{}
	min = int64(math.MaxInt64)
	max = int64(0)

	mu     sync.Mutex
	qps    = int64(0)
	sum    = int64(0)
	exited = make(chan struct{})

	optRandTopic bool
	optTopicNum  int
	pprofAddr    string
	optBroker    string
	NewBroker    func(...broker.Option) broker.Broker
)

func onServiceStat(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		switch r.FormValue("key") {
		case "loglevel":
			n, _ := strconv.Atoi(r.FormValue("value"))
			if echo.LogLevel(n) <= echo.DebugLevel {
				echo.SetLevel(echo.LogLevel(n))
			}
			w.Write([]byte("ok"))
		}
	}
}
func startPprof() {
	mux := http.NewServeMux()
	mux.HandleFunc("/service/stat", onServiceStat)
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	echo.Info("http pprof serving", echo.String("addr", pprofAddr))
	err := http.ListenAndServe(pprofAddr, mux)
	if err != nil {
		echo.Error("http pprof failed", echo.Errors("error", err))
	}
}

func main() {
	var (
		optLog    string
		optSubNum int
		optItime  int64
		optPubNum int
		optMq     string
	)

	flag.StringVar(&optLog, "log", "", "path/to/roll.log")
	flag.BoolVar(&optRandTopic, "randtopic", false, "rand pub/sub a topic")
	flag.IntVar(&optSubNum, "sub", 1, "sub num")
	flag.IntVar(&optPubNum, "pub", 1, "pub num")
	flag.IntVar(&optTopicNum, "topic", 1, "topic num (only randtopic=true)")
	flag.Int64Var(&optItime, "itime", 300, "pub sleep time(ms)")
	flag.StringVar(&optBroker, "broker", "127.0.0.1:1883", "broker addr")
	flag.StringVar(&pprofAddr, "pprof", ":0", "address:port")
	flag.StringVar(&optMq, "mq", "mqtt", "nats/nsq/kafka/rabbitmq/redis/mqtt")

	flag.Parse()
	NewBroker = cmd.DefaultBrokers[optMq]
	if NewBroker == nil {
		echo.Error("not support", echo.String("mq", optMq))
		return
	}
	if optLog != "" {
		echo.SetOutput(logwriter.New(optLog, 256*1024*1024, 32))
	}
	if pprofAddr != "" {
		go startPprof()
	}

	echo.Info("start", echo.Int("sub num", optSubNum), echo.Int("pub num", optPubNum), echo.String("broker addr", optBroker))
	go func() {
		ti := time.NewTicker(time.Second)
		for _ = range ti.C {
			mu.Lock()
			tmpqps := qps
			tmpsum := sum
			mu.Unlock()
			avg := int64(0)
			if tmpqps > 0 {
				avg = tmpsum / tmpqps
			}

			echo.Info(
				"statistics",
				echo.Int64("qps", tmpqps),
				echo.Stringer("\tmin", time.Duration(atomic.LoadInt64(&min))),
				echo.Stringer("\tmax", time.Duration(atomic.LoadInt64(&max))),
				echo.Int64("\tavg(ms)", avg),
			)
			atomic.StoreInt64(&min, math.MaxInt64)
			atomic.StoreInt64(&max, 0)
			mu.Lock()
			qps = 0
			sum = 0
			mu.Unlock()
		}
	}()

	topic := "test-mq"
	var clients []broker.Broker
	for i := 0; i < optSubNum; i++ {
		// cOpts.SetClientID(fmt.Sprintf("sub_%d", i))
		c := Suber(topic)
		if c != nil {
			clients = append(clients, c)
		}
	}

	for i := 0; i < optPubNum; i++ {
		// cOpts.SetClientID(fmt.Sprintf("pub_%d", i))
		wg.Add(1)
		go Puber(topic, time.Millisecond*time.Duration(optItime))
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	sign := <-signalChan
	echo.Info("get", echo.String("sign", sign.String()))
	close(exited)
	for _, c := range clients {
		c.Disconnect()
	}
	wg.Wait()
	echo.Info("end")
}
