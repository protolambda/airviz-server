# airviz - server

EthNewYork hack, websocket server

## Server

Hub: maintains clients and routes data pushes to each client

Client: listens for messages from user, gets pushes from hub.

DataReqHandler: filters pushes by time range, batches pushes into requests. Handles requests together with user requests. Only acts on latest request.

Topic: `byte`

Time index: `uint32`

DAG: dag structure, items per layer

Node: dag item, *points* to parent node

Box: index, parent key, self key, value

Status: client status, count (depth) of items at each time index. Offset by possibly different time index than server, window is of constant length.

Update atom: only new items are send to the client, i.e. the `[old_depth:new_depth]` range. One item per message.

## Messages:

### request

```
1 byte | 4 bytes    | 4 bytes per index
topic  | time index | depth per index, starting from time index, defines window of request.
```

### response: update item

```
1 byte | 4 bytes    | 4 bytes  | 32 bytes   | 32 bytes | n bytes
topic  | time index | depth    | parent key | self key | SSZ serialized data
```
