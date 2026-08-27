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

a small redis-compatible server built from scratch to learn how redis works under the hood.

## features

- tcp server
- resp protocol
- `set`, `get`, `del`
- ttl / expiration
- pub/sub
- concurrent clients with goroutines
- thread-safe storage with `sync.rwmutex`
- snapshot persistence
- append-only logs
- recovery using snapshots + logs
- redis client compatibility

## architecture

```text
redis client
    ↓
tcp
    ↓
resp parser
    ↓
command handler
    ↓
storage / pubsub
    ↓
persistence
```


## getting started

clone the repository:

```bash
git clone https://github.com/rajandhamala/goRedis.git
cd goRedis
```
