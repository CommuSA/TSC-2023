package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gtank/cryptopasta"
)

var aesKey *[32]byte

func encryptFiles(path string) ([]byte, error) {
	// 判断路径是文件还是文件夹
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	var encrypted []byte
	if fileInfo.IsDir() {
		// 如果是文件夹，则遍历文件夹中的所有文件并加密
		filepath.Walk(path, func(filePath string, fileInfo os.FileInfo, err error) error {
			if !fileInfo.IsDir() {
				fileContent, err := ioutil.ReadFile(filePath)
				if err != nil {
					return err
				}

				encryptedContent, err := cryptopasta.Encrypt(fileContent, aesKey)
				if err != nil {
					return err
				}

				// 添加加密后的文件内容到encrypted切片中
				encrypted = append(encrypted, encryptedContent...)
			}
			return nil
		})
	} else {
		// 如果是文件，则直接加密文件内容
		fileContent, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}

		encrypted, err = cryptopasta.Encrypt(fileContent, aesKey)
		if err != nil {
			return nil, err
		}
	}

	return encrypted, nil
}

func main() {
	folderPath := "../../msg/kddcup.part"
	aesKey = cryptopasta.NewEncryptionKey()

	if encrypted, err := encryptFiles(folderPath); err != nil {
		panic(err)
	} else {
		fmt.Printf("%s", encrypted)
	}
}
