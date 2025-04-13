package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SmartContract provides functions for managing an Asset
type SmartContract struct {
	contractapi.Contract
}

type PolicePersonnel struct {
	OfficerID        string `json:"officerId"`
	Name             string `json:"name"`
	Rank             string `json:"rank"`
	DOB              string `json:"dob"`
	Posting          string `json:"posting"`
	BadgeNumber      string `json:"badgeNumber"`
	EmploymentStatus string `json:"employmentStatus"`
	DateOfJoining    string `json:"dateOfJoining"`
	Award            string `json:"award"`       // Changed from []string to string
	Suspension       string `json:"suspension"`  // Changed from []string to string
	LastUpdatedBy    string `json:"lastUpdatedBy"`
	LastUpdatedOn    string `json:"lastUpdatedOn"`
}

// InitLedger adds a base set of assets to the ledger
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	if err := onlyPolice(ctx); err != nil {
		return err
	}

	personnel := []PolicePersonnel{
		{
			OfficerID:        "POL12345",
			Name:             "Inspector Anjali Mehta",
			Rank:             "Inspector",
			DOB:              "1985-08-15",
			Posting:          "Cyber Crime Unit, Mumbai",
			BadgeNumber:      "MUM-4521",
			EmploymentStatus: "Active",
			DateOfJoining:    "2010-06-12",
			Award:            "Gallantry Award 2018",
			Suspension:       "",
			LastUpdatedBy:    "Org1",
			LastUpdatedOn:    "2025-04-06",
		},
	}

	for _, p := range personnel {
		pJSON, err := json.Marshal(p)
		if err != nil {
			return err
		}
		err = ctx.GetStub().PutState(p.OfficerID, pJSON)
		if err != nil {
			return fmt.Errorf("failed to put to world state: %v", err)
		}
	}

	return nil
}

// CreatePolicePersonnel issues a new record to the ledger
func (s *SmartContract) CreatePolicePersonnel(
	ctx contractapi.TransactionContextInterface,
	officerID, name, rank, dob, posting, badgeNumber,
	employmentStatus, dateOfJoining, award, suspension, lastUpdatedBy, lastUpdatedOn string,
) error {
    if err := onlyPolice(ctx); err != nil {
		return err
	}
	exists, err := s.PersonnelExists(ctx, officerID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("the officer %s already exists", officerID)
	}

	personnel := PolicePersonnel{
		OfficerID:        officerID,
		Name:             name,
		Rank:             rank,
		DOB:              dob,
		Posting:          posting,
		BadgeNumber:      badgeNumber,
		EmploymentStatus: employmentStatus,
		DateOfJoining:    dateOfJoining,
		Award:            award,
		Suspension:       suspension,
		LastUpdatedBy:    lastUpdatedBy,
		LastUpdatedOn:    lastUpdatedOn,
	}

	pJSON, err := json.Marshal(personnel)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(officerID, pJSON)
}

// ReadPolicePersonnel returns a record by ID
func (s *SmartContract) ReadPolicePersonnel(ctx contractapi.TransactionContextInterface, officerID string) (*PolicePersonnel, error) {
	if err := onlyPolice(ctx); err != nil {
		return nil, err
	}

	pJSON, err := ctx.GetStub().GetState(officerID)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if pJSON == nil {
		return nil, fmt.Errorf("the officer %s does not exist", officerID)
	}

	var personnel PolicePersonnel
	err = json.Unmarshal(pJSON, &personnel)
	if err != nil {
		return nil, err
	}

	return &personnel, nil
}

// UpdatePolicePersonnel updates a record
func (s *SmartContract) UpdatePolicePersonnel(
	ctx contractapi.TransactionContextInterface,
	officerID, name, rank, dob, posting, badgeNumber,
	employmentStatus, dateOfJoining, award, suspension, lastUpdatedBy, lastUpdatedOn string,
) error {
	if err := onlyPolice(ctx); err != nil {
		return err
	}

	exists, err := s.PersonnelExists(ctx, officerID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the officer %s does not exist", officerID)
	}

	personnel := PolicePersonnel{
		OfficerID:        officerID,
		Name:             name,
		Rank:             rank,
		DOB:              dob,
		Posting:          posting,
		BadgeNumber:      badgeNumber,
		EmploymentStatus: employmentStatus,
		DateOfJoining:    dateOfJoining,
		Award:            award,
		Suspension:       suspension,
		LastUpdatedBy:    lastUpdatedBy,
		LastUpdatedOn:    lastUpdatedOn,
	}

	pJSON, err := json.Marshal(personnel)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(officerID, pJSON)
}

// DeletePolicePersonnel deletes a record
func (s *SmartContract) DeletePolicePersonnel(ctx contractapi.TransactionContextInterface, officerID string) error {
	if err := onlyPolice(ctx); err != nil {
		return err
	}

	exists, err := s.PersonnelExists(ctx, officerID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the officer %s does not exist", officerID)
	}

	return ctx.GetStub().DelState(officerID)
}

// PersonnelExists checks if the record exists
func (s *SmartContract) PersonnelExists(ctx contractapi.TransactionContextInterface, officerID string) (bool, error) {
	pJSON, err := ctx.GetStub().GetState(officerID)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}
	return pJSON != nil, nil
}

// GetAllPersonnel returns all records
func (s *SmartContract) GetAllPersonnel(ctx contractapi.TransactionContextInterface) ([]*PolicePersonnel, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var personnelList []*PolicePersonnel
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var personnel PolicePersonnel
		err = json.Unmarshal(queryResponse.Value, &personnel)
		if err != nil {
			return nil, err
		}
		personnelList = append(personnelList, &personnel)
	}

	return personnelList, nil
}

// Utility function: GetMSPID
func getMSPID(ctx contractapi.TransactionContextInterface) (string, error) {
	return ctx.GetClientIdentity().GetMSPID()
}

// Authorization check for police-only access
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


