package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"

	"github.com/go-redis/redis"
	"gopkg.in/ini.v1"
)

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
	cfg.App["port"] = ini.Section("app").Key("port").String()
	cfg.Redis["host"] = ini.Section("redis").Key("host").String()
	cfg.Redis["port"] = ini.Section("redis").Key("port").String()
	cfg.Redis["password"] = ini.Section("redis").Key("password").String()
	cfg.Redis["db"] = ini.Section("redis").Key("db").String()
	// db, _ := strconv.Atoi(cfg.Redis["db"])

	if cfg.App["name"] == "" {
		panic("请配置应用名称")
	}
	if cfg.App["port"] == "" {
		panic("请配置应用端口")
	}

	runtime.GOMAXPROCS(1)
	http.HandleFunc("/index.php", servePHP)             // 设置访问的路由
	http.HandleFunc("/index.html", serveHTML)           // 设置访问的路由
	err = http.ListenAndServe(":"+cfg.App["port"], nil) // 设置监听的端口
	if err != nil {
		panic(err)
	}

}

func servePHP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var data = map[string]string{
		"data": r.Form.Get("data"),
		"x":    r.Form.Get("x"),
		"y":    r.Form.Get("y"),
		"z":    r.Form.Get("z"),
	}
	json, _ := json.Marshal(data)

	key := cfg.App["name"] + "dingtab"
	db, _ := strconv.Atoi(cfg.Redis["db"])
	var rdb = redis.NewClient(&redis.Options{
		Addr:     cfg.Redis["host"] + ":" + cfg.Redis["port"],
		Password: cfg.Redis["password"], // no password set
		DB:       db,                    // use default DB
	})
	err := rdb.Publish(key, string(json)).Err()
	if err != nil {
		panic(err)
	}
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	html := `
	<!DOCTYPE html>
	<html lang="en">
	  <head>
		<meta charset="UTF-8" />
		<meta name="viewport" content="width=device-width,initial-scale=1.0, minimum-scale=1.0, maximum-scale=1.0, user-scalable=no"/>
		<title>Document</title>
		<script
		  type="text/javascript"
		  src="https://code.jquery.com/jquery-3.4.1.min.js"
		></script>
	  </head>
	  <body>
		<div>
		  <img id="screen" src="" width="360px" />
		</div>
		<button id="fresh">更新截图</button>
		<button id="start">启动钉钉</button>
		<button id="text">输入字符</button>
	
		<script type="text/javascript">
		  $(function () {
	
			$("#screen").attr("src", "http://mmstatic.meimeifa.com/` + cfg.App["name"] + `-screen.png?a=" + Math.random());
	
			$("#screen").click(function (e) {
			  var clickX = e.offsetX * 2;
			  var clickY = e.offsetY * 2;
			  $.post(
				"/index.php",
				{
				  x: clickX,
				  y: clickY,
				  z: "0",
				  data: ""
				},
				function (data) {
				  console.log(data);
				},
				"json"
			  );
			});
			$("#start").click(function (e) {
			  $.post(
				"/index.php",
				{
				  x: "0",
				  y: "0",
				  z: "1",
				  data: ""
				},
				function (data) {
				  console.log(data);
				},
				"json"
			  );
			});
			$("#text").click(function (e) {
			  var data = prompt("输入内容");
			  $.post(
				"/index.php",
				{
				  x: "0",
				  y: "0",
				  z: "2",
				  data: data
				},
				function (data) {
				  console.log(data);
				},
				"json"
			  );
			});
		  $("#fresh").click(function (e) {
			  $.post(
				"/index.php",
				{
				  x: "0",
				  y: "0",
				  z: "3",
				  data: ""
				},
				function (data) {
				  console.log(data);
				},
				"json"
			  );
			});
		  });
		</script>
	  </body>
	</html>
	`
	fmt.Fprintf(w, html)
}
