package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SmartContract provides functions for managing FIR records
type SmartContract struct {
	contractapi.Contract
}

// FIR describes a First Information Report
type FIR struct {
	Accused     string `json:"Accused"`
	CrimeType   string `json:"CrimeType"`
	Description string `json:"Description"`
	FiledBy     string `json:"FiledBy"`
	FIRID       string `json:"FIRID"`
	Status      string `json:"Status"`
	Timestamp   string `json:"Timestamp"`
}

// getMSPID returns the client's MSP ID
func getMSPID(ctx contractapi.TransactionContextInterface) (string, error) {
	return ctx.GetClientIdentity().GetMSPID()
}

// onlyPolice enforces access for Org1MSP (Police)
func onlyPolice(ctx contractapi.TransactionContextInterface) error {
	mspid, err := getMSPID(ctx)
	if err != nil {
		return fmt.Errorf("unable to get MSP ID: %v", err)
	}
	if mspid != "Org1MSP" {
		return fmt.Errorf("access denied: only Org1 (Police) can perform this operation")
	}
	return nil
}

// InitLedger adds a base set of FIRs to the ledger
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	firs := []FIR{
		{FIRID: "FIR1", FiledBy: "OfficerA", Accused: "John Doe", CrimeType: "Theft", Description: "Stolen bike", Status: "Open", Timestamp: "2024-01-01T10:00:00Z"},
		{FIRID: "FIR2", FiledBy: "OfficerB", Accused: "Jane Smith", CrimeType: "Assault", Description: "Physical altercation", Status: "Investigation", Timestamp: "2024-01-02T14:30:00Z"},
	}

	for _, fir := range firs {
		firJSON, err := json.Marshal(fir)
		if err != nil {
			return err
		}
		err = ctx.GetStub().PutState(fir.FIRID, firJSON)
		if err != nil {
			return fmt.Errorf("failed to put to world state: %v", err)
		}
	}
	return nil
}

// FileFIR creates a new FIR entry in the ledger
func (s *SmartContract) FileFIR(ctx contractapi.TransactionContextInterface, firID, filedBy, accused, crimeType, description, status, timestamp string) error {
	if err := onlyPolice(ctx); err != nil {
		return err
	}

	exists, err := s.FIRExists(ctx, firID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("the FIR %s already exists", firID)
	}

	fir := FIR{
		FIRID:       firID,
		FiledBy:     filedBy,
		Accused:     accused,
		CrimeType:   crimeType,
		Description: description,
		Status:      status,
		Timestamp:   timestamp,
	}
	firJSON, err := json.Marshal(fir)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(firID, firJSON)
}

// ReadFIR retrieves a FIR record by ID
func (s *SmartContract) ReadFIR(ctx contractapi.TransactionContextInterface, firID string) (*FIR, error) {
	firJSON, err := ctx.GetStub().GetState(firID)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if firJSON == nil {
		return nil, fmt.Errorf("the FIR %s does not exist", firID)
	}

	var fir FIR
	err = json.Unmarshal(firJSON, &fir)
	if err != nil {
		return nil, err
	}
	return &fir, nil
}

// UpdateFIR allows updates to an existing FIR record
func (s *SmartContract) UpdateFIR(ctx contractapi.TransactionContextInterface, firID, status string) error {
	if err := onlyPolice(ctx); err != nil {
		return err
	}

	fir, err := s.ReadFIR(ctx, firID)
	if err != nil {
		return err
	}

	fir.Status = status
	updatedJSON, err := json.Marshal(fir)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(firID, updatedJSON)
}

// DeleteFIR removes a FIR record from the ledger
func (s *SmartContract) DeleteFIR(ctx contractapi.TransactionContextInterface, firID string) error {
	if err := onlyPolice(ctx); err != nil {
		return err
	}

	exists, err := s.FIRExists(ctx, firID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the FIR %s does not exist", firID)
	}
	return ctx.GetStub().DelState(firID)
}

// FIRExists checks if a FIR exists in world state
func (s *SmartContract) FIRExists(ctx contractapi.TransactionContextInterface, firID string) (bool, error) {
	firJSON, err := ctx.GetStub().GetState(firID)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}
	return firJSON != nil, nil
}

// GetAllFIRs returns all FIRs from the ledger
func (s *SmartContract) GetAllFIRs(ctx contractapi.TransactionContextInterface) ([]*FIR, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var firs []*FIR
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var fir FIR
		err = json.Unmarshal(queryResponse.Value, &fir)
		if err != nil {
			return nil, err
		}
		firs = append(firs, &fir)
	}
	return firs, nil
}
