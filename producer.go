package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/micro/go-micro/broker"
	"github.com/vizee/echo"
)

func Puber(topic string, itime time.Duration) {
	defer wg.Done()
	client := NewBroker(broker.Addrs(optBroker))
	if err := client.Init(); err != nil {
		echo.Error("Broker init error", echo.Errval(err))
		return
	}
	if err := client.Connect(); err != nil {
		echo.Error("Broker Connect error", echo.Errval(err))
		return
	}
	defer client.Disconnect()
	for {
		tp := topic
		if optRandTopic {
			tp = fmt.Sprintf("%s%d", topic, rand.Int()%(optTopicNum))
		}
		msg := &broker.Message{
			Body: []byte(fmt.Sprintf("%d", time.Now().UnixNano())),
		}
		if err := client.Publish(tp, msg); err != nil {
			echo.Error("connect err", echo.Errval(err))
			return
		}
		select {
		case <-exited:
			return
		case <-time.After(itime):
		}
	}

}
