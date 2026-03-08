package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

func main() {
	// 定义命令行参数
	port := flag.String("p", "5000", "Listen port")
	host := flag.String("host", "127.0.0.1", "Listen host")
	whiteRegexStr := flag.String("white-regex", "^(github\\.com|.*\\.githubusercontent\\.com)$", "Whitelist regex pattern")
	secretPath := flag.String("secret-path", "/secret-path/", "Secret path for the proxy endpoint")

	// 解析命令行参数
	flag.Parse()

	// 编译正则表达式
	whitelistRegex, err := regexp.Compile(*whiteRegexStr)
	if err != nil {
		log.Fatalf("Invalid whitelist regex: %v", err)
	}

	// 规范化 secretPath，保证前后都有斜杠
	path := *secretPath
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 1. 验证路径 (Secret Path)
		if !strings.HasPrefix(r.URL.Path, path) {
			http.Error(w, "Not Found\n", http.StatusNotFound)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			return
		}

		// 2. 提取目标 URL
		targetURLStr := strings.TrimPrefix(r.URL.RequestURI(), path)

		if targetURLStr == "" || targetURLStr == "/" {
			http.Error(w, "Not Found\n", http.StatusNotFound)
			return
		}

		// 兼容反代网关会将连续的斜杠合并成一个的问题
		if !strings.HasPrefix(targetURLStr, "http://") && !strings.HasPrefix(targetURLStr, "https://") {
			if strings.HasPrefix(targetURLStr, "https:/") {
				targetURLStr = strings.Replace(targetURLStr, "https:/", "https://", 1)
			} else if strings.HasPrefix(targetURLStr, "http:/") {
				targetURLStr = strings.Replace(targetURLStr, "http:/", "http://", 1)
			} else {
				targetURLStr = "https://" + targetURLStr
			}
		}

		targetURL, err := url.Parse(targetURLStr)
		if err != nil || targetURL.Scheme == "" || targetURL.Host == "" {
			http.Error(w, "Not Found\n", http.StatusNotFound)
			return
		}

		// 3. 核心：正则白名单校验
		if !whitelistRegex.MatchString(targetURL.Host) {
			log.Printf("Blocked access: [%s] does not match whitelist", targetURL.Host)
			http.Error(w, "Not Found\n", http.StatusNotFound)
			return
		}

		log.Printf("Proxying request to: %s", targetURLStr)

		// 4. 发送代理请求
		proxyReq, err := http.NewRequest(r.Method, targetURLStr, r.Body)
		if err != nil {
			http.Error(w, "Not Found\n", http.StatusNotFound)
			return
		}

		// 透传请求头
		for k, vv := range r.Header {
			if strings.ToLower(k) != "host" {
				for _, v := range vv {
					proxyReq.Header.Add(k, v)
				}
			}
		}

		client := &http.Client{}
		resp, err := client.Do(proxyReq)
		if err != nil {
			http.Error(w, "Not Found\n", http.StatusNotFound)
			return
		}
		defer resp.Body.Close()

		// 回传响应头
		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)

		// 流式回传
		io.Copy(w, resp.Body)
	})

	listenAddr := fmt.Sprintf("%s:%s", *host, *port)
	fmt.Println("=======================================")
	fmt.Printf("🚀 micro-path-proxy is running!\n")
	fmt.Printf("🔒 Secret Path : %s\n", path)
	fmt.Printf("🛡️  Whitelist   : %s (Regex mode)\n", *whiteRegexStr)
	fmt.Printf("📡 Listen Addr : http://%s\n", listenAddr)
	fmt.Println("=======================================")

	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		log.Fatal(err)
	}
}
