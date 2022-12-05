/*******************************************************************************
 *
 *
 * (c) Copyright Merative US L.P. and others 2020-2022 
 *
 * SPDX-Licence-Identifier: Apache 2.0
 *
 *******************************************************************************/

// Package datatype_interface provides an interface for datatype methods.
package datatype_interface

import (
	"common/bchcls/cached_stub"
	"common/bchcls/data_model"
)

// DatatypeInterface is an interface for Datatype functions.
type DatatypeInterface interface {

	// GetDatatypeStruct returns the Datatype object.
	GetDatatypeStruct() data_model.Datatype

	// GetDatatypeID returns the DatatypeID.
	GetDatatypeID() string

	// GetDescription returns the description of datatype.
	GetDescription() string

	// SetDescription sets the description of the datatype.
	SetDescription(description string) error

	// IsActive returns true of the datatype is in active state.
	IsActive() bool

	// Activate() changes state to active if it's not already in active state.
	// If the state is already active, return success.
	Activate() error

	// Deactivate() changes state to inactive if it's in active state.
	// If the state is already inactive, return success.
	// All child datatypes are also deactivated when you call Deactivate on a parent.
	Deactivate() error

	// PutDatatype saves the updated datatype to the ledger.
	// If status of the datatype is "inactive", then all child datatypes are also deactivated.
	PutDatatype(stub cached_stub.CachedStubInterface) error

	// GetChildDatatypes returns a list of child datatype IDs.
	GetChildDatatypes(stub cached_stub.CachedStubInterface) ([]string, error)

	// GetParentDatatypeID returns ID of the parent datatype.
	GetParentDatatypeID(stub cached_stub.CachedStubInterface) (string, error)

	// GetParentDatatypes returns a list of parent datatype IDs, excluding ROOT datatype.
	// The first element is the direct parent, and the second is the grandparent, etc.
	GetParentDatatypes(stub cached_stub.CachedStubInterface) ([]string, error)

	// IsParentOf returns true or false based on whether the child datatype is in the child datatype chain.
	IsParentOf(stub cached_stub.CachedStubInterface, childDatatypeID string) (bool, error)

	// IsChildOf returns true or false based on whether the parent datatype is in the parent datatype chain.
	IsChildOf(stub cached_stub.CachedStubInterface, parentDatatypeID string) (bool, error)

	// GetDatatypeKeyID returns the datatype symkey ID for a given owner
	GetDatatypeKeyID(ownerID string) string
}
