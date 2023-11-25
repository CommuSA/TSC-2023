package main

import (
	"fmt"
	"os"
	"time"

	shell "github.com/ipfs/go-ipfs-api"
)

func main() {
	// 连接本地IPFS节点
	sh := shell.NewShell("localhost:5001")

	// 打开本地文件
	filePath := os.Args[1]

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("无法打开文件：%s\n", err.Error())
		return
	}
	defer file.Close()

	// 将文件内容添加到IPFS
	start := time.Now()
	_, err = sh.Add(file)
	end := time.Now()

	if err != nil {
		fmt.Printf("无法将文件添加到IPFS:%s\n", err.Error())
		return
	}
	duration := end.Sub(start).Milliseconds()

	fmt.Printf(" %dms\n", duration)
}

// func main() {
// 	// 连接本地IPFS节点
// 	sh := shell.NewShell("localhost:5001")

// 	// 获取待上传文件/文件夹路径
// 	path := os.Args[1]

// 	// 记录开始上传时间
// 	start := time.Now()

// 	// 上传文件/文件夹
// 	_, err := sh.AddDir(path)
// 	if err != nil {
// 		fmt.Println("Error uploading directory: ", err)
// 		return
// 	}

// 	// 计算上传时间并打印结果
// 	end := time.Now()
// 	duration := end.Sub(start).Milliseconds()
// 	// fmt.Printf("Upload complete. CID: %s\n", cid)
// 	fmt.Printf("%dms\n", duration)
// }
