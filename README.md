# 微服务高可用之熔断器实现原理与 Golang 实践

在微服务架构中，经常会碰到服务超时或通讯失败的问题，由于服务间层层依赖，很可能由于某个服务出现问题，不合理的重试和超时设置，导致问题层层传递引发雪崩现象，而限流和熔断是解决这个问题重要的方式。

本章内容提要：
- 微服务高可用容错机制
- 熔断器设计原理及 Golang 实现
- 服务网格和代理网关熔断机制

### 阅读全文链接
[微服务熔断实现](https://mp.weixin.qq.com/s?__biz=MzIyMzMxNjYwNw==&mid=2247484006&idx=1&sn=14083070db9a5aa54d6d55718c6f967c&chksm=e8215d76df56d4605b7364fafbb7dc344cac86f3f7955d7131a8b6263caa78514c2976aa4fab&token=1518190680&lang=zh_CN#rd)

扫码关注微信订阅号支持：

技术岁月

techyears

![技术岁月](https://raw.githubusercontent.com/skyhackvip/ratelimit/master/techyears.jpg)
