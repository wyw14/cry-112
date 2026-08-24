基于 Go 实现的 SteriCycle 项目，一款医疗器械蒸汽灭菌过程联控服务，负责协调热力循环、介质路径与双门安全证明。

# SteriCycle

SteriCycle 通过本地事件日志和原子快照保存灭菌批次状态，并提供腔体、蒸汽、门锁和事故查询 API。

## 构建

```text
go build -mod=vendor ./...
```

## 运行

```text
STERICYCLE_ADDR=127.0.0.1:21212 STERICYCLE_DATA=./data go run -mod=vendor ./cmd/stericycle
```

服务暴露 `/healthz`、`/api/cycles`、`/api/chambers`、`/api/steam`、`/api/doors` 和 `/api/incidents`。
