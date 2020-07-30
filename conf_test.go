package conf

import (
	"fmt"
	"meili_conf/fileutil"
	"testing"
	"time"
)

func TestSetConfig(t *testing.T) {
	path := "testdir/test.json"
	SetConfig("default", path, nil)
	if val, ok := configManager.confs.Load("default"); ok {
		conf := val.(*MConfig)
		if conf.dir != "testdir" {
			t.Errorf("SetConfig(%s) = %s; expected %s", path, conf.dir, "testdir")
		}
		if conf.filename != "test.json" {
			t.Errorf("SetConfig(%s) = %s; expected %s", path, conf.filename, "test.json")
		}
	}

}

func TestGet(t *testing.T) {
	path := "testdir/test.json"
	SetConfig("default", path, nil)
	val, err := Config().GetInt("key1")
	if err != nil || val != 1 {
		t.Errorf("GetInt(%s) = %d; expected %d, error:%+v", "key1", val, 1, err)
	}

	val2, err := Config().GetString("key2")
	if err != nil || val2 != "value2" {
		t.Errorf("GetString(%s) = %s; expected %s, error:%+v", "key2", val2, "value2", err)
	}

	val3, err := Config().GetBool("key3")
	if err != nil || val3 != true {
		t.Errorf("GetBool(%s) = %t; expected %t error:%+v", "key3", val3, true, err)
	}

	val4, err := Config().GetFloat("key4")
	if err != nil || val4 != 0.1 {
		t.Errorf("GetFloat(%s) = %f; expected %f, error:%+v", "key4", val4, 0.1, err)
	}

	val5, err := Config().GetString("key5.key6")
	if err != nil || val5 != "value6" {
		t.Errorf("GetString(%s) = %s; expected %s, error:%+v", "key5.key6", val5, "value6", err)
	}

	val6, err := Config().GetStringSlice("key7")
	if err != nil {
		t.Errorf("GetString(%s), error:%+v", "key7", err)
	}
	for i, val := range val6 {
		if val != fmt.Sprintf("key7_%d", i+1) {
			t.Errorf("GetStringSlice(%s) = %s; expected %s", "key7", val, fmt.Sprintf("key7_%d", i+1))
		}
	}

	val7, err := Config().GetIntSlice("key8.key9")
	if err != nil {
		t.Errorf("GetIntSlice(%s), error:%+v", "key8.key9", err)
	}
	for i, val := range val7 {
		if val != i+1 {
			t.Errorf("GetIntSlice(%s) = %d; expected %d", "key8.key9", val, i+1)
		}
	}

	_, err = Config().GetString("key8.not")
	if err != KeyNotFoundErr {
		t.Errorf("not exist key check fail, err:%+v", err)
	}

	val12, err := MultiConfig("default").GetString("key10.key11[0].key12")
	if err != nil || val12 != "value12" {
		t.Errorf("GetString(%s) = %s; expected %s, error:%+v", "key10.key11[0].key12", val12, "value12", err)
	}

	val12, err = MultiConfig("default").GetString("key10.key11[ 0 ].key12")
	if err != nil || val12 != "value12" {
		t.Errorf("GetString(%s) = %s; expected %s, error:%+v", "key10.key11[0].key12", val12, "value12", err)
	}

	val12, err = MultiConfig("default").GetString("key10 .  key11[ 0 ].key12")
	if err != nil || val12 != "value12" {
		t.Errorf("GetString(%s) = %s; expected %s, error:%+v", "key10.key11[0].key12", val12, "value12", err)
	}

	_, err = MultiConfig("default").GetString("key10.key110].key12")
	if err != KeyNotFoundErr {
		t.Errorf("not eixst key check fail, err:%s", err.Error())
	}
}

func TestMonitor(t *testing.T) {
	path := "testdir/test.json"
	SetConfig("default", path, func(s string) {
		fmt.Printf("config:%s content changed!\n", s)
	})
	StartMonitor()
	val, err := Config().GetString("key10.key11[1].key15")
	if err != nil || val != "value15" {
		t.Errorf("GetString(%s) = %s; expected %s, error:%+v", "key10.key11[1].key15", val, "value15", err)
	}

	fileutil.Copy("testdir/test.json", "testdir/test.json.back", true)
	fileutil.Copy("testdir/test2.json", "testdir/test.json", true)
	time.Sleep((monitorDuration + 4) * time.Second)
	val, err = Config().GetString("key10.key11[1].key15")
	if err != nil || val != "value15_new" {
		t.Errorf("GetString(%s) = %s; expected %s, error:%+v", "key10.key11[1].key15", val, "value15_new", err)
	}
	fileutil.Copy("testdir/test.json.back", "testdir/test.json", true)
	StopMonitor()
}
