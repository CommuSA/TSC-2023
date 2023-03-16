package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fentec-project/gofe/abe"
	shell "github.com/ipfs/go-ipfs-api"

	"github.com/gtank/cryptopasta"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// MyChaincode defines the chaincode structure
type MyChaincode struct {
	contractapi.Contract
}

var aesKey *[32]byte

type AbeUserPara struct {
	abefame *abe.FAME           `json:"ABEFAME"`
	mpk     *abe.FAMEPubKey     `json:"mpk"`
	keys    *abe.FAMEAttribKeys `json:"keys"`
}

// type AbePeerParaDB struct {
// 	ID      string `json:"ID"`
// 	ABEFAME string `json:"ABEFAME"`
// 	mpk     string `json:"mpk"`
// 	msk     string `json:"msk"`
// 	msp     string `json:"msp"`
// }

type Circuit struct {
	leftOperand  string
	operator     string
	rightOperand string
}

type MessageKey struct {
	ID  string `json:"ID"`
	Key string `json:"key"`
	Cid string `json:"cid"`
}

//------------------------------------------------------------ABE SETUP-------------------------------------------------------------------//

// Randomly generate gate logic statements
func generateCircuits(numCircuits int) []Circuit {
	var circuits []Circuit
	for i := 0; i < numCircuits; i++ {
		left := strconv.Itoa(rand.Intn(10000))
		right := strconv.Itoa(rand.Intn(10000))
		operator := ""
		switch rand.Intn(2) {
		case 0:
			operator = "AND"
		case 1:
			operator = "OR"
		}
		circuits = append(circuits, Circuit{left, operator, right})
	}
	return circuits
}

// Converts gate logic statements to strings
func (c Circuit) String() string {
	return fmt.Sprintf("(%s %s %s)", c.leftOperand, c.operator, c.rightOperand)
}

// Converts gate logic statements to strings
func generateAttributes() []string {
	var attributes []string
	for i := 0; i < rand.Intn(7)+2; i++ {
		length := rand.Intn(8) + 1
		letters := make([]rune, length)
		for j := range letters {
			letters[j] = rune(rand.Intn(26) + 97)
		}
		attributes = append(attributes, string(letters))
	}
	return attributes
}

// Converts gate logic statements to strings
func generateLogicOperator() string {
	operators := []string{"AND", "OR"}
	return operators[rand.Intn(len(operators))]
}

func policySetup(num int) string {
	// Converts gate logic statements to strings
	numCircuits := num

	// Randomly generate gate logic statements
	circuits := generateCircuits(numCircuits)

	// Randomly generate gate logic statements
	attributes := generateAttributes()

	// Assemble gate logic statements and attribute content
	var parts []string
	for _, circuit := range circuits {
		parts = append(parts, circuit.String())
	}
	parts = append(parts, strings.Join(attributes, " "+generateLogicOperator()+" "))
	finalStr := strings.Join(parts, " AND ")

	return finalStr
}

// Resolves out property names and Boolean operators
func evaluateLogicCircuit(circuit string, numMax int) [][]string {

	properties := make([][]string, 0)
	for _, condition := range strings.Split(circuit, " AND ") {
		props := make([]string, 0)
		props = append(props, strings.Split(condition, " OR ")...)
		// for _, p := range strings.Split(condition, " OR ") {
		// 	props = append(props, p)
		// }
		properties = append(properties, props)
	}

	cartesianProduct := make([][]string, 1)
	for _, props := range properties {
		if len(props) > 1 {
			tmp := make([][]string, 0)
			for _, prop := range props {
				for _, cp := range cartesianProduct {
					newCP := append([]string{prop}, cp...)
					tmp = append(tmp, newCP)
				}
			}
			cartesianProduct = tmp
		} else {
			for i := range cartesianProduct {
				cartesianProduct[i] = append(cartesianProduct[i], props[0])
			}
		}
	}

	// Resolves out property names and Boolean operators
	results := make([][]string, 0)
	i := 0
	for _, props := range cartesianProduct {
		isMatched := true
		for _, condition := range properties {
			isConditionMatched := false
			for _, p := range condition {
				if containsString(props, p) {
					isConditionMatched = true
					break
				}
			}
			if !isConditionMatched {
				isMatched = false
				break
			}
		}
		if isMatched {
			results = append(results, props)
			i++

			if i >= numMax {
				return results
			}
		}
	}

	return results
}

func containsString(strings []string, s string) bool {
	for _, str := range strings {
		if str == s {
			return true
		}
	}
	return false
}

func (mc *MyChaincode) ParaExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	ParaJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return ParaJSON != nil, nil
}

func (mc *MyChaincode) PrepareAbe(ctx contractapi.TransactionContextInterface, numMagnitude int, AttributesNeeded int) error {
	// txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	// seed := int64(txTimestamp.Seconds) + int64(txTimestamp.Nanos)
	rand.Seed(time.Now().Unix())
	relay := abe.NewFAME()
	// Generate an ABE key pair
	mpk, msk, err := relay.GenerateMasterKeys()
	if err != nil {
		panic(err)
	}
	policy := policySetup(numMagnitude)
	msp, _ := abe.BooleanToMSP(policy, false) // The MSP structure defining the policy
	mskJSON, _ := json.Marshal(msk)
	mpkJSON, _ := json.Marshal(mpk)
	mspJSON, _ := json.Marshal(msp)
	abeJSON, _ := json.Marshal(relay)

	err = ctx.GetStub().PutState("AbeParaMsk", mskJSON)
	if err != nil {
		return err

	}
	err = ctx.GetStub().PutState("AbeParaMpk", mpkJSON)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState("AbeParaMsp", mspJSON)
	if err != nil {
		return err
	}
	err = ctx.GetStub().PutState("AbeParaFame", abeJSON)
	if err != nil {
		return err
	}

	return nil

	// return peerAbeParaPut

	// qualifiedAttributes := evaluateLogicCircuit(policy, AttributesNeeded)

	// for _, subArr := range qualifiedAttributes {
	// 	for i, str := range subArr {
	// 		str = strings.ReplaceAll(str, "(", "")
	// 		str = strings.ReplaceAll(str, ")", "")
	// 		subArr[i] = str
	// 	}
	// }

	// for i, qualifiedAttribute := range qualifiedAttributes {
	// 	keys, _ := relay.GenerateAttribKeys(qualifiedAttribute, msk)
	// 	keysJSON, err := json.Marshal(keys)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	MemberEnv := fmt.Sprintf("Member%dEnv", i)
	// 	file, err := os.Create(MemberEnv)
	// 	if err != nil {
	// 		fmt.Println(err)
	// 		return
	// 	}

	// 	_, err = fmt.Fprintf(file, "ABEFAME=%s\nMPK=%s\nSecKeys=%s", abeJSON, mpkJSON, keysJSON)
	// 	if err != nil {
	// 		fmt.Println(err)
	// 		return
	// 	}
	// }

}

func (mc *MyChaincode) GetPeerAbePara(ctx contractapi.TransactionContextInterface, id string) (string, error) {
	peerAbeParaJson, err := ctx.GetStub().GetState(id)

	if err != nil {
		return "", fmt.Errorf("failed to read from world state: %v", err)
	}
	if peerAbeParaJson == nil {
		return "", fmt.Errorf("the para %s does not exist", id)
	}

	abeStr := string(peerAbeParaJson)

	return abeStr, nil
}

//--------------------------------------------------------------------------------------------------------//

func clearFolder(path string) error {
	d, err := os.Open(path)
	if err != nil {
		return err
	}
	defer d.Close()

	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, name := range names {
		err = os.RemoveAll(filepath.Join(path, name))
		if err != nil {
			return err
		}
	}

	return nil
}

func downloadFile(url string, filepath string) error {
	// Create file
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the data to file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// ===================================      AES algorithm     ======================================//
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

func aesKeyEncrypt(ctx contractapi.TransactionContextInterface, key *[32]byte, ABEFAME []byte, MpkJson []byte, MspJson []byte) (string, error) {
	var abeFame *abe.FAME
	var Mpk *abe.FAMEPubKey
	var Msp *abe.MSP

	err := json.Unmarshal(ABEFAME, &abeFame)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(MpkJson, &Mpk)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(MspJson, &Msp)
	if err != nil {
		panic(err)
	}

	CT, _ := abeFame.Encrypt(string(key[:]), Msp, Mpk)
	CTJson, _ := json.Marshal(CT)
	CTstr := string(CTJson)

	return CTstr, nil

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

//-----------------------------------------------------------------------------------------------------------------------

func (mc *MyChaincode) PrepareFile(ctx contractapi.TransactionContextInterface, fileUrl string) error {
	aesKey = cryptopasta.NewEncryptionKey()

	clearFolder("/tmp/")
	// Download file
	// fileUrl := "http://kdd.ics.uci.edu/databases/kddcup99/kddcup.newtestdata_10_percent_unlabeled.gz"
	fileName := "/tmp/kddcup.gz"
	err := downloadFile(fileUrl, fileName)
	if err != nil {
		return err
	}

	// Unzip file
	cmd := exec.Command("gunzip", "-k", fileName)
	err = cmd.Run()
	if err != nil {
		return err
	}
	defer os.Remove(fileName)

	// Split file
	file, err := os.Open(strings.TrimSuffix(fileName, ".gz"))
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())

	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	var chunkSize int64 = fileInfo.Size() / 10

	for i := 0; i < 10; i++ {
		chunkFileName := fmt.Sprintf("/tmp/kddcup.%d", i+1)
		chunkFile, err := os.OpenFile(chunkFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer chunkFile.Close()

		written, err := io.CopyN(chunkFile, file, chunkSize)
		if err != nil && err != io.EOF {
			return err
		}

		fmt.Printf("Wrote %d bytes to %s\n", written, chunkFileName)
	}

	return nil
}

// UploadFile encrypts the file and upload to IPFS, and return the CID
func (mc *MyChaincode) UploadFile(ctx contractapi.TransactionContextInterface, idMKey string, ip string) error {
	peerAbeParaMpk, err := ctx.GetStub().GetState("AbeParaMpk")
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if peerAbeParaMpk == nil {
		return fmt.Errorf("AbeParaMpk does not exist")
	}
	peerAbeParaMsp, err := ctx.GetStub().GetState("AbeParaMsp")
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	peerAbeParaFame, err := ctx.GetStub().GetState("AbeParaFame")
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}

	sh := shell.NewShell(ip)
	err = encryptFolder("/tmp")
	if err != nil {
		return err
	}

	cid, err := sh.AddDir("/tmp")
	if err != nil {
		return err
	}
	keyEn, err := aesKeyEncrypt(ctx, aesKey, peerAbeParaFame, peerAbeParaMpk, peerAbeParaMsp)
	if err != nil {
		return err
	}

	MessageSend := MessageKey{
		ID:  idMKey,
		Key: keyEn,
		Cid: cid,
	}
	peerMessageKeyJSON, err := json.Marshal(MessageSend)
	if err != nil {
		return err
	}
	err = ctx.GetStub().PutState(idMKey, peerMessageKeyJSON)
	return err

}

func main() {
	cc, err := contractapi.NewChaincode(&MyChaincode{})
	if err != nil {
		panic(fmt.Sprintf("Error creating my chaincode: %v", err))
	}

	if err := cc.Start(); err != nil {
		panic(fmt.Sprintf("Error starting my chaincode: %v", err))
	}
}
