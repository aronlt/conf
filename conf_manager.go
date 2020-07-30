package conf

import (
	"conf/fileutil"
	"context"
	"errors"
	"sync"
	"time"
)

var configManager *MConfigManager
var emptyConfig *MConfig

const monitorDuration = 10

type MConfigManager struct {
	confs    sync.Map
	ctx      context.Context
	cancel   context.CancelFunc
	callback map[string]func(string)
}

func init() {
	ctx, cancel := context.WithCancel(context.Background())
	configManager = &MConfigManager{
		ctx:      ctx,
		cancel:   cancel,
		callback: make(map[string]func(string), 0),
	}
	emptyConfig = newMConfig("")
}

/*
 * 设置配置文件名和路径信息
 */
func SetConfig(confName string, fpath string, callback func(string)) error {
	ok, err := fileutil.IsFile(fpath)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("fileutil not exist")
	}
	if _, ok := configManager.confs.Load(confName); ok == false {
		conf := newMConfig(fpath)
		configManager.confs.Store(confName, conf)
		configManager.callback[confName] = callback
	}
	return nil
}

/*
 * 获取默认的配置（针对只有一个配置文件的时候，简化操作)
 */
func Config() *MConfig {
	var conf *MConfig
	configManager.confs.Range(func(key, value interface{}) bool {
		conf = value.(*MConfig)
		return false
	})
	if conf == nil {
		return emptyConfig
	}
	return conf
}

/*
 * 存在多个配置文件的情况下，返回对应的配置文件
 */

func MultiConfig(name string) *MConfig {
	if val, ok := configManager.confs.Load(name); ok {
		return val.(*MConfig)
	}
	return emptyConfig
}

/*
 * 关闭监听
 */
func StopMonitor() {
	configManager.cancel()
}

func StartMonitor() {
	go configManager.monitor()
}

func (m *MConfigManager) monitor() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-time.After(monitorDuration * time.Second):
			updatedConfs := sync.Map{}
			// 检查所有的配置，查看是否有变化
			m.confs.Range(func(key, value interface{}) bool {
				conf := value.(*MConfig)
				if conf.checkFileDiff() {
					updatedConf := newMConfig(conf.path)
					updatedConfs.Store(key, updatedConf)
				}
				return true
			})
			// 更新变化的配置（直接替换)
			updatedConfs.Range(func(key, value interface{}) bool {
				m.confs.Store(key, value)
				if handler, ok := m.callback[key.(string)]; ok {
					if handler != nil {
						handler(key.(string))
					}
				}
				return true
			})
		}
	}
}
