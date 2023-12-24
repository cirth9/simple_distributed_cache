# simpleDistributedCache
一个简单实现的分布式缓存

lru算法用于缓存淘汰策略

一致性哈希用于在分布式系统中，同一个key命中同一个peer，在后续添加peer节点的时候，也只需要使一部分缓存失效，而不需要像普通哈希一样近乎全部失效，导致缓存雪崩问题

另外因为可能存在缓存击穿和缓存穿透问题，这里采用了singleflight的解决方案，将多个相同请求合并为一个请求，以此不至于对数据库有太多的压力

基于HTTP协议通过protobuf进行通信，加快通行效率


