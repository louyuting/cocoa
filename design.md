> 主要描述cache实现的几个点：

# 核心设计点

1. 首先需要有一个高性能的并发线程安全的 hash map，map用于存储实际的key value。
    1.1  目前还是读多写少的场景，初期可以复用 读写锁 + map 结构
    1.2 后面优化实际的实现可以参考 ConcurrentHashMap
2. 异步化，单独是数据结构去维护淘汰策略。比如基于时间过期，基于access过期等等。
3. 淘汰策略的数据结构异步执行具体的淘汰策略。
4. 频率统计使用 Count-Min Sketch

automatic loading of entries into the cache, optionally asynchronously
size-based eviction when a maximum is exceeded based on frequency and recency
time-based expiration of entries, measured since last access or last write
asynchronously refresh when the first stale request for an entry occurs
notification of evicted (or otherwise removed) entries
writes propagated to an external resource
accumulation of cache access statistics

















