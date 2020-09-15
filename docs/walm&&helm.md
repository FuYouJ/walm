# Helm VS WAlM

## 通过Helm部署容器面临的问题：

- Helm 只提供静态依赖管理，也无所做到应用配置实时感知，对于有依赖链关系的应用难以维护 。（见场景一和场景二）
- 不支持JSONNET。 大数据组件都很复杂， Inceptor 依赖 metastore， guardian，hadoop， zk等， 开源的 helm charts 语法不支持这样复杂的 chart 编排。
- Helm 只提供了命令行接口， 操作复杂且无法满足需求
- 维护成本。复杂应用（如 HDFS）的chart不容易维护
- HElM 没有提供详细的应用状态信息
- 应用定制化麻烦

而 WALM 可以解决上述的问题。

|                         | WALM                | HELM        |
| ----------------------- | ------------------- | ----------- |
| 接口                    | 命令行， RESTAPI    | 命令行      |
| 服务生命周期管理        | 支持                | 支持        |
| 软件包管理              | 支持                | 支持        |
| K8s应用管理             | 支持                | 支持        |
| K8s资源管理             | 支持                | 支持        |
| 服务升级/回滚           | 支持                | 支持        |
| 应用状态实时同步        | 支持                | 不支持      |
| 服务动态依赖            | 支持                | 不支持      |
| 编排语言                | GO-TEMLATE，JSONNET | GO-TEMPLATE |
| k8s复杂应用生命周期管理 | 支持                | 不支持      |
| 应用详细信息            | 支持                | 不支持      |
| 应用微服务化            | 支持                | 不支持      |

### 场景1

k8s集群上已经存在了一个运行良好的Zookeeper集群，现在需要安装一个Kafka集群，依赖这个已经存在的Zookeeper集群。

|      | 能否满足需求 | 原因                                                        |
| ---- | ------------ | ----------------------------------------------------------- |
| Walm | 能           | 支持应用的动态依赖， 即可以先安装Zookeeper， 再安装Kafka。  |
| Helm | 不能         | 只支持应用的静态依赖， 即Kafka和Zookeeper必须同时安装与卸载 |

- 查看Zookeeper的详细信息。通过Walm可以查看应用更加详细的信息，例如是否ready, 没有ready的原因，应用包含的所有k8s资源的状态，动态依赖信息等。

  ![image.png](https://i.loli.net/2020/09/14/d6a1I4cCAUxsJlF.png)

- 部署Kafka集群，并依赖于已经存在的Zookeeper集群

  ![image3](https://i.loli.net/2020/09/14/kdMOi4NXgvsSou2.png)

  

- 查看部署好的Kafka集群。可以看到Kafka依赖于已经存在的Zookeeper集群， 并且可以看到依赖的配置

  ![image2](https://i.loli.net/2020/09/14/UOLnc9bqGzgN83J.png)

  ![image3.png](https://i.loli.net/2020/09/14/4WpUy1YRPFCrGds.png)

  

### 场景二

- 扩容Zookeeper集群

  ![image.png](https://i.loli.net/2020/09/14/OzXnA6TaYGRKodm.png)

- 查看Kafka集群是否自动感知到了Zookeeper集群的变化

  ![image.png](https://i.loli.net/2020/09/14/4o6KgEsJxAeidCH.png)

  ![image.png](https://i.loli.net/2020/09/14/w7TPqupHxRigK9h.png)

  ​              

## JSONNET 的优点

> [Jsonnet 简明教程与应用](https://aleiwu.com/post/jsonnet-grafana/)

Jsonnet是Google开源的一门配置语言，用于弥补JSON所暴露的短板，它完全兼容JSON，并加入了JSON所没有的一些特性，包括注释、引用、算数运算、条件操作符、数组和对象深入、引入函数、局部变量、继承等，Jsonnet程序被编译为兼容JSON的数据格式。

Jsonnet 使用场景主要集中在配置管理上. 社区的实践主要是用 jsonnet 做 Kubernetes, Prometheus, Grafana 的配置管理，通过 jsonnet， 我们可以解决复杂的应用模版相关的问题。 相关的库有:

- [kubecfg](https://github.com/bitnami/kubecfg): 使用 jsonnet 生成 kubernetes API 对象 并 apply
- [ksonnet-lib](https://github.com/ksonnet/ksonnet-lib): 一个 jsonnet 的库, 用于生成 kubernetes API 对象
- [kube-prometheus](https://github.com/coreos/prometheus-operator/tree/master/contrib/kube-prometheus): 使用 jsonnet 生成 Prometheus-Operator, Prometheus, Grafana 以及一系列监控组件的配置
- [grafonnet-lib](https://github.com/grafana/grafonnet-lib/tree/master/grafonnet): 一个 jsonnet 的库, 用于生成 json 格式的 Grafana 看板配置

- ......

## 解决复杂的应用模版

应用模板内容
- 编排信息
  - 资源使用情况（request vs limit）
  - 调度偏好
    - affinity， anti-affinity，node selector
- 容器环境变量
  - 依赖信息
  - 用户配置 **（最复杂）**
  - 资源配置 （container资源信息，jvm）
- 存储卷
- 网络

## 流程图
![image.png](https://i.loli.net/2020/09/15/XTfLoIvZYFSuCa5.png)
