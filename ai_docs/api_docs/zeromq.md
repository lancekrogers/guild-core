# ZeroMQ Integration

Guild uses ZeroMQ for high-performance agent communication.

## Pattern

- `PUB/SUB`: Used to broadcast task state updates.
- `REQ/REP`: Used for tool invocation.

## Libraries

- [pebbe/zmq4](https://github.com/pebbe/zmq4) (Go binding for ZeroMQ)

## Design Notes

- Messages are JSON-encoded structs.
- Channels are namespaced by guild ID.
