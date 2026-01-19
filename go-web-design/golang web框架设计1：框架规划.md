> GO WEB 编程13节，[如何设计一个web框架](https://github.com/astaxie/build-web-application-with-golang/blob/master/zh/13.0.md)

**golang web framework 框架系列文章：**
- [7. golang web框架设计7：整合框架](https://www.cnblogs.com/jiujuan/p/11899010.html)
- [6. golang web框架设计6：上下文设计](https://www.cnblogs.com/jiujuan/p/11898983.html)
- [5. golang web框架设计5：配置设计](https://www.cnblogs.com/jiujuan/p/11898928.html)
- [4. golang web框架设计4：日志设计](https://www.cnblogs.com/jiujuan/p/11898825.html)
- [3. golang web框架设计3：controller设计](https://www.cnblogs.com/jiujuan/p/11898798.html)
- [2. golang web框架设计2：自定义路由](https://www.cnblogs.com/jiujuan/p/11898745.html)
- [1. golang web框架设计1：框架规划](https://www.cnblogs.com/jiujuan/p/11898714.html)

学习谢大的web框架设计

## 总体介绍
实现一个简易的web框架，我们采用mvc模式来进行开发。
model：模型，代表数据结构。通常来说，模型类时包含查询，插入，更新数据库资料等这些共
view：视图，向用户展示信息
controller：控制器，它是模型和视图以及其他http请求所必须的资源之间的中介

## 框架功能
设计一个最小化的web框架，包括功能
- 路由
- RESTful的控制器
- 模板
- 日志系统
- 配置管理

等基本功能