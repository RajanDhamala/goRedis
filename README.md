# goRedis

Rewriting Redis in Go for learning and fun. This project explores how an
in-memory database works: data structures, expiration, persistence, pub/sub,
and transactions. It is a prototype with RESP2 client compatibility and a
subset of Redis commands.

## Run

```bash
export REDIS_PASSWORD='replace-with-your-password'
go run .
```

The server defaults to `127.0.0.1:6379`. Set `HOST` or `PORT` to change it.
Use the same password in your client application's environment.

## Clients

Use RESP2 and database 0.

### Node.js / Express (`npm install redis`)

```js
import { createClient } from "redis";

const cache = createClient({
  url: "redis://127.0.0.1:6379",
  RESP: 2,
  password: process.env.REDIS_PASSWORD,
});
cache.on("error", console.error);
await cache.connect();

await cache.set("greeting", "hello", { EX: 60 });
console.log(await cache.get("greeting"));

await cache.multi().set("counter", "0").incr("counter").exec();
```

### Go (`go get github.com/redis/go-redis/v9`)

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/redis/go-redis/v9"
)

func main() {
    client := redis.NewClient(&redis.Options{
        Addr:     "127.0.0.1:6379",
        Protocol: 2,
        Password: os.Getenv("REDIS_PASSWORD"),
        DB:       0,
    })
    defer client.Close()

    ctx := context.Background()
    if err := client.Set(ctx, "greeting", "hello", 0).Err(); err != nil {
        panic(err)
    }
    value, err := client.Get(ctx, "greeting").Result()
    if err != nil {
        panic(err)
    }
    fmt.Println(value)
}
```

### Python (`pip install redis`)

```python
import os
import redis

client = redis.Redis(host="127.0.0.1", port=6379, protocol=2,
                     password=os.environ["REDIS_PASSWORD"],
                     decode_responses=True)
client.set("greeting", "hello")
print(client.get("greeting"))
```

## Supported commands

| Area | Commands |
| --- | --- |
| Connection | `PING`, `ECHO`, `QUIT`, `AUTH`, `HELLO 2`, `SELECT 0` |
| Client metadata | `CLIENT SETINFO`, `CLIENT SETNAME`, `CLIENT GETNAME`, `CLIENT ID` |
| Strings | `SET`, `GET`, `INCR`, `DECR`, `INCRBY`, `DECRBY` |
| Keys and expiration | `DEL`, `EXISTS`, `TYPE`, `TTL`, `PTTL`, `EXPIRE`, `PEXPIRE` |
| Hashes | `HSET`, `HGET`, `HDEL`, `HLEN`, `HEXISTS`, `HGETALL` |
| Sets | `SADD`, `SREM`, `SISMEMBER`, `SMEMBERS`, `SCARD` |
| Lists | `LPUSH`, `RPUSH`, `LPOP`, `RPOP`, `LRANGE`, `LLEN` |
| Sorted sets | `ZADD`, `ZSCORE` |
| Pub/sub | `SUBSCRIBE`, `UNSUBSCRIBE`, `PUBLISH` |
| Transactions | `MULTI`, `EXEC`, `DISCARD` |
| Server | Basic `INFO` |

`SET` supports `NX`, `XX`, `GET`, `KEEPTTL`, `EX`, `PX`, `EXAT`, and `PXAT`.
Transactions execute queued commands in order without other clients interrupting;
there is no rollback or `WATCH` support.

Strings are persisted in `appendonly.aof`; collections remain in memory only.
AOF flushes every five seconds without an fsync guarantee, so recent writes can
be lost on shutdown. Pub/sub messages are not stored for offline subscribers.

## AI notice

The RESP2/client compatibility layer and transaction support were implemented
with an AI coding agent. The protocol compatibility work was complex, and using
an agent helped keep the project focused on learning Go and Redis internals.
