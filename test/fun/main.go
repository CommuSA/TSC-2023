package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/gtank/cryptopasta"
)

var aesKey *[32]byte

func encryptFile(filename string) error {
	plaintext, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	ciphertext, err := cryptopasta.Encrypt(plaintext, aesKey)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(filename, ciphertext, 0644); err != nil {
		return err
	}

	return nil
}

func decryptFile(filename string) error {
	ciphertext, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	plaintext, err := cryptopasta.Decrypt(ciphertext, aesKey)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(filename, plaintext, 0644); err != nil {
		return err
	}

	return nil
}

func decryptFolder(folderPath string) error {
	startTime := time.Now()

	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			if err := decryptFile(path); err != nil {
				return err
			}
		}

		return nil
	})

	elapsedTime := time.Since(startTime)

	fmt.Printf("Decryption complete. Time elapsed: %v\n", elapsedTime)

	return err
}

func encryptFolder(folderPath string) error {
	aesKey = cryptopasta.NewEncryptionKey()

	startTime := time.Now()

	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			if err := encryptFile(path); err != nil {
				return err
			}
		}

		return nil
	})

	elapsedTime := time.Since(startTime)

	fmt.Printf("Encryption complete. Time elapsed: %v\n", elapsedTime)

	return err
}

func main() {
	folderPath := "../../msg/kddcup.part"

	if err := encryptFolder(folderPath); err != nil {
		panic(err)
	}
	if err := decryptFolder(folderPath); err != nil {
		panic(err)
	}
}
