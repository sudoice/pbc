/*
Copyright 2021 IBM All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"time"
    "log"
	
	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/hash"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"github.com/hyperledger/fabric-protos-go-apiv2/gateway"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

const (
	mspID        = "Org1MSP"
	cryptoPath   = "../../police-network/organizations/peerOrganizations/org1.example.com"
	certPath     = cryptoPath + "/users/User1@org1.example.com/msp/signcerts"
	keyPath      = cryptoPath + "/users/User1@org1.example.com/msp/keystore"
	tlsCertPath  = cryptoPath + "/peers/peer0.org1.example.com/tls/ca.crt"
	peerEndpoint = "dns:///localhost:7051"
	gatewayPeer  = "peer0.org1.example.com"
)

var now = time.Now()
var assetId = fmt.Sprintf("asset%d", now.Unix()*1e3+int64(now.Nanosecond())/1e6)

func main() {
	// The gRPC client connection should be shared by all Gateway connections to this endpoint
	clientConnection := newGrpcConnection()
	defer clientConnection.Close()

	id := newIdentity()
	sign := newSign()

	// Create a Gateway connection for a specific client identity
	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithHash(hash.SHA256),
		client.WithClientConnection(clientConnection),
		// Default timeouts for different gRPC calls
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer gw.Close()

	// Override default values for chaincode and channel name as they may differ in testing contexts.
	chaincodeName := "policeman"
	if ccname := os.Getenv("CHAINCODE_NAME"); ccname != "" {
		chaincodeName = ccname
	}

	channelName := "mychannel"
	if cname := os.Getenv("CHANNEL_NAME"); cname != "" {
		channelName = cname
	}

	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	initLedger(contract)
    createPolicePersonnel(contract)
    readPolicePersonnel(contract, "POL12345")
    updatePolicePersonnel(contract)
    deletePolicePersonnel(contract, "POL12345")
    exists, err := personnelExists(contract, "POL12345")
    if err != nil {
     panic(err)
    }
    fmt.Printf("Personnel exists: %t\n", exists)
    getAllPersonnel(contract)

}

// newGrpcConnection creates a gRPC connection to the Gateway server.
func newGrpcConnection() *grpc.ClientConn {
	certificatePEM, err := os.ReadFile(tlsCertPath)
	if err != nil {
		panic(fmt.Errorf("failed to read TLS certifcate file: %w", err))
	}

	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		panic(err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(certificate)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, gatewayPeer)

	connection, err := grpc.NewClient(peerEndpoint, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		panic(fmt.Errorf("failed to create gRPC connection: %w", err))
	}

	return connection
}

// newIdentity creates a client identity for this Gateway connection using an X.509 certificate.
func newIdentity() *identity.X509Identity {
	certificatePEM, err := readFirstFile(certPath)
	if err != nil {
		panic(fmt.Errorf("failed to read certificate file: %w", err))
	}

	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		panic(err)
	}

	id, err := identity.NewX509Identity(mspID, certificate)
	if err != nil {
		panic(err)
	}

	return id
}

// newSign creates a function that generates a digital signature from a message digest using a private key.
func newSign() identity.Sign {
	privateKeyPEM, err := readFirstFile(keyPath)
	if err != nil {
		panic(fmt.Errorf("failed to read private key file: %w", err))
	}

	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		panic(err)
	}

	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		panic(err)
	}

	return sign
}

func readFirstFile(dirPath string) ([]byte, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, err
	}

	fileNames, err := dir.Readdirnames(1)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(path.Join(dirPath, fileNames[0]))
}

// This type of transaction would typically only be run once by an application the first time it was started after its
// initial deployment. A new version of the chaincode deployed later would likely not need to run an "init" function.
func initLedger(contract *client.Contract) {
	fmt.Printf("\n--> Submit Transaction: InitLedger, initializes police personnel records in the ledger\n")

	_, err := contract.SubmitTransaction("InitLedger")
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

func getAllPersonnel(contract *client.Contract) {
	fmt.Println("\n--> Evaluate Transaction: GetAllPersonnel, returns all police personnel records")

	evaluateResult, err := contract.EvaluateTransaction("GetAllPersonnel")
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}

func createPolicePersonnel(contract *client.Contract) {
	fmt.Printf("\n--> Submit Transaction: CreatePolicePersonnel, creates a new police personnel record\n")

	officerID := "POL12346"
	name := "Sub-Inspector Raj Kumar"
	rank := "Sub-Inspector"
	dob := "1990-01-25"
	posting := "Crime Branch, Delhi"
	badgeNumber := "DEL-7890"
	employmentStatus := "Active"
	dateOfJoining := "2015-07-10"
	lastUpdatedBy := "Org1"
	lastUpdatedOn := time.Now().Format("2006-01-02")
	award := "Meritorious Service Medal"
	suspension := "" // or any suspension notes as plain string

	_, err := contract.SubmitTransaction(
		"CreatePolicePersonnel",
		officerID,
		name,
		rank,
		dob,
		posting,
		badgeNumber,
		employmentStatus,
		dateOfJoining,
		award,
		suspension,
		lastUpdatedBy,
		lastUpdatedOn,
	)
	if err != nil {
		log.Fatalf("Failed to Submit transaction: %v\n", err)
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

func readPolicePersonnel(contract *client.Contract, officerID string) {
	fmt.Printf("\n--> Evaluate Transaction: ReadPolicePersonnel, returns officer record by ID\n")

	evaluateResult, err := contract.EvaluateTransaction("ReadPolicePersonnel", officerID)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}

func updatePolicePersonnel(contract *client.Contract) {
	fmt.Printf("\n--> Submit Transaction: UpdatePolicePersonnel, updates an existing officer's record\n")

	// Updated sample data
	officerID := "POL12346"
	name := "Sub-Inspector Raj Kumar"
	rank := "Sub-Inspector"
	dob := "1990-01-25"
	posting := "Cyber Crime Cell, Delhi"
	badgeNumber := "DEL-7890"
	employmentStatus := "Active"
	dateOfJoining := "2015-07-10"
	lastUpdatedBy := "Org1"
	lastUpdatedOn := time.Now().Format("2006-01-02")
	award := "Meritorious Service Medal, Best Investigator 2024"
	suspension := "" // No suspensions

	_, err := contract.SubmitTransaction(
		"UpdatePolicePersonnel",
		officerID,
		name,
		rank,
		dob,
		posting,
		badgeNumber,
		employmentStatus,
		dateOfJoining,
		award,
		suspension,
		lastUpdatedBy,
		lastUpdatedOn,
	)
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}


func deletePolicePersonnel(contract *client.Contract, officerID string) {
	fmt.Printf("\n--> Submit Transaction: DeletePolicePersonnel, deletes officer record\n")

	_, err := contract.SubmitTransaction("DeletePolicePersonnel", officerID)
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Officer record deleted successfully\n")
}

func personnelExists(contract *client.Contract, personnelID string) (bool, error) {
    result, err := contract.EvaluateTransaction("PersonnelExists", personnelID)
    if err != nil {
        return false, err
    }
    return string(result) == "true", nil
}


// Submit transaction, passing in the wrong number of arguments ,expected to throw an error containing details of any error responses from the smart contract.
func exampleErrorHandling(contract *client.Contract) {
	fmt.Println("\n--> Submit Transaction: UpdateAsset asset70, asset70 does not exist and should return an error")

	_, err := contract.SubmitTransaction("UpdateAsset", "asset70", "blue", "5", "Tomoko", "300")
	if err == nil {
		panic("******** FAILED to return an error")
	}

	fmt.Println("*** Successfully caught the error:")

	var endorseErr *client.EndorseError
	var submitErr *client.SubmitError
	var commitStatusErr *client.CommitStatusError
	var commitErr *client.CommitError

	if errors.As(err, &endorseErr) {
		fmt.Printf("Endorse error for transaction %s with gRPC status %v: %s\n", endorseErr.TransactionID, status.Code(endorseErr), endorseErr)
	} else if errors.As(err, &submitErr) {
		fmt.Printf("Submit error for transaction %s with gRPC status %v: %s\n", submitErr.TransactionID, status.Code(submitErr), submitErr)
	} else if errors.As(err, &commitStatusErr) {
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Printf("Timeout waiting for transaction %s commit status: %s", commitStatusErr.TransactionID, commitStatusErr)
		} else {
			fmt.Printf("Error obtaining commit status for transaction %s with gRPC status %v: %s\n", commitStatusErr.TransactionID, status.Code(commitStatusErr), commitStatusErr)
		}
	} else if errors.As(err, &commitErr) {
		fmt.Printf("Transaction %s failed to commit with status %d: %s\n", commitErr.TransactionID, int32(commitErr.Code), err)
	} else {
		panic(fmt.Errorf("unexpected error type %T: %w", err, err))
	}

	// Any error that originates from a peer or orderer node external to the gateway will have its details
	// embedded within the gRPC status error. The following code shows how to extract that.
	statusErr := status.Convert(err)

	details := statusErr.Details()
	if len(details) > 0 {
		fmt.Println("Error Details:")

		for _, detail := range details {
			switch detail := detail.(type) {
			case *gateway.ErrorDetail:
				fmt.Printf("- address: %s; mspId: %s; message: %s\n", detail.Address, detail.MspId, detail.Message)
			}
		}
	}
}

// Format JSON data
func formatJSON(data []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		panic(fmt.Errorf("failed to parse JSON: %w", err))
	}
	return prettyJSON.String()
}