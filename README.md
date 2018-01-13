# tcptunnel
用于两种场景：
1，直接的端口转发，这个好理解
2，做内网服务器到公网的映射访问，用于解决内网服务器没有公网IP或者无法进行端口映射的场景

## 编译
```
git clone https://github.com/LubyRuffy/tcptunnel
go build
```
生成tcptunnel文件

跨平台编译，比如到NAS服务器或者树莓派等ARM平台，执行：
```
GOOS=linux GOARCH=arm GOARM=5 go build
```

## 运行
直接执行是读取config.toml配置文件中的内容，最主要的是Mode和对应的配置内容，后续在配置文件中说明。
```
./tcptunnel 
```

### 作为内网映射运行：
- 作为publciserver执行，放到公网服务器
```
./tcptunnel -m publicserver
```
- 作为natserver执行，放到内网的服务器
```
./tcptunnel -m natserver
```
- 作为clientconnect执行，放到需要访问内网服务器的客户端
```
./tcptunnel -m clientconnect
```

### 作为tcpproxy执行，也就是端口转发
```
./tcptunnel -m tcpproxy
```

## 配置文件说明
默认读取config.toml文件，
```
# 模式： 支持publicserver，natserver，clientconnect，tcpproxy。可以通过命令行的-m参数覆盖
Mode = "publicserver"

# 连接公网服务器的地址，格式为 host:port
# 在Mode为 natserver 和 clientconnect 时有效
PublicServerAddr = "127.0.0.1:10011"

# 端口转发模式，仅仅在Mode为 tcpproxy 时有效
[TcpProxies]
    # 数组，可以多个映射关系
    [TcpProxies.proxy80]
    LocalBindAddr = "127.0.0.1:1234"
    RemoteServerAddr = "fofa.so:80"

    [TcpProxies.proxy22]
    LocalBindAddr = "127.0.0.1:1235"
    RemoteServerAddr = "106.39.208.89:22"

# 公网服务器监听的地址，仅仅在Mode为 publicserver 时有效，格式为 ip:port
[PublicServer]
LocalBindAddr = "127.0.0.1:10011"

# 端口转发模式，仅仅在Mode为 tcpproxy 时有效
[NatServer]
    # 数组，可以多个映射关系，ID用于注册，客户端连接的时候直接通过ID来进行查找
    [NatServer.test]
    RemoteServerAddr = "fofa.so:80"
    ID = "test"

    [NatServer.test1]
    RemoteServerAddr = "106.39.208.89:22"
    ID = "test1"

[ClientConnect]
    # 数组，可以多个映射关系，ID用于标示连接时指定NAT后的服务器对象
    [ClientConnect.test]
    LocalBindAddr = "127.0.0.1:1234"
    ID = "test"
```