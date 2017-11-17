# mqtest
## What is mqtest ?
-A mq Performance Test tool, support nats/nsq/kafka/rabbitmq/redis/mqtt broker.
## How to use ?
```
go build
./mqtest
``` 

## args
```
-broker string
        broker addr (default "tcp://127.0.0.1:1883")
  -itime int
        pub sleep time(ms) (default 300)
  -log string
        path/to/roll.log
  -pprof string
        address:port (default ":0")
  -pub int
        pub num (default 1)
  -randtopic
        rand pub/sub a topic
  -sub int
        sub num (default 1)
  -topic int
        topic num (only randtopic=true) (default 1)
```