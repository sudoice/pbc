/*
Copyright 2021 IBM All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"bytes"
	// "context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

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

func main() {
	clientConnection := newGrpcConnection()
	defer clientConnection.Close()

	id := newIdentity()
	sign := newSign()

	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithHash(hash.SHA256),
		client.WithClientConnection(clientConnection),
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer gw.Close()

	chaincodeName := "fir"
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
	getAllFIRs(contract)
	createFIR(contract)
	readFIRByID(contract)
	updateFIR(contract)
	getAllFIRs(contract)
	exampleErrorHandling(contract)
}

func newGrpcConnection() *grpc.ClientConn {
	certificatePEM, err := os.ReadFile(tlsCertPath)
	if err != nil {
		panic(fmt.Errorf("failed to read TLS certificate file: %w", err))
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

func initLedger(contract *client.Contract) {
	fmt.Println("\n--> Submit Transaction: InitLedger")
	_, err := contract.SubmitTransaction("InitLedger")
	if err != nil {
		panic(fmt.Errorf("failed to submit InitLedger: %w", err))
	}
	fmt.Println("*** InitLedger committed successfully")
}

func getAllFIRs(contract *client.Contract) {
	fmt.Println("\n--> Evaluate Transaction: GetAllFIRs")
	result, err := contract.EvaluateTransaction("GetAllFIRs")
	if err != nil {
		panic(fmt.Errorf("failed to evaluate GetAllFIRs: %w", err))
	}
	fmt.Printf("*** Result: %s\n", formatJSON(result))
}

func createFIR(contract *client.Contract) {
	fmt.Printf("\n--> Submit Transaction: CreateFIR, creates a new FIR record\n")

	_, err := contract.SubmitTransaction("FileFIR",
		"FIR3",
		"OfficerC",
		"Robbery",
		"Bank robbery at downtown",
		"Alex Murphy",
		"Open",
		"2024-01-03T12:00:00Z",
	)

	if err != nil {
		fmt.Println("*** CreateFIR failed with error:", err)

		statusErr := status.Convert(err)
		details := statusErr.Details()
		if len(details) > 0 {
			fmt.Println("Error Details:")
			for _, detail := range details {
				switch d := detail.(type) {
				case *gateway.ErrorDetail:
					fmt.Printf("- address: %s; mspId: %s; message: %s\n", d.Address, d.MspId, d.Message)
				}
			}
		}
		panic(fmt.Errorf("failed to create FIR: %w", err))
	}

	fmt.Printf("*** FIR created successfully\n")
}

func readFIRByID(contract *client.Contract) {
	fmt.Println("\n--> Evaluate Transaction: ReadFIR")
	result, err := contract.EvaluateTransaction("ReadFIR", "FIR1")
	if err != nil {
		panic(fmt.Errorf("failed to read FIR: %w", err))
	}
	fmt.Printf("*** Result: %s\n", formatJSON(result))
}

func updateFIR(contract *client.Contract) {
	fmt.Println("\n--> Submit Transaction: UpdateFIR")
	_, err := contract.SubmitTransaction("UpdateFIR", "FIR1", "Closed")
	if err != nil {
		panic(fmt.Errorf("failed to update FIR: %w", err))
	}
	fmt.Println("*** FIR updated successfully")
}

func exampleErrorHandling(contract *client.Contract) {
	fmt.Println("\n--> Submit Transaction: UpdateFIR with wrong ID")
	_, err := contract.SubmitTransaction("UpdateFIR", "NON_EXISTENT_FIR", "Closed")
	if err == nil {
		panic("******** FAILED to return an error")
	}
	fmt.Println("*** Successfully caught the error:")

	var endorseErr *client.EndorseError
	var submitErr *client.SubmitError
	var commitStatusErr *client.CommitStatusError
	var commitErr *client.CommitError

	if errors.As(err, &endorseErr) {
		fmt.Printf("Endorse error for transaction %s: %v\n", endorseErr.TransactionID, endorseErr)
	} else if errors.As(err, &submitErr) {
		fmt.Printf("Submit error for transaction %s: %v\n", submitErr.TransactionID, submitErr)
	} else if errors.As(err, &commitStatusErr) {
		fmt.Printf("Commit status error for transaction %s: %v\n", commitStatusErr.TransactionID, commitStatusErr)
	} else if errors.As(err, &commitErr) {
		fmt.Printf("Transaction %s failed with status %d: %v\n", commitErr.TransactionID, commitErr.Code, commitErr)
	} else {
		fmt.Printf("Unexpected error type: %v\n", err)
	}

	statusErr := status.Convert(err)
	for _, detail := range statusErr.Details() {
		if d, ok := detail.(*gateway.ErrorDetail); ok {
			fmt.Printf("- address: %s; mspId: %s; message: %s\n", d.Address, d.MspId, d.Message)
		}
	}
}

func formatJSON(data []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		panic(fmt.Errorf("failed to format JSON: %w", err))
	}
	return prettyJSON.String()
}