## 介绍
conf是一个针对json配置文件的工具库，它具有如下特征
* 线程安全
* 支持监听文件变化, 自动更新配置项
* 支持复杂的配置项获取，只需传入字符串即可
* 支持同时设置多个配置文件


## 使用
```go
    // 使用SetConfig函数，可以设置配置文件的名，路径，以及如果配置文件更新后的回调函数
    SetConfig("default", path, func(s string) {
		fmt.Printf("config:%s content changed!\n", s)
	})
    // 如果你不关心文件变化，就不用启动监控
	StartMonitor()
   // 记得关闭监听
    defer StopMonitor()
    /*
    * {
        "key10": {
          "key11": [
            {
              "key12": "value12",
              "key13": "value13"
            }
          ]
        }
      }
    */
    // 获取复杂的配置项的值, 使用Config()函数，你可以省去传入配置文件
	val, err := Config().GetString("key10.key11[0].key12")
    // 如果你不关心error，那么可以设置一个默认值，这样可以实现链式调用
    val := Config().GetStringWithDefault("key10.key11[0].key12", "default")
    // 如果你有多个配置文件就可以使用这种方法获取指定配置文件的配置项
    val := MultiConfig("default").GetStringWithDefault("key10.key11[0].key12", "default")
    
```
