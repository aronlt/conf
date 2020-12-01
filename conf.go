package conf

import (
	"bytes"
	"conf/fileutil"
	"encoding/json"
	"errors"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var TypeErr = errors.New("invalid type")
var KeyNotFoundErr = errors.New("not found key")
var InvalidKeyErr = errors.New("invalid key")
var InvalidSliceIndexErr = errors.New("invalid slice index")

type MConfig struct {
	// 配置所在的目录
	dir string
	// 配置文件名
	filename string
	// 配置文件路径
	path string
	// md5
	md5 string
	// 解析过的配置项
	parsedEntryMap sync.Map
	// 未解析的配置项
	rawEntryMap map[string]json.RawMessage
	locker      sync.Mutex
}

func newMConfig(p string) *MConfig {
	// 空的conf
	if p == "" {
		return &MConfig{
			rawEntryMap: make(map[string]json.RawMessage, 0),
		}
	}
	if ok, err := fileutil.IsFile(p); ok == false {
		log.Printf("file:%s not exist, err:%+v", p, err)
		return nil
	}
	cf := &MConfig{
		rawEntryMap: make(map[string]json.RawMessage, 0),
		dir:         filepath.Dir(p),
		filename:    filepath.Base(p),
		path:        p,
	}

	content, err := fileutil.ReadContent(p)
	if err != nil {
		log.Printf("can't read file:%s content, error:%s\n", p, err.Error())
		return nil
	}
	cf.md5, err = fileutil.HashFileMd5(p)
	if err != nil {
		log.Printf("can't get file:%s md5, error:%s\n", p, err.Error())
		return nil
	}
	err = json.Unmarshal(content, &cf.rawEntryMap)
	if err != nil {
		log.Printf("decode config fileutil:%s failed, error:%s\n", p, err.Error())
		return cf
	}
	return cf
}

func (m *MConfig) checkFileDiff() bool {
	md5, err := fileutil.HashFileMd5(m.path)
	if err != nil {
		log.Printf("can't get file:%s md5, error:%s\n", m.path, err.Error())
		return false
	}
	return md5 != m.md5
}

/*
 * 遍历json串中的对象
 * travel 支持按照字符串的模式遍历json，例如传入key1.key2.key3的模式，特别针对数组的情况
 * 你可以使用key1.key2[1].key3的模式传入，函数会帮你解析数组下标
 */
func (m *MConfig) travel(key string, lastElemHandler func(val json.RawMessage, index int) (interface{}, error)) (string, interface{}, error) {
	elems := strings.Split(key, ".")
	rawMap := m.rawEntryMap

	shapingKey := bytes.NewBuffer([]byte{})

	// 对每个split开来的单元进行处理
	for i, elem := range elems {
		if elem == "" {
			return "", nil, InvalidKeyErr
		}
		// 查看是否是数组模式
		index := -1
		if elem[len(elem)-1] == ']' {
			boundary := false
			nums := bytes.NewBuffer([]byte{})
			for i := len(elem) - 2; i >= 0; i-- {
				if elem[i] == '[' {
					// 取出真正的key
					elem = elem[0:i]
					var err error
					index, err = strconv.Atoi(nums.String())
					if err != nil {
						return "", nil, InvalidKeyErr
					}
					if index < 0 {
						return "", nil, InvalidSliceIndexErr
					}
					boundary = true
					break
				} else if elem[i] == ' ' {
					continue
				} else {
					nums.WriteByte(elem[i])
				}
			}
			if boundary == false {
				return "", nil, InvalidKeyErr
			}
		}
		// 从rawMap中找到对应的项，所以这里要生成key字符串
		elem = strings.Trim(elem, " ")
		shapingKey.WriteString(elem)
		if index >= 0 {
			shapingKey.WriteString("[")
			shapingKey.WriteString(strconv.Itoa(index))
			shapingKey.WriteString("]")
		}
		if i != len(elem)-1 {
			shapingKey.WriteString(".")
		}

		// 查看当前key的内容
		if val2, ok := rawMap[elem]; ok {
			// 如果已经遍历到最后一个elem，就直接调用处理函数进行处理
			if i == len(elems)-1 {
				value, err := lastElemHandler(val2, index)
				return shapingKey.String(), value, err
			} else {
				// 检查是否是数组
				if index >= 0 {
					var rawSliceMap []json.RawMessage
					if err := json.Unmarshal(val2, &rawSliceMap); err != nil {
						return "", nil, err
					} else {
						// 下标不对
						if len(rawSliceMap) <= index {
							return "", nil, InvalidSliceIndexErr
						}
						// 解析失败
						if err := json.Unmarshal(rawSliceMap[index], &rawMap); err != nil {
							return "", nil, err
						}
					}
				} else if err := json.Unmarshal(val2, &rawMap); err != nil {
					return "", nil, err
				}
			}
		} else {
			// 如果key不存在，就直接返回失败
			return "", nil, KeyNotFoundErr
		}
	}
	return "", nil, KeyNotFoundErr
}

func (m *MConfig) GetString(key string) (string, error) {
	val, ok := m.parsedEntryMap.Load(key)
	if ok {
		strVal, ok := val.(string)
		if !ok {
			return "", TypeErr
		}
		return strVal, nil
	} else {
		shapingKey, val, err := m.travel(key, func(val json.RawMessage, index int) (interface{}, error) {
			if index < 0 {
				var strVal string
				if err := json.Unmarshal(val, &strVal); err != nil {
					return "", err
				}
				return strVal, nil
			} else {
				var strSliceVal []string
				if err := json.Unmarshal(val, &strSliceVal); err != nil {
					return "", err
				}
				if len(strSliceVal) <= index {
					return "", InvalidSliceIndexErr
				}
				return strSliceVal[index], nil
			}
		})
		if err != nil {
			return "", err
		} else {
			m.parsedEntryMap.Store(shapingKey, val.(string))
			return val.(string), nil
		}
	}
}

func (m *MConfig) GetStringSlice(key string) ([]string, error) {
	val, ok := m.parsedEntryMap.Load(key)
	if ok {
		strSliceVal, ok := val.([]string)
		if !ok {
			return []string{}, TypeErr
		}
		return strSliceVal, nil
	} else {
		shapingKey, val, err := m.travel(key, func(val json.RawMessage, index int) (interface{}, error) {
			var strSliceVal []string
			if err := json.Unmarshal(val, &strSliceVal); err != nil {
				return []string{}, err
			}
			return strSliceVal, nil
		})
		if err != nil {
			return []string{}, err
		} else {
			m.parsedEntryMap.Store(shapingKey, val.([]string))
			return val.([]string), nil
		}
	}
}

func (m *MConfig) GetStringWithDefault(key string, defaultVal string) string {
	if val, err := m.GetString(key); err != nil {
		return defaultVal
	} else {
		return val
	}
}

func (m *MConfig) GetInt(key string) (int, error) {
	val, ok := m.parsedEntryMap.Load(key)
	if ok {
		intVal, ok := val.(int)
		if !ok {
			return 0, TypeErr
		}
		return intVal, nil
	} else {
		shapingKey, val, err := m.travel(key, func(val json.RawMessage, index int) (interface{}, error) {
			if index < 0 {
				var intVal int
				if err := json.Unmarshal(val, &intVal); err != nil {
					return 0, err
				}
				return intVal, nil
			} else {
				var intSliceVal []int
				if err := json.Unmarshal(val, &intSliceVal); err != nil {
					return 0, err
				}
				if len(intSliceVal) <= index {
					return 0, InvalidSliceIndexErr
				}
				return intSliceVal[index], nil
			}
		})
		if err != nil {
			return 0, err
		} else {
			m.parsedEntryMap.Store(shapingKey, val.(int))
			return val.(int), nil
		}
	}
}

func (m *MConfig) GetIntSlice(key string) ([]int, error) {
	val, ok := m.parsedEntryMap.Load(key)
	if ok {
		intSliceVal, ok := val.([]int)
		if !ok {
			return []int{}, TypeErr
		}
		return intSliceVal, nil
	} else {
		shapingKey, val, err := m.travel(key, func(val json.RawMessage, index int) (interface{}, error) {
			var intSliceVal []int
			if err := json.Unmarshal(val, &intSliceVal); err != nil {
				return []int{}, err
			}
			return intSliceVal, nil
		})
		if err != nil {
			return []int{}, err
		} else {
			m.parsedEntryMap.Store(shapingKey, val.([]int))
			return val.([]int), nil
		}
	}
}

func (m *MConfig) GetIntWithDefault(key string, defaultVal int) int {
	if val, err := m.GetInt(key); err != nil {
		return defaultVal
	} else {
		return val
	}
}

func (m *MConfig) GetFloat(key string) (float32, error) {
	val, ok := m.parsedEntryMap.Load(key)
	if ok {
		floatVal, ok := val.(float32)
		if !ok {
			return 0.0, TypeErr
		}
		return floatVal, nil
	} else {
		shapingKey, val, err := m.travel(key, func(val json.RawMessage, index int) (interface{}, error) {
			if index < 0 {
				var floatVal float32
				if err := json.Unmarshal(val, &floatVal); err != nil {
					return 0.0, err
				}
				return floatVal, nil
			} else {
				var floatSliceVal []float32
				if err := json.Unmarshal(val, &floatSliceVal); err != nil {
					return 0, err
				}
				if len(floatSliceVal) <= index {
					return 0, InvalidSliceIndexErr
				}
				return floatSliceVal[index], nil
			}
		})
		if err != nil {
			return 0.0, err
		} else {
			m.parsedEntryMap.Store(shapingKey, val.(float32))
			return val.(float32), nil
		}
	}
}

func (m *MConfig) GetFloatSlice(key string) ([]float32, error) {
	val, ok := m.parsedEntryMap.Load(key)
	if ok {
		floatSliceVal, ok := val.([]float32)
		if !ok {
			return []float32{}, TypeErr
		}
		return floatSliceVal, nil
	} else {
		shapingKey, val, err := m.travel(key, func(val json.RawMessage, index int) (interface{}, error) {
			var floatSliceVal []float32
			if err := json.Unmarshal(val, &floatSliceVal); err != nil {
				return []float32{}, err
			}
			return floatSliceVal, nil
		})
		if err != nil {
			return []float32{}, err
		} else {
			m.parsedEntryMap.Store(shapingKey, val.([]float32))
			return val.([]float32), nil
		}
	}
}

func (m *MConfig) GetFloatWithDefault(key string, defaultVal float32) float32 {
	if val, err := m.GetFloat(key); err != nil {
		return defaultVal
	} else {
		return val
	}
}

func (m *MConfig) GetBool(key string) (bool, error) {
	val, ok := m.parsedEntryMap.Load(key)
	if ok {
		boolVal, ok := val.(bool)
		if !ok {
			return false, TypeErr
		}
		return boolVal, nil
	} else {
		shapingKey, val, err := m.travel(key, func(val json.RawMessage, index int) (interface{}, error) {
			if index < 0 {
				var boolVal bool
				if err := json.Unmarshal(val, &boolVal); err != nil {
					return 0.0, err
				}
				return boolVal, nil
			} else {
				var boolSliceVal []bool
				if err := json.Unmarshal(val, &boolSliceVal); err != nil {
					return 0, err
				}
				if len(boolSliceVal) <= index {
					return 0, InvalidSliceIndexErr
				}
				return boolSliceVal[index], nil
			}
		})
		if err != nil {
			return false, err
		} else {
			m.parsedEntryMap.Store(shapingKey, val.(bool))
			return val.(bool), nil
		}
	}
}

func (m *MConfig) GetBoolSlice(key string) ([]bool, error) {
	// 先从缓存中取出
	val, ok := m.parsedEntryMap.Load(key)
	if ok {
		boolSliceVal, ok := val.([]bool)
		if !ok {
			return []bool{}, TypeErr
		}
		return boolSliceVal, nil
	} else {
		shapingKey, val, err := m.travel(key, func(val json.RawMessage, index int) (interface{}, error) {
			var boolSliceVal []bool
			if err := json.Unmarshal(val, &boolSliceVal); err != nil {
				return []bool{}, err
			}
			return boolSliceVal, nil
		})
		if err != nil {
			return []bool{}, err
		} else {
			m.parsedEntryMap.Store(shapingKey, val.([]bool))
			return val.([]bool), nil
		}
	}
}

func (m *MConfig) GetBoolWithDefault(key string, defaultVal bool) bool {
	if val, err := m.GetBool(key); err != nil {
		return defaultVal
	} else {
		return val
	}
}

/*
 * 返回原始片段，如果需要自己处理的话，可以自己处理
 */
func (m *MConfig) GetRawMessage(key string) (json.RawMessage, error) {
	_, val, err := m.travel(key, func(val json.RawMessage, index int) (interface{}, error) {
		var rawVal json.RawMessage
		if err := json.Unmarshal(val, &rawVal); err != nil {
			return json.RawMessage{}, err
		}
		return rawVal, nil
	})
	if err != nil {
		return json.RawMessage{}, err
	} else {
		return val.(json.RawMessage), nil
	}
}
