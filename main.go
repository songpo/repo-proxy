package main

import (
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	r := gin.Default()

	r.GET("/repo/maven-group/*subUrl", handleMavenProxy)
	r.GET("/repo/npm-group/*subUrl", handleNpmProxy)
	r.POST("/repo/npm-group/*subUrl", handleNpmProxy)

	r.Run()
}

func handleMavenProxy(c *gin.Context) {
	// 获取要代理的子路径
	subUrl := c.Param("subUrl")

	baseDir := "data/maven"

	baseUrl := "https://maven.aliyun.com/repository/public"

	req, _ := http.NewRequest(c.Request.Method, baseUrl+subUrl, nil)
	req.Header.Set("", "")

	file := handleRepoProxy(c, req, baseDir, baseUrl, subUrl)

	io.Copy(c.Writer, file)
}

func handleNpmProxy(c *gin.Context) {
	// 获取要代理的子路径
	subUrl := c.Param("subUrl")

	baseDir := "data/npm"

	baseUrl := "https://registry.npmmirror.com"

	req, _ := http.NewRequest(c.Request.Method, baseUrl+subUrl, nil)
	req.Header.Set("", "")

	file := handleRepoProxy(c, req, baseDir, baseUrl, subUrl)
	bytes, err := io.ReadAll(file)
	if err != nil {
		return
	}

	result := string(bytes)
	requestPath := strings.Replace(c.Request.RequestURI, subUrl, "", -1)
	localeBaseUrl := "http://" + c.Request.Host + requestPath
	newResult := strings.Replace(result, strings.Join([]string{"\"tarball\":\"", baseUrl}, ""), strings.Join([]string{"\"tarball\":\"", localeBaseUrl}, ""), -1)

	c.Writer.Write([]byte(newResult))
}

func handleRepoProxy(c *gin.Context, req *http.Request, baseDir string, baseUrl string, subUrl string) *os.File {
	fullPath := baseDir + subUrl
	log.Printf("文件路径：%s\n", fullPath)
	fullUrl := baseUrl + subUrl
	log.Printf("下载地址：%s\n", fullUrl)

	// 检查文件是否存在
	if _, err := os.Stat(fullPath); !errors.Is(err, os.ErrNotExist) {
		file, err := os.Open(fullPath)
		if err != nil {
			log.Fatalf("打开文件 %s 失败，%s\n", fullPath, err.Error())
			return nil
		}

		return file
	} else {
		err = os.MkdirAll(filepath.Dir(fullPath), os.ModePerm)
		if err != nil {
			log.Fatalf("创建文件 %s 父级目录失败，%s\n", fullPath, err.Error())
			return nil
		}

		file, err := os.Create(fullPath)
		if err != nil {
			log.Fatalf("打开文件 %s 失败，%s\n", fullPath, err.Error())
			return nil
		}

		var resp = &http.Response{}
		if c.Request.Method == "GET" {
			temp, err := http.Get(fullUrl)
			if err != nil {
				return nil
			}
			resp = temp
		} else if c.Request.Method == "POST" {
			temp, err := http.Post(fullUrl, c.Request.Header.Get("Content-Type"), c.Request.Body)
			if err != nil {
				return nil
			}
			resp = temp
		}

		if err != nil {
			c.String(resp.StatusCode, resp.Status)
			return nil
		}
		defer resp.Body.Close()

		io.Copy(file, resp.Body)

		return file
	}
	return nil
}
