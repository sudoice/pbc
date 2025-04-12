/*
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"log"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

func main() {
	firChaincode, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		log.Panicf("Error creating police FIR chaincode: %v", err)
	}

	if err := firChaincode.Start(); err != nil {
		log.Panicf("Error starting police FIR chaincode: %v", err)
	}
}
