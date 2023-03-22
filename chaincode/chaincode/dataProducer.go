/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chaincode

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gtank/cryptopasta"

	"github.com/fentec-project/gofe/abe"
	shell "github.com/ipfs/go-ipfs-api"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

const paraCollection = "assetCollection"
const RequestInformationObjectType = "RequestInformation"

// SmartContract of this fabric sample
type SmartContract struct {
	contractapi.Contract
}

// para describes main para details that are visible to all organizations
type Asset struct {
	Type  string `json:"objectType"` //Type is used to distinguish the various types of objects in state database
	ID    string `json:"paraID"`
	Color string `json:"color"`
	Size  int    `json:"size"`
	Owner string `json:"owner"`
}

// paraPrivateDetails describes details that are private to owners
type ApplyValue struct {
	ID        string `json:"paraID"`
	ApplyInfo string `json:"applyInfo"`
}

// ApplersInfo describes the User agreement returned by ReadRequestInformation
type ApplersInfo struct {
	ID     string `json:"paraID"`
	UserID string `json:"UserID"`
}

type AbeUserPara struct {
	ID      string `json:"ID"`
	Owner   string `json:"Owner"`
	ABEFAME string `json:"ABEFAME"`
	Mpk     string `json:"mpk"`
	Keys    string `json:"keys"`
}

type AbePeerPara struct {
	ID      string `json:"ID"`
	Owner   string `json:"Owner"`
	ABEFAME string `json:"ABEFAME"`
	Mpk     string `json:"Mpk"`
	Msk     string `json:"Msk"`
	Msp     string `json:"Msp"`
}

type Circuit struct {
	leftOperand  string
	operator     string
	rightOperand string
}

type MessageKey struct {
	ID     string `json:"ID"`
	AesKey string `json:"key"`
	Cid    string `json:"cid"`
	Owner  string `json:"Owner"`
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

func (s *SmartContract) ParaExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	ParaJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return ParaJSON != nil, nil
}

func (s *SmartContract) PrepareAbe(ctx contractapi.TransactionContextInterface) error {

	transientMap, err := ctx.GetStub().GetTransient()
	if err != nil {
		return fmt.Errorf("error getting transient: %v", err)
	}
	transientParaJSON, ok := transientMap["abe_properties"]
	if !ok {
		// log error to stdout
		return fmt.Errorf("para not found in the transient map input")
	}
	type AbePropertiesDef struct {
		NumMagnitude     int    `json:"numMagnitude"`
		AttributesNeeded int    `json:"AttributesNeeded"`
		Id               string `json:"id"`
	}
	var abeInput AbePropertiesDef
	err = json.Unmarshal(transientParaJSON, &abeInput)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	// txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	// seed := int64(txTimestamp.Seconds) + int64(txTimestamp.Nanos)
	// exists, err := s.ParaExists(ctx, id)

	// if err != nil {
	// 	return err
	// }
	// if exists {
	// 	return fmt.Errorf("the para %s already exists", id)
	// }

	rand.Seed(time.Now().Unix())
	// start = time.Now()
	relay := abe.NewFAME()
	// Generate an ABE key pair
	mpk, msk, err := relay.GenerateMasterKeys()
	if err != nil {
		panic(err)
	}
	policy := policySetup(abeInput.NumMagnitude)
	msp, _ := abe.BooleanToMSP(policy, false) // The MSP structure defining the policy

	// end = time.Now()
	//------------------------------------------------------------------
	//------------------------------------------------------------------
	// 记录abe密钥生成时间，不同的属性策略和属性数量  6 8 10 12 14
	// abe加密时间、解密时间、用户密钥的生成时间
	// user数量为1
	//------------------------------------------------------------------
	//------------------------------------------------------------------

	//------------------------------------------------------------------
	//------------------------------------------------------------------
	// test2 设置用户数为 2 4 6 8 10 12 14
	// 设置不同灵活度的访问厕策略和属性数量对用户获取请求数据总时间的影响
	// 请求数据大小固定 文件为kddcup.newtestdata_10_percent_unlabeled.gz 45mb
	//实验1和实验2设置的属性策略一致，4总属性策略，尽量可以从中看出属性策略灵活度的区别，（or 的数量对实验结果的影响） 不同的属性策略和属性数量  6 8 10 12 14 16 18
	// 访问策略在文章中的交待一下，需要复制出来
	//
	// 1 - 四种不同风格的访问策略，不同的属性增加方式
	// 前面5个or，后面不同的访问策略增加方式（）
	// - 1.1 支持改变属性数量
	// 2 - 不设置访问策略的变化，设置不同的数据大小
	//******************************************************************
	//------------------------------------------------------------------
	//------------------------------------------------------------------
	mskJSON, _ := json.Marshal(msk)
	mpkJSON, _ := json.Marshal(mpk)
	mspJSON, _ := json.Marshal(msp)
	abeJSON, _ := json.Marshal(relay)
	clientID, err := submittingClientIdentity(ctx)
	if err != nil {
		return err
	}
	err = verifyClientOrgMatchesPeerOrg(ctx)
	if err != nil {
		return fmt.Errorf("PrepareAbe cannot be performed: Error %v", err)
	}

	peerAbeParaPut := AbePeerPara{
		ID:      abeInput.Id,
		Owner:   clientID,
		ABEFAME: string(abeJSON),
		Mpk:     string(mpkJSON),
		Msk:     string(mskJSON),
		Msp:     string(mspJSON),
	}
	orgCollection, err := getCollectionName(ctx)
	if err != nil {
		return fmt.Errorf("failed to infer private collection name for the org: %v", err)
	}

	peerAbeParaPutJSON, err := json.Marshal(peerAbeParaPut)
	if err != nil {
		return err
	}
	err = ctx.GetStub().PutPrivateData(orgCollection, peerAbeParaPut.ID, peerAbeParaPutJSON)
	if err != nil {
		return fmt.Errorf("failed to put abePara for master  %v", err)
	}

	qualifiedAttributes := evaluateLogicCircuit(policy, abeInput.AttributesNeeded)

	for _, subArr := range qualifiedAttributes {
		for i, str := range subArr {
			str = strings.ReplaceAll(str, "(", "")
			str = strings.ReplaceAll(str, ")", "")
			subArr[i] = str
		}
	}

	for i, qualifiedAttribute := range qualifiedAttributes {
		keys, _ := relay.GenerateAttribKeys(qualifiedAttribute, msk)
		keysJSON, err := json.Marshal(keys)
		if err != nil {
			panic(err)
		}
		mid := fmt.Sprintf("MemberKey%d", i)

		UerAbePara := AbeUserPara{
			ID:      mid,
			Owner:   clientID,
			ABEFAME: string(abeJSON),
			Mpk:     string(mpkJSON),
			Keys:    string(keysJSON),
		}

		UerAbeParaJSON, err := json.Marshal(UerAbePara)
		if err != nil {
			return err
		}
		err = ctx.GetStub().PutPrivateData(orgCollection, mid, UerAbeParaJSON)
		if err != nil {
			return fmt.Errorf("failed to put userAbePara : %v", err)
		}

	}
	return nil
}

func (s *SmartContract) GetPeerAbePara(ctx contractapi.TransactionContextInterface, collection string, id string) (*AbePeerPara, error) {
	peerAbeParaJson, err := ctx.GetStub().GetPrivateData(collection, id) // Get the para from chaincode state

	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if peerAbeParaJson == nil {
		return nil, fmt.Errorf("the para %s does not exist", id)
	}

	var peerAbePara *AbePeerPara
	err = json.Unmarshal(peerAbeParaJson, &peerAbePara)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}
	return peerAbePara, nil
}

//------------------------------------------------------------------------------------------------//
//------------------------------------------------------------------------------------------------//
//------------------------------------------------------------------------------------------------//
//-----------------------------------------------ABE----------------------------------------------//
//------------------------------------------------------------------------------------------------//
//------------------------------------------------------------------------------------------------//

// ------------------------------------------------------------------------------------------------//
// ----------------------------------------------Folder--------------------------------------------//
// ------------------------------------------------------------------------------------------------//
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
func encryptFile(filename string, aesKey *[32]byte) error {
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

func aesKeyEncrypt(key *[32]byte, peerAbePara AbePeerPara) (string, error) {
	ABEFAME := peerAbePara.ABEFAME
	MpkStr := peerAbePara.Mpk
	MspStr := peerAbePara.Msp
	var abeFame *abe.FAME
	var Mpk *abe.FAMEPubKey
	var Msp *abe.MSP

	err := json.Unmarshal([]byte(ABEFAME), &abeFame)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal([]byte(MpkStr), &Mpk)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal([]byte(MspStr), &Msp)
	if err != nil {
		panic(err)
	}

	// start := time.Now()
	CT, _ := abeFame.Encrypt(string(key[:]), Msp, Mpk) // 使用abe对aeskey 加密    //test2:abe加密测试
	// end := time.Now()

	CTJson, _ := json.Marshal(CT)
	CTstr := string(CTJson)

	return CTstr, nil

}

func encryptFolder(folderPath string, aesKey *[32]byte) error {
	startTime := time.Now()

	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			if err := encryptFile(path, aesKey); err != nil {
				return err
			}
		}

		return nil
	})

	elapsedTime := time.Since(startTime)

	fmt.Printf("Encryption complete. Time elapsed: %v\n", elapsedTime)

	return err
}
func (s *SmartContract) PrepareFile(ctx contractapi.TransactionContextInterface, fileUrl string) error {
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

func (s *SmartContract) UploadFile(ctx contractapi.TransactionContextInterface, idAbe string, ip string) error {
	aesKey := cryptopasta.NewEncryptionKey()

	transientMap, err := ctx.GetStub().GetTransient()
	if err != nil {
		return fmt.Errorf("error getting transient: %v", err)
	}

	// para properties are private, therefore they get passed in transient field, instead of func args
	transientparaJSON, ok := transientMap["file_properties"]
	if !ok {
		// log error to stdout
		return fmt.Errorf("para not found in the transient map input")
	}

	type paraTransientInput struct {
		ID string `json:"paraID"`
	}

	var paraInput paraTransientInput
	err = json.Unmarshal(transientparaJSON, &paraInput)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	if len(paraInput.ID) == 0 {
		return fmt.Errorf("paraID field must be a non-empty string")
	}

	// Check if para already exists
	paraAsBytes, err := ctx.GetStub().GetPrivateData(paraCollection, paraInput.ID)
	if err != nil {
		return fmt.Errorf("failed to get para: %v", err)
	} else if paraAsBytes != nil {
		fmt.Println("para already exists: " + paraInput.ID)
		return fmt.Errorf("this para already exists: " + paraInput.ID)
	}

	// Get ID of submitting client identity
	clientID, err := submittingClientIdentity(ctx)
	if err != nil {
		return err
	}

	// Verify that the client is submitting request to peer in their organization
	// This is to ensure that a client from another org doesn't attempt to read or
	// write private data from this peer.
	err = verifyClientOrgMatchesPeerOrg(ctx)
	if err != nil {
		return fmt.Errorf("CreatePara cannot be performed: Error %v", err)
	}
	collectionOwner, err := getCollectionName(ctx) // get owner collection from caller identity
	if err != nil {
		return fmt.Errorf("failed to infer private collection name for the org: %v", err)
	}

	peerAbeParaJson, err := ctx.GetStub().GetPrivateData(collectionOwner, idAbe) // Get the para from chaincode state

	if err != nil {
		panic(err)
	}

	if peerAbeParaJson == nil {
		return fmt.Errorf("%v does not exist", idAbe)
	}
	var peerAbePara AbePeerPara
	err = json.Unmarshal(peerAbeParaJson, &peerAbePara)
	if err != nil {
		panic(err)
	}

	sh := shell.NewShell(ip)
	err = encryptFolder("/tmp", aesKey)
	if err != nil {
		return err
	}

	cid, err := sh.AddDir("/tmp")
	if err != nil {
		return err
	}

	keyEn, err := aesKeyEncrypt(aesKey, peerAbePara)
	if err != nil {
		return err
	}

	MessageSend := MessageKey{
		ID:     paraInput.ID,
		Owner:  clientID,
		AesKey: keyEn,
		Cid:    cid,
	}

	paraJSONasBytes, err := json.Marshal(MessageSend)
	if err != nil {
		return fmt.Errorf("failed to marshal para into JSON: %v", err)
	}

	log.Printf("UploadFile Put: collection %v, ID %v, owner %v", paraCollection, paraInput.ID, clientID)

	err = ctx.GetStub().PutPrivateData(paraCollection, paraInput.ID, paraJSONasBytes)
	if err != nil {
		return fmt.Errorf("failed to put para into private data collection: %v", err)
	}
	return nil
}

//-----------------------------------------------------------------------------------------------------------------------

//------------------------------------------------------------------------------------------------//
//------------------------------------------------------------------------------------------------//
//------------------------------------------------------------------------------------------------//

// SetUserAbePara applys the para to the new owner by setting a new owner ID
func (s *SmartContract) SetUserAbePara(ctx contractapi.TransactionContextInterface) error {

	transientMap, err := ctx.GetStub().GetTransient()
	if err != nil {
		return fmt.Errorf("error getting transient %v", err)
	}

	// para properties are private, therefore they get passed in transient field
	transientApplyJSON, ok := transientMap["para_owner"]
	if !ok {
		return fmt.Errorf("para owner not found in the transient map")
	}

	type paraApplyTransientInput struct {
		ID string `json:"paraID"`
	}

	var paraApplyInput paraApplyTransientInput
	err = json.Unmarshal(transientApplyJSON, &paraApplyInput)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	if len(paraApplyInput.ID) == 0 {
		return fmt.Errorf("paraID field must be a non-empty string")
	}
	log.Printf("SetUserAbePara: verify para exists ID %v", paraApplyInput.ID)

	// Get collection name for this organization
	ownersCollection, err := getCollectionName(ctx)
	if err != nil {
		return fmt.Errorf("failed to infer private collection name for the org: %v", err)
	}
	log.Printf("---------getCollectionName: getCollectionName for org1 %v-----------", paraApplyInput.ID)

	// Read para from the private data collection
	para, err := s.GetUserAbePara(ctx, ownersCollection, paraApplyInput.ID)
	log.Printf("---------GetUserAbePara: GetUserAbePara for org1 %v-----------", paraApplyInput.ID)

	if err != nil {
		return fmt.Errorf("error reading para: %v", err)
	}
	if para == nil {
		return fmt.Errorf("%v does not exist", paraApplyInput.ID)
	}
	// Verify that the client is submitting request to peer in their organization
	err = verifyClientOrgMatchesPeerOrg(ctx)
	log.Printf("---------verifyClientOrgMatchesPeerOrg: verifyClientOrgMatchesPeerOrg for org1 %v-----------", paraApplyInput.ID)

	if err != nil {
		return fmt.Errorf("SetUserAbePara cannot be performed: %v", err)
	}

	RequestInformation, err := s.ReadRequestInformation(ctx, paraApplyInput.ID)
	log.Printf("---------RequestInformation: RequestInformation for org1 %v-----------", paraApplyInput.ID)

	if err != nil {
		return fmt.Errorf("failed ReadRequestInformation to find UserID: %v", err)
	}
	if RequestInformation.UserID == "" {
		return fmt.Errorf("UserID not found in ApplersInfo for %v", paraApplyInput.ID)
	}

	// apply para in private data collection to new owner
	para.Owner = RequestInformation.UserID
	log.Printf("--------- to new owner:  to new owner for org1 %v-----------", paraApplyInput.ID)

	paraJSONasBytes, err := json.Marshal(para)
	if err != nil {
		return fmt.Errorf("failed marshalling para %v: %v", paraApplyInput.ID, err)
	}

	log.Printf("SetUserAbePara Put: collection %v, ID %v", paraCollection, paraApplyInput.ID)
	err = ctx.GetStub().PutPrivateData(paraCollection, paraApplyInput.ID, paraJSONasBytes) //rewrite the para
	if err != nil {
		return err
	}

	// Delete the para appraised value from this organization's private data collection
	err = ctx.GetStub().DelPrivateData(ownersCollection, paraApplyInput.ID)
	if err != nil {
		return err
	}

	// Delete the apply agreement from the para collection
	ApplyKey, err := ctx.GetStub().CreateCompositeKey(RequestInformationObjectType, []string{paraApplyInput.ID})
	if err != nil {
		return fmt.Errorf("failed to create composite key: %v", err)
	}

	err = ctx.GetStub().DelPrivateData(paraCollection, ApplyKey)
	if err != nil {
		return err
	}

	return nil

}

// Deletepara can be used by the owner of the para to delete the para
func (s *SmartContract) Deletepara(ctx contractapi.TransactionContextInterface) error {

	transientMap, err := ctx.GetStub().GetTransient()
	if err != nil {
		return fmt.Errorf("error getting transient: %v", err)
	}

	// para properties are private, therefore they get passed in transient field
	transientDeleteJSON, ok := transientMap["para_delete"]
	if !ok {
		return fmt.Errorf("para to delete not found in the transient map")
	}

	type paraDelete struct {
		ID string `json:"paraID"`
	}

	var paraDeleteInput paraDelete
	err = json.Unmarshal(transientDeleteJSON, &paraDeleteInput)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	if len(paraDeleteInput.ID) == 0 {
		return fmt.Errorf("paraID field must be a non-empty string")
	}

	// Verify that the client is submitting request to peer in their organization
	err = verifyClientOrgMatchesPeerOrg(ctx)
	if err != nil {
		return fmt.Errorf("Deletepara cannot be performed: Error %v", err)
	}

	log.Printf("Deleting para: %v", paraDeleteInput.ID)
	valAsbytes, err := ctx.GetStub().GetPrivateData(paraCollection, paraDeleteInput.ID) //get the para from chaincode state
	if err != nil {
		return fmt.Errorf("failed to read para: %v", err)
	}
	if valAsbytes == nil {
		return fmt.Errorf("para not found: %v", paraDeleteInput.ID)
	}

	ownerCollection, err := getCollectionName(ctx) // Get owners collection
	if err != nil {
		return fmt.Errorf("failed to infer private collection name for the org: %v", err)
	}

	// Check the para is in the caller org's private collection
	valAsbytes, err = ctx.GetStub().GetPrivateData(ownerCollection, paraDeleteInput.ID)
	if err != nil {
		return fmt.Errorf("failed to read para from owner's Collection: %v", err)
	}
	if valAsbytes == nil {
		return fmt.Errorf("para not found in owner's private Collection %v: %v", ownerCollection, paraDeleteInput.ID)
	}

	// delete the para from state
	err = ctx.GetStub().DelPrivateData(paraCollection, paraDeleteInput.ID)
	if err != nil {
		return fmt.Errorf("failed to delete state: %v", err)
	}

	// Finally, delete private details of para
	err = ctx.GetStub().DelPrivateData(ownerCollection, paraDeleteInput.ID)
	if err != nil {
		return err
	}

	return nil

}

// Purgepara can be used by the owner of the para to delete the para
// Trigger removal of the para
func (s *SmartContract) Purgepara(ctx contractapi.TransactionContextInterface) error {

	transientMap, err := ctx.GetStub().GetTransient()
	if err != nil {
		return fmt.Errorf("Error getting transient: %v", err)
	}

	// para properties are private, therefore they get passed in transient field
	transientDeleteJSON, ok := transientMap["para_purge"]
	if !ok {
		return fmt.Errorf("para to purge not found in the transient map")
	}

	type paraPurge struct {
		ID string `json:"paraID"`
	}

	var paraPurgeInput paraPurge
	err = json.Unmarshal(transientDeleteJSON, &paraPurgeInput)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	if len(paraPurgeInput.ID) == 0 {
		return fmt.Errorf("paraID field must be a non-empty string")
	}

	// Verify that the client is submitting request to peer in their organization
	err = verifyClientOrgMatchesPeerOrg(ctx)
	if err != nil {
		return fmt.Errorf("Purgepara cannot be performed: Error %v", err)
	}

	log.Printf("Purging para: %v", paraPurgeInput.ID)

	// Note that there is no check here to see if the id exist; it might have been 'deleted' already
	// so a check here is pointless. We would need to call purge irrespective of the result
	// A delete can be called before purge, but is not essential

	ownerCollection, err := getCollectionName(ctx) // Get owners collection
	if err != nil {
		return fmt.Errorf("failed to infer private collection name for the org: %v", err)
	}

	// delete the para from state
	err = ctx.GetStub().PurgePrivateData(paraCollection, paraPurgeInput.ID)
	if err != nil {
		return fmt.Errorf("failed to purge state from para collection: %v", err)
	}

	// Finally, delete private details of para
	err = ctx.GetStub().PurgePrivateData(ownerCollection, paraPurgeInput.ID)
	if err != nil {
		return fmt.Errorf("failed to purge state from owner collection: %v", err)
	}

	return nil

}

// DeleteApplyAgreement can be used by the User to withdraw a proposal from
// the para collection and from his own collection.
func (s *SmartContract) DeleteApplyAgreement(ctx contractapi.TransactionContextInterface) error {

	transientMap, err := ctx.GetStub().GetTransient()
	if err != nil {
		return fmt.Errorf("error getting transient: %v", err)
	}

	// para properties are private, therefore they get passed in transient field
	transientDeleteJSON, ok := transientMap["agreement_delete"]
	if !ok {
		return fmt.Errorf("para to delete not found in the transient map")
	}

	type paraDelete struct {
		ID string `json:"paraID"`
	}

	var paraDeleteInput paraDelete
	err = json.Unmarshal(transientDeleteJSON, &paraDeleteInput)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	if len(paraDeleteInput.ID) == 0 {
		return fmt.Errorf("transient input ID field must be a non-empty string")
	}

	// Verify that the client is submitting request to peer in their organization
	err = verifyClientOrgMatchesPeerOrg(ctx)
	if err != nil {
		return fmt.Errorf("DeleteApplyAgreement cannot be performed: Error %v", err)
	}
	// Delete private details of agreement
	orgCollection, err := getCollectionName(ctx) // Get proposers collection.
	if err != nil {
		return fmt.Errorf("failed to infer private collection name for the org: %v", err)
	}
	ApplyAgreeKey, err := ctx.GetStub().CreateCompositeKey(RequestInformationObjectType, []string{paraDeleteInput.
		ID}) // Create composite key
	if err != nil {
		return fmt.Errorf("failed to create composite key: %v", err)
	}

	valAsbytes, err := ctx.GetStub().GetPrivateData(paraCollection, ApplyAgreeKey) //get the apply_agreement
	if err != nil {
		return fmt.Errorf("failed to read apply_agreement: %v", err)
	}
	if valAsbytes == nil {
		return fmt.Errorf("para's apply_agreement does not exist: %v", paraDeleteInput.ID)
	}

	log.Printf("Deleting ApplyAgreement: %v", paraDeleteInput.ID)
	err = ctx.GetStub().DelPrivateData(orgCollection, paraDeleteInput.ID) // Delete the para
	if err != nil {
		return err
	}

	// Delete apply agreement record
	err = ctx.GetStub().DelPrivateData(paraCollection, ApplyAgreeKey) // remove agreement from state
	if err != nil {
		return err
	}

	return nil

}

// getCollectionName is an internal helper function to get collection of submitting client identity.
func getCollectionName(ctx contractapi.TransactionContextInterface) (string, error) {

	// Get the MSP ID of submitting client identity
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return "", fmt.Errorf("failed to get verified MSPID: %v", err)
	}

	// Create the collection name
	orgCollection := clientMSPID + "PrivateCollection"

	return orgCollection, nil
}

// verifyClientOrgMatchesPeerOrg is an internal function used verify client org id and matches peer org id.
func verifyClientOrgMatchesPeerOrg(ctx contractapi.TransactionContextInterface) error {
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed getting the client's MSPID: %v", err)
	}
	peerMSPID, err := shim.GetMSPID()
	if err != nil {
		return fmt.Errorf("failed getting the peer's MSPID: %v", err)
	}

	if clientMSPID != peerMSPID {
		return fmt.Errorf("client from org %v is not authorized to read or write private data from an org %v peer", clientMSPID, peerMSPID)
	}

	return nil
}

func submittingClientIdentity(ctx contractapi.TransactionContextInterface) (string, error) {
	b64ID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return "", fmt.Errorf("Failed to read clientID: %v", err)
	}
	decodeID, err := base64.StdEncoding.DecodeString(b64ID)
	if err != nil {
		return "", fmt.Errorf("failed to base64 decode clientID: %v", err)
	}
	return string(decodeID), nil
}

// ReadRequestInformation gets the User's identity
func (s *SmartContract) ReadRequestInformation(ctx contractapi.TransactionContextInterface, paraID string) (*ApplersInfo, error) {
	log.Printf("ReadRequestInformation: collection %v, ID %v", paraCollection, paraID)
	// composite key for ApplersInfo of this para
	ApplyKey, err := ctx.GetStub().CreateCompositeKey(RequestInformationObjectType, []string{paraID})
	if err != nil {
		return nil, fmt.Errorf("failed to create composite key: %v", err)
	}

	UserIdentity, err := ctx.GetStub().GetPrivateData(paraCollection, ApplyKey) // Get the identity from collection
	if err != nil {
		return nil, fmt.Errorf("failed to read ApplersInfo: %v", err)
	}
	if UserIdentity == nil {
		log.Printf("ApplersInfo for %v does not exist", paraID)
		return nil, nil
	}
	agreement := &ApplersInfo{
		ID:     paraID,
		UserID: string(UserIdentity),
	}
	return agreement, nil
}

func (s *SmartContract) PrepareAbeTest(ctx contractapi.TransactionContextInterface, NumMagnitude int, AttributesNeeded int) []string {
	// var keysUser []*abe.FAMEAttribKeys

	aesKey := cryptopasta.NewEncryptionKey()

	rand.Seed(time.Now().Unix())
	fameArray := make([]*abe.FAMEAttribKeys, AttributesNeeded)

	ABE_KeyGen_start := time.Now()
	relay := abe.NewFAME()
	// Generate an ABE key pair
	mpk, msk, err := relay.GenerateMasterKeys()
	if err != nil {
		panic(err)
	}
	policy := policySetup(NumMagnitude)
	msp, _ := abe.BooleanToMSP(policy, false) // The MSP structure defining the policy
	mskJSON, _ := json.Marshal(msk)
	mpkJSON, _ := json.Marshal(mpk)
	mspJSON, _ := json.Marshal(msp)
	abeJSON, _ := json.Marshal(relay)
	clientID, err := submittingClientIdentity(ctx)
	if err != nil {
		panic(err)
	}
	// end = time.Now()

	if err != nil {
		panic(err)
	}
	qualifiedAttributes := evaluateLogicCircuit(policy, AttributesNeeded)

	for _, subArr := range qualifiedAttributes {
		for i, str := range subArr {
			str = strings.ReplaceAll(str, "(", "")
			str = strings.ReplaceAll(str, ")", "")
			subArr[i] = str
		}
	}

	for i, qualifiedAttribute := range qualifiedAttributes {
		keys, _ := relay.GenerateAttribKeys(qualifiedAttribute, msk)
		fameArray[i] = keys
		keysJSON, err := json.Marshal(keys)
		if err != nil {
			panic(err)
		}
		mid := fmt.Sprintf("MemberKey%d", i)

		UerAbePara := AbeUserPara{
			ID:      mid,
			Owner:   clientID,
			ABEFAME: string(abeJSON),
			Mpk:     string(mpkJSON),
			Keys:    string(keysJSON),
		}

		UerAbeParaJSON, err := json.Marshal(UerAbePara)
		if err != nil {
			panic(err)
		}
		err = ctx.GetStub().PutPrivateData("Org1MSPPrivateCollection", mid, UerAbeParaJSON)
		if err != nil {
			panic(err)
		}
		// log.Printf("%v %v", keys, i)
	}
	ABE_KeyGen_Time := time.Since(ABE_KeyGen_start)

	err = verifyClientOrgMatchesPeerOrg(ctx)
	if err != nil {
		panic(err)
	}

	peerAbeParaPut := AbePeerPara{
		ID:      "paraAbeTest",
		Owner:   clientID,
		ABEFAME: string(abeJSON),
		Mpk:     string(mpkJSON),
		Msk:     string(mskJSON),
		Msp:     string(mspJSON),
	}
	orgCollection, err := getCollectionName(ctx)
	if err != nil {
		panic(err)
	}

	peerAbeParaPutJSON, err := json.Marshal(peerAbeParaPut)
	if err != nil {
		panic(err)
	}
	err = ctx.GetStub().PutPrivateData(orgCollection, peerAbeParaPut.ID, peerAbeParaPutJSON)
	if err != nil {
		panic(err)
	}

	// qualifiedAttributes := evaluateLogicCircuit(policy, abeInput.AttributesNeeded)

	ABE_Encrypt_start := time.Now()
	CT, _ := relay.Encrypt(string(aesKey[:]), msp, mpk) // 使用abe对aeskey 加密    //test2:abe加密测试
	ABE_Encrypt_Time := time.Since(ABE_Encrypt_start)

	ABE_Decrypt_start := time.Now()
	_, err = relay.Decrypt(CT, fameArray[1], mpk)
	ABE_Decrypt_Time := time.Since(ABE_Decrypt_start)
	if err != nil {
		panic(err)
	}

	log.Printf("=====================PrepareABe: ABE_KeyGen, ABE_Encrypt====================== qualifiedAttributes %v", qualifiedAttributes)

	return []string{ABE_KeyGen_Time.String(), ABE_Encrypt_Time.String(), ABE_Decrypt_Time.String()}
}

func (s *SmartContract) AbeEnTest(ctx contractapi.TransactionContextInterface) ([]string, error) {
	peerAbeParaJson, err := ctx.GetStub().GetPrivateData("Org1MSPPrivateCollection", "paraAbeTest") // Get the para from chaincode state
	if err != nil {
		return nil, err
	}
	if peerAbeParaJson == nil {
		return nil, fmt.Errorf("paraAbeTest does not exist")
	}

	aesKey := cryptopasta.NewEncryptionKey()

	var peerAbePara *AbePeerPara
	err = json.Unmarshal(peerAbeParaJson, &peerAbePara)
	if err != nil {
		return nil, fmt.Errorf("err = json.Unmarshal(peerAbeParaJson, &peerAbePara) failed %v", err)
	}

	ABEFAME := peerAbePara.ABEFAME
	MpkStr := peerAbePara.Mpk
	MspStr := peerAbePara.Msp
	MskStr := peerAbePara.Msk

	var abeFame *abe.FAME
	var Mpk *abe.FAMEPubKey
	var Msp *abe.MSP
	var Msk *abe.FAMESecKey

	err = json.Unmarshal([]byte(ABEFAME), &abeFame)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(MpkStr), &Mpk)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(MspStr), &Msp)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(MskStr), &Msk)

	if err != nil {
		return nil, err
	}

	ABE_Encrypt_start := time.Now()
	CT, _ := abeFame.Encrypt(string(aesKey[:]), Msp, Mpk) // 使用abe对aeskey 加密    //test2:abe加密测试
	ABE_Encrypt_Time := time.Since(ABE_Encrypt_start)

	userKeys, err := s.GetUserAbePara(ctx, "Org1MSPPrivateCollection", "MemberKey1")

	if err != nil {
		return nil, fmt.Errorf("userKeys, err := s.GetUserAbePara failed %v", err)
	}

	memKeysStr := userKeys.Keys

	var memKeys *abe.FAMEAttribKeys

	err = json.Unmarshal([]byte(ABEFAME), &abeFame)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(MpkStr), &Mpk)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(memKeysStr), &memKeys)
	if err != nil {
		return nil, err
	}

	// log.Printf("--------aeskey--------aeskey--------aeskey--------memKeys: %+v ", memKeys)
	// log.Printf("--------aeskey--------aeskey--------aeskey--------Mpk: %+v ", Mpk)
	// log.Printf("--------aeskey--------aeskey--------aeskey--------abeFame: %+v ", abeFame)

	ABE_Decrypt_start := time.Now()
	_, err = abeFame.Decrypt(CT, memKeys, Mpk)
	ABE_Decrypt_Time := time.Since(ABE_Decrypt_start)

	if err != nil {
		return nil, fmt.Errorf("abeFame.Decrypt failed %v", err)
	}
	return []string{ABE_Encrypt_Time.String(), ABE_Decrypt_Time.String()}, nil
}

func (s *SmartContract) PrepareFileTest(ctx contractapi.TransactionContextInterface, fileUrl string, Multiples int) error {
	clearFolder("/tmp/")
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
	// Download file
	// fileUrl := "http://kdd.ics.uci.edu/databases/kddcup99/kddcup.newtestdata_10_percent_unlabeled.gz"
	for j := 0; j < Multiples; j++ {

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
			chunkFileName := fmt.Sprintf("/tmp/kddcup.%d", i+1+10*j)
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
	}

	return nil
}
