package chaincode

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fentec-project/gofe/abe"
	"github.com/gtank/cryptopasta"

	shell "github.com/ipfs/go-ipfs-api"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type FileDownload struct {
	aesKey string `json:"aesKey"`
	Cid    string `json:"cid"`
}

func (s *SmartContract) GetUploadKey(ctx contractapi.TransactionContextInterface, keyId string) (*MessageKey, error) {

	log.Printf("GetUploadKey: collection %v, ID %v", paraCollection, keyId)
	UploadKeyJson, err := ctx.GetStub().GetPrivateData(paraCollection, keyId) //get the para from chaincode state
	if err != nil {
		return nil, fmt.Errorf("failed to read para: %v", err)
	}

	// No Asset found, return empty response
	if UploadKeyJson == nil {
		return nil, fmt.Errorf("%v does not exist in collection %v", keyId, paraCollection)
	}

	var UploadKey *MessageKey
	err = json.Unmarshal(UploadKeyJson, &UploadKey)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	return UploadKey, nil
}

func (s *SmartContract) GetUserAbePara(ctx contractapi.TransactionContextInterface, collection string, id string) (*AbeUserPara, error) {
	userAbeParaJson, err := ctx.GetStub().GetPrivateData(collection, id) // Get the para from chaincode state

	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if userAbeParaJson == nil {
		return nil, fmt.Errorf("the para %s does not exist", id)
	}

	var userAbePara *AbeUserPara
	err = json.Unmarshal(userAbeParaJson, &userAbePara)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}
	return userAbePara, nil
}

func (s *SmartContract) GetFileFromIPFS(ctx contractapi.TransactionContextInterface, collection string, UserKey string, ip string, messageId string) (string, error) {
	sh := shell.NewShell(ip)
	clearFolder("/tmp/")
	log.Printf("--------clearFolder--------clearFolder--------clearFolder--------clearFolder ")

	Download_start := time.Now()

	messageFile, err := s.AesKeyDecrypt(ctx, collection, UserKey, messageId)
	if err != nil {
		return "", fmt.Errorf("error aesKeyDecrypt aeskey: %v", err)
	}
	// aeskey, err := aesKeyDecrypt(para.AesKey, *userKeys)

	aesKeyBytes := []byte(messageFile.aesKey)
	var aesKey [32]byte
	copy(aesKey[:], aesKeyBytes)
	log.Printf("--------aeskey--------aeskey--------aeskey--------abeFame: %+v ", aesKey)

	tempDir, err := ioutil.TempDir("/tmp", "ipfs_download")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save the files in the temporary directory
	fileInfos, err := sh.List(messageFile.Cid)
	if err != nil {
		return "", fmt.Errorf("failed to read directory contents: %v", err)
	}
	for _, fileInfo := range fileInfos {
		// Get the file from IPFS network
		fileResp, err := sh.Cat(fmt.Sprintf("%s/%s", messageFile.Cid, fileInfo.Name))
		if err != nil {
			return "", fmt.Errorf("failed to get file from IPFS network: %v", err)
		}
		defer fileResp.Close()

		// Save the file to the temporary directory
		fileContent, err := ioutil.ReadAll(fileResp)
		if err != nil {
			return "", fmt.Errorf("failed to read file content: %v", err)
		}
		err = ioutil.WriteFile(fmt.Sprintf("%s/%s", tempDir, fileInfo.Name), fileContent, 0644)
		if err != nil {
			return "", fmt.Errorf("failed to save file in temporary directory: %v", err)
		}
	}

	// Decrypt the files

	err = decryptFolder(tempDir, &aesKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt files: %v", err)
	}

	// Move the files to /tmp directory
	err = os.Rename(tempDir, "/tmp/"+messageFile.Cid)
	if err != nil {
		return "", fmt.Errorf("failed to move files to /tmp directory: %v", err)
	}
	Download_Time := time.Since(Download_start)

	return Download_Time.String(), nil
}

// ReadApplyValue reads the para private details in organization specific collection
func (s *SmartContract) ReadApplyValue(ctx contractapi.TransactionContextInterface, collection string, paraID string) (*ApplyValue, error) {
	log.Printf("ReadApplyValue: collection %v, ID %v", collection, paraID)
	paraDetailsJSON, err := ctx.GetStub().GetPrivateData(collection, paraID) // Get the para from chaincode state
	if err != nil {
		return nil, fmt.Errorf("failed to read para details: %v", err)
	}
	if paraDetailsJSON == nil {
		log.Printf("ApplyValue for %v does not exist in collection %v", paraID, collection)
		return nil, nil
	}

	var paraDetails *ApplyValue
	err = json.Unmarshal(paraDetailsJSON, &paraDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	return paraDetails, nil
}

func decryptFile(filename string, aesKey *[32]byte) error {
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

func (s *SmartContract) ApplyForMem(ctx contractapi.TransactionContextInterface) error {

	// Get ID of submitting client identity
	clientID, err := submittingClientIdentity(ctx)
	if err != nil {
		return err
	}

	// Value is private, therefore it gets passed in transient field
	transientMap, err := ctx.GetStub().GetTransient()
	if err != nil {
		return fmt.Errorf("error getting transient: %v", err)
	}

	// Persist the JSON bytes as-is so that there is no risk of nondeterministic marshaling.
	valueJSONasBytes, ok := transientMap["apply_value"]
	if !ok {
		return fmt.Errorf("apply_value key not found in the transient map")
	}

	// Unmarshal the tranisent map to get the para ID.
	var valueJSON ApplyValue
	err = json.Unmarshal(valueJSONasBytes, &valueJSON)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	// Do some error checking since we get the chance
	if len(valueJSON.ID) == 0 {
		return fmt.Errorf("paraID field must be a non-empty string")
	}
	if len(valueJSON.ApplyInfo) == 0 {
		return fmt.Errorf("appraisedValue field must be a positive integer")
	}
	// Verify that the client is submitting request to peer in their organization
	err = verifyClientOrgMatchesPeerOrg(ctx)
	if err != nil {
		return fmt.Errorf("ApplyForMember cannot be performed: Error %v", err)
	}

	// Get collection name for this organization. Needs to be read by a member of the organization.
	orgCollection, err := getCollectionName(ctx)
	if err != nil {
		return fmt.Errorf("failed to infer private collection name for the org: %v", err)
	}

	log.Printf("ApplyForMember Put: collection %v, ID %v", orgCollection, valueJSON.ID)
	// Put agreed value in the org specifc private data collection
	err = ctx.GetStub().PutPrivateData(orgCollection, valueJSON.ID, valueJSONasBytes)
	if err != nil {
		return fmt.Errorf("failed to put para bid: %v", err)
	}

	// Create agreeement that indicates which identity has agreed to purchase
	// In a more realistic apply scenario, a apply agreement would be secured to ensure that it cannot
	// be overwritten by another channel member
	//"RequestInformationMemberKey\int
	ApplyKey, err := ctx.GetStub().CreateCompositeKey(RequestInformationObjectType, []string{valueJSON.ID})
	if err != nil {
		return fmt.Errorf("failed to create composite key: %v", err)
	}

	log.Printf("ApplyForMember Put: collection %v, ID %v, Key %v", paraCollection, valueJSON.ID, ApplyKey)
	err = ctx.GetStub().PutPrivateData(paraCollection, ApplyKey, []byte(clientID))
	if err != nil {
		return fmt.Errorf("failed to put para bid: %v", err)
	}

	return nil
}

func decryptFolder(folderPath string, aesKey *[32]byte) error {
	startTime := time.Now()

	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			if err := decryptFile(path, aesKey); err != nil {
				return err
			}
		}

		return nil
	})

	elapsedTime := time.Since(startTime)

	log.Printf("Decryption complete. Time elapsed: %v\n", elapsedTime)

	return err
}

func (s *SmartContract) AesKeyDecrypt(ctx contractapi.TransactionContextInterface, collection string, UserKey string, messageId string) (*FileDownload, error) {
	userKeys, err := s.GetUserAbePara(ctx, collection, UserKey)

	if err != nil {
		return nil, fmt.Errorf("error GetUserAbePara userKeys: %v", err)
	}
	ABEFAME := userKeys.ABEFAME
	MpkStr := userKeys.Mpk
	memKeysStr := userKeys.Keys

	var abeFame *abe.FAME
	var Mpk *abe.FAMEPubKey
	var memKeys *abe.FAMEAttribKeys
	var aesKeyEncrypt *abe.FAMECipher

	err = json.Unmarshal([]byte(ABEFAME), &abeFame)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal([]byte(MpkStr), &Mpk)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal([]byte(memKeysStr), &memKeys)
	if err != nil {
		panic(err)
	}

	para, err := s.GetUploadKey(ctx, messageId)
	if err != nil {
		return nil, fmt.Errorf("error reading para: %v", err)
	}
	if para == nil {
		return nil, fmt.Errorf("%v does not exist", messageId)
	}
	aesKeyEncryptStr := para.AesKey
	err = json.Unmarshal([]byte(aesKeyEncryptStr), &aesKeyEncrypt)
	if err != nil {
		panic(err)
	}

	log.Printf("--------aeskey--------aeskey--------aeskey--------aesKeyEncrypt: %+v", aesKeyEncrypt)
	log.Printf("--------aeskey--------aeskey--------aeskey--------memKeys: %+v ", memKeys)
	log.Printf("--------aeskey--------aeskey--------aeskey--------Mpk: %+v ", Mpk)
	log.Printf("--------aeskey--------aeskey--------aeskey--------abeFame: %+v ", abeFame)

	aesKeyStr, err := abeFame.Decrypt(aesKeyEncrypt, memKeys, Mpk)
	if err != nil {
		panic(err)
	}

	FileDownloadMessage := FileDownload{
		aesKey: aesKeyStr,
		Cid:    para.Cid,
	}

	return &FileDownloadMessage, nil
}
