package main

import (
	"fmt"
	"github.com/tedcy/fdfs_client"
)

// go get -u -v github.com/tedcy/fdfs_client
func main() {
	//	使用配置文件，初始化客户端
	clt, err := fdfs_client.NewClientWithConfig("/etc/fdfs/client.conf")
	if err != nil {
		fmt.Println("初始化客户端错误", err)
		return
	}
	//上传文件名
	resp, err := clt.UploadByFilename("/home/neko/Learning/WorkTools/Go_WorkSapce/src/gin-micro-ihome/web/test/p1.png")
	fmt.Println("上传错误", resp, err)
}
