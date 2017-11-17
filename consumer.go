package main

import (
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/micro/go-micro/broker"
	"github.com/micro/go-plugins/broker/mqtt"
	"github.com/vizee/echo"
)

func sub(client broker.Broker, topic string) (suber broker.Subscriber, err error) {
	t, err := client.Subscribe(topic, func(p broker.Publication) error {
		var sendTime int64
		fmt.Sscanf(string(p.Message().Body), "%d", &sendTime)
		cost := time.Now().UnixNano() - sendTime
		if cost < atomic.LoadInt64(&min) {
			atomic.StoreInt64(&min, cost)
		}
		if cost > atomic.LoadInt64(&max) {
			atomic.StoreInt64(&max, cost)
		}

		mu.Lock()
		qps++
		sum += cost / int64(time.Millisecond)
		mu.Unlock()
		// fmt.Println(mq.Topic(), mq.MessageID(), string(mq.Payload()))
		return nil
	})
	if err != nil {
		echo.Error("sub err", echo.Errval(err))
		return nil, err
	}
	return t, nil
}

func Suber(topic string) broker.Broker {
	client := mqtt.NewBroker(broker.Addrs(optBroker))
	if err := client.Init(); err != nil {
		echo.Error("Broker init error", echo.Errval(err))
		return nil
	}
	if err := client.Connect(); err != nil {
		echo.Error("Broker Connect error", echo.Errval(err))
		return nil
	}

	if optRandTopic { // 随机取消订阅重新订阅
		go func() {
			var (
				suber broker.Subscriber
				err   error
			)
			tp := topic
			tp = fmt.Sprintf("%s%d", topic, rand.Int()%(optTopicNum))
			if suber, err = sub(client, tp); err != nil {
				return
			}
			for _ = range time.Tick(time.Second * 100) {
				if err = suber.Unsubscribe(); err != nil {
					echo.Error("Unsubscribe err", echo.Errval(err))
					return
				}
				tp = fmt.Sprintf("%s%d", topic, rand.Int()%(optTopicNum))
				if suber, err = sub(client, tp); err != nil {
					return
				}

			}
		}()
	} else {
		sub(client, topic)
	}
	return client
}
