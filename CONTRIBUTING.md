<div align="center">
  <img src="https://cdn.jsdelivr.net/gh/devicons/devicon/icons/go/go-original.svg" width="70" align="middle" alt="Go logo" />
  &nbsp;&nbsp;
  <img src="https://img.shields.io/badge/%2B-6B7280?style=for-the-badge" height="32" align="middle" alt="plus" />
  &nbsp;&nbsp;
  <img src="https://cdn.jsdelivr.net/gh/devicons/devicon/icons/redis/redis-original.svg" width="62" align="middle" alt="Redis logo" />

  <h1>Contributing to goRedis</h1>
</div>

Thanks for checking out **goRedis**.

This project is a Redis-inspired server written in Go for fun, learning, and understanding how systems like Redis work under the hood.

Contributions, ideas, feedback, reviews, and suggestions are all welcome.

## Contributions Welcome

You can help with things like:

- RESP parsing
- `SET`, `GET`, `DEL`
- TTL and expiration
- Pub/Sub
- JSON commands
- Concurrent client handling
- Mutex / `RWMutex` improvements
- Snapshot persistence
- Append-only logs
- Crash recovery
- Durable streams
- Consumer offsets
- Acknowledgements
- Consumer groups
- Tests
- Benchmarks
- Documentation
- Performance improvements
- Architecture suggestions

If you notice a better way to implement something, feel free to open an issue or pull request.

## Feedback

This project is being built to learn.

Feedback is encouraged, especially around:

- Go concurrency patterns
- Networking
- RESP compatibility
- Redis client compatibility
- Storage design
- Persistence
- Pub/Sub architecture
- Performance
- Code organization

If something looks wrong, inefficient, unsafe, or overcomplicated, feel free to point it out.

## Getting Started

Clone the repository:

```bash
git clone https://github.com/RajanDhamala/goRedis.git
cd goRedis
