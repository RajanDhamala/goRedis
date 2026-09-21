# goRedis

<p align="left">
  <img src="https://cdn.jsdelivr.net/gh/devicons/devicon/icons/go/go-original.svg" width="70" />
  &nbsp;&nbsp;
  <img src="https://img.shields.io/badge/+-000000?style=flat-square" height="28" />
  &nbsp;&nbsp;
  <img src="https://cdn.jsdelivr.net/gh/devicons/devicon/icons/redis/redis-original.svg" width="70" />
</p>

<p align="center">
  rewriting redis in go for fun.
</p>

A small Redis-inspired server built from scratch in Go to learn how Redis works.
The prototype supports **RESP2** and a subset of Redis commands. It is not a full
Redis replacement.

## run

Create a `.env` file using `.env.example` as a guide. Set `REDIS_PASSWORD` to a
long random secret (for example, generate one with `openssl rand -hex 32`). Keep
`.env` private; it is Git-ignored. An existing environment variable takes precedence
over the `.env` file.

```bash
go run .                 # defaults to 127.0.0.1:6379
# or: PORT=6380 go run .
```

Startup fails if `REDIS_PASSWORD` is missing or blank. `HOST` defaults to
`127.0.0.1`; set it explicitly to listen on another interface. The server reads
and writes `appendonly.aof` in its working directory.

```bash
redis-cli -2 --askpass -p 6379
# Enter the same password, then run:
PING
SET greeting "hello world"
GET greeting
SET temporary value EX 60
```

`SET key value` now creates a persistent key. Use `EX seconds` or `PX milliseconds`
for expiration; the old positional `SET key value ttl` syntax is no longer accepted
on the network.

## client compatibility

Configure clients to use RESP2, database 0, and the same password as the server.
The only supported username is `default`; password-only authentication also works.
Load `REDIS_PASSWORD` into each application's environment (the server's `.env`
is not automatically loaded by other applications).

For an Express/Node.js app using `redis`:

```js
import { createClient } from "redis";

const cache = createClient({
  url: "redis://127.0.0.1:6379",
  RESP: 2,
  username: "default",
  password: process.env.REDIS_PASSWORD,
});
cache.on("error", console.error);
await cache.connect();
```

For Go using `github.com/redis/go-redis/v9`:

```go
client := redis.NewClient(&redis.Options{
    Addr:     "127.0.0.1:6379",
    Protocol: 2,
    Username: "default",
    Password: os.Getenv("REDIS_PASSWORD"),
    DB:       0,
})
```

For `redis-py`:

```python
import os
import redis

client = redis.Redis(host="127.0.0.1", port=6379, protocol=2,
                     password=os.environ["REDIS_PASSWORD"])
client.set("greeting", "hello world")
print(client.get("greeting"))

# Ordinary pipelining is supported; MULTI/EXEC transactions are not implemented.
with client.pipeline(transaction=False) as pipe:
    pipe.set("counter", "0")
    pipe.incr("counter")
    print(pipe.execute())
```

Clients authenticate each connection with `AUTH password`, `AUTH default password`,
or `HELLO 2 AUTH default password [SETNAME name]`. Before authentication, only
`AUTH`, `HELLO`, and `QUIT` are accepted. Other commands return `NOAUTH`; invalid
credentials return `WRONGPASS`. Authentication commands are never written to AOF.

This is single-password authentication, not per-user ACLs. Passwords and data are
still sent over plain TCP: keep access local/private, or use a TLS/SSH tunnel for
remote connections. Authentication alone does not make this prototype suitable
for public Internet exposure. See the [Redis security documentation](https://redis.io/docs/latest/operate/oss_and_stack/management/security/).

| Area | Supported commands |
| --- | --- |
| Connection | `PING [message]`, `ECHO`, `QUIT`, `AUTH [default] password`, `HELLO [2 [AUTH default password] [SETNAME name]]`, `SELECT 0` |
| Client metadata | `CLIENT SETINFO LIB-NAME/LIB-VER`, `SETNAME`, `GETNAME`, `ID` |
| Strings | `SET`, `GET`, `INCR`, `DECR`, `INCRBY`, `DECRBY` |
| Keys | `DEL key [key ...]`, `EXISTS key [key ...]`, `TYPE`, `TTL`, `PTTL`, `EXPIRE`, `PEXPIRE` |
| Hashes | `HSET`, `HGET`, `HDEL`, `HLEN`, `HEXISTS`, `HGETALL` |
| Sets | `SADD`, `SREM`, `SISMEMBER`, `SMEMBERS`, `SCARD` |
| Lists | `LPUSH`, `RPUSH`, `LPOP`, `RPOP`, `LRANGE`, `LLEN` |
| Sorted sets | `ZADD`, `ZSCORE` |
| Pub/sub | `SUBSCRIBE`, `UNSUBSCRIBE`, `PUBLISH` |
| Server | Basic `INFO` |

`SET` supports `NX`, `XX`, `GET`, `KEEPTTL`, `EX`, `PX`, `EXAT`, and `PXAT`.
Hash/set writes and list pushes accept multiple fields/members/values. `ZADD`
accepts multiple score/member pairs and updates existing members. Optional `ZADD`
flags, count arguments for list pops, and conditional expiration flags are not
implemented. `LRANGE` supports negative indexes.

Replies use RESP strings, integers, nulls, errors, and arrays, including pub/sub
acknowledgments and messages. Keys share type checks, deletion, and expiration
across the implemented data structures. During a subscription, supported commands
are `SUBSCRIBE`, `UNSUBSCRIBE`, `PING`, and `QUIT`.

## prototype boundaries

- RESP3, per-user ACLs, native TLS, additional databases, transactions, scripting, pattern
  subscriptions, and unlisted Redis commands are not implemented. Unsupported
  commands/options return errors; `HELLO 3` returns `NOPROTO`.
- Strings are persisted through AOF, including counter and expiration changes.
  Hashes, sets, lists, and sorted sets remain in memory only.
- New AOF records use binary-safe RESP and absolute expiry timestamps. Existing
  inline AOF records are still readable; their original relative-TTL behavior remains.
  Old records that already lost spaces/newlines cannot be reconstructed.
- AOF retains the prototype's five-second buffered flush and has no shutdown flush
  or fsync guarantee. Recent writes can be lost when the process stops.
- Command execution is serialized for correctness. The linked-list and map data
  structures remain deliberately simple.
- Requests are limited to 64 MiB of argument data and 65,536 arguments. A client
  whose outbound queue fills (1,024 messages) or whose socket write stalls for ten
  seconds is disconnected. Pub/sub is not a durable message queue.
- Simple unquoted inline commands remain available for manual testing. Normal
  clients should use RESP requests.

## verification

```bash
go test -race ./...
go vet ./...
```

Tests cover fragmented/binary requests, pipelining, malformed frames, typed
responses, command arity, collections, pub/sub, concurrent clients, AOF replay,
and authentication failures, isolation, and handshakes.
The implementation was also exercised with `redis-cli` 7.0.15 and `redis-py` 8.1.0
in RESP2 mode, including an actual server restart for string recovery.

Authentication was additionally verified with Node Redis 5.12.1, go-redis v9.7.3,
redis-py 8.1.0, and redis-cli 7.0.15. Checks include password/HELLO handshakes,
missing and incorrect credentials, startup rejection without a password, and the
default loopback listener.
