package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/go-redis/redis"
	"github.com/qiniu/api.v7/v7/auth/qbox"
	"github.com/qiniu/api.v7/v7/storage"
	"gopkg.in/ini.v1"
)

// Position 坐标
type Position struct {
	X    string
	Y    string
	Z    string
	Data string
}

// MyPutRet 七牛上传结果集
type MyPutRet struct {
	Key    string
	Hash   string
	Fsize  int
	Bucket string
	Name   string
}

// Cfg 配置
type Cfg struct {
	Sys   map[string]string
	App   map[string]string
	Redis map[string]string
}

var cfg = &Cfg{
	Sys:   map[string]string{"-c": ""},
	App:   map[string]string{},
	Redis: map[string]string{},
}

// FixPNG 修复png图
func FixPNG(data []byte) string {
	len := len(data)
	idx := 0
	count := 0
	idxFirst0d := 0
	idxFirst0a := 0
	is0d := false
	fix := make([]byte, len)
	for i := 0; i < len; i++ {
		b := data[i]
		if b == 0x0d && idxFirst0d == 0 {
			idxFirst0d = i
			is0d = true
		}
		if b == 0x0a && idxFirst0a == 0 {
			idxFirst0a = i
		}
		if i > 2 && b == 0x0a && is0d {
			count++
			idx = idx - (idxFirst0a - idxFirst0d - 1)
			fix[idx] = b
			idx++
		} else {
			fix[idx] = b
			idx++
		}
		if b == 0x0d {
			is0d = true
		} else {
			is0d = false
		}
	}
	// encodeString := base64.StdEncoding.EncodeToString(fix)

	// return "data:image/png;base64," + encodeString
	return string(fix)
}

// ScreenShot 截图
func ScreenShot() {
	// cmd := exec.Command("adb", "shell", "input", "tap", "360", "1250")
	// cmd.Output()

	cmd := exec.Command("adb", "shell", "screencap", "-p")
	out, _ := cmd.Output()
	png := FixPNG(out)

	accessKey := "Oi1FgiRjNvdxYD3-yynJrahxeLF9f3JRY-ZjKwEI"
	secretKey := "kjt7N5A7HCaVEtKN89mWK9Ww0Vv74blp3UMYHJ5d"
	bucket := "meimi"
	key := cfg.App["name"] + "-screen.png"
	putPolicy := storage.PutPolicy{
		Scope:      bucket + ":" + key,
		ReturnBody: `{"key":"$(key)","hash":"$(etag)","fsize":$(fsize),"bucket":"$(bucket)","name":"$(x:name)"}`,
	}
	mac := qbox.NewMac(accessKey, secretKey)
	upToken := putPolicy.UploadToken(mac)
	formUploader := storage.NewFormUploader(&storage.Config{})
	ret := MyPutRet{}
	putExtra := storage.PutExtra{
		Params: map[string]string{
			"x:name": "test png",
		},
	}
	data := []byte(png)
	dataLen := int64(len(data))
	err := formUploader.Put(context.Background(), &ret, upToken, key, bytes.NewReader(data), dataLen, &putExtra)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(key)
}

func main() {
	fmt.Println("start")
	fmt.Println("loading configure")

	len := len(os.Args)
	for k, v := range os.Args {
		if _, ok := cfg.Sys[v]; ok {
			if k+1 < len {
				vv := os.Args[k+1]
				if _, ok2 := cfg.Sys[vv]; ok2 {
					panic(v + " 值不存在")
				} else {
					cfg.Sys[v] = vv
				}
			} else {
				panic(v + " 值不存在")
			}
		}
	}

	configFilePath := cfg.Sys["-c"]
	ini, err := ini.Load(configFilePath)
	if err != nil {
		panic("文件不存在")
	}

	cfg.App["name"] = ini.Section("app").Key("name").String()
	cfg.Redis["host"] = ini.Section("redis").Key("host").String()
	cfg.Redis["port"] = ini.Section("redis").Key("port").String()
	cfg.Redis["password"] = ini.Section("redis").Key("password").String()
	cfg.Redis["db"] = ini.Section("redis").Key("db").String()
	db, _ := strconv.Atoi(cfg.Redis["db"])

	if cfg.App["name"] == "" {
		panic("请配置应用名称")
	}

	runtime.GOMAXPROCS(1)
	var rdb = redis.NewClient(&redis.Options{
		Addr:     cfg.Redis["host"] + ":" + cfg.Redis["port"],
		Password: cfg.Redis["password"], // no password set
		DB:       db,                    // use default DB
		// Addr:     "118.126.104.70:3306",
		// Password: "ChenXuyi123!@",
		// DB:       0,
	})

	key := cfg.App["name"] + "dingtab"
	fmt.Println("pubsub: " + key)
	fmt.Println("image url: http://mmstatic.meimeifa.com/" + cfg.App["name"] + "-screen.png")
	pubsub := rdb.Subscribe(key)
	defer pubsub.Close()

	for {
		msgi, _ := pubsub.ReceiveTimeout(0 * time.Second)
		fmt.Println(msgi)
		switch msg := msgi.(type) {
		case *redis.Message:
			content := msg.Payload
			var pos = &Position{}
			json.Unmarshal([]byte(content), pos)
			switch pos.Z {
			default:
				break
			case "0":
				cmd := exec.Command("adb", "shell", "input", "tap", pos.X, pos.Y)
				cmd.Output()
				break
			case "1":
				cmd := exec.Command("adb", "shell", "am", "start", "-n", "com.alibaba.android.rimet/.biz.LaunchHomeActivity")
				cmd.Output()
				break
			case "2":
				cmd := exec.Command("adb", "shell", "input", "text", pos.Data)
				cmd.Output()
				break
			case "3":
				ScreenShot()
				break
			case "4":
				cmd := exec.Command("adb", "shell", "input", "keyevent", "26")
				cmd.Output()
				cmd = exec.Command("adb", "shell", "input", "swipe", "360", "960", "360", "320")
				cmd.Output()
				break
			}

			break
		default:
			break
		}
	}

}
