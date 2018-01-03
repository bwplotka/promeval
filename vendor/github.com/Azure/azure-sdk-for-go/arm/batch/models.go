package batch

// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Code generated by Microsoft (R) AutoRest Code Generator 0.17.0.0
// Changes may cause incorrect behavior and will be lost if the code is
// regenerated.

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/Azure/go-autorest/autorest/to"
	"net/http"
)

// AccountKeyType enumerates the values for account key type.
type AccountKeyType string

const (
	// Primary specifies the primary state for account key type.
	Primary AccountKeyType = "Primary"
	// Secondary specifies the secondary state for account key type.
	Secondary AccountKeyType = "Secondary"
)

// PackageState enumerates the values for package state.
type PackageState string

const (
	// Active specifies the active state for package state.
	Active PackageState = "active"
	// Pending specifies the pending state for package state.
	Pending PackageState = "pending"
	// Unmapped specifies the unmapped state for package state.
	Unmapped PackageState = "unmapped"
)

// ProvisioningState enumerates the values for provisioning state.
type ProvisioningState string

const (
	// Cancelled specifies the cancelled state for provisioning state.
	Cancelled ProvisioningState = "Cancelled"
	// Creating specifies the creating state for provisioning state.
	Creating ProvisioningState = "Creating"
	// Deleting specifies the deleting state for provisioning state.
	Deleting ProvisioningState = "Deleting"
	// Failed specifies the failed state for provisioning state.
	Failed ProvisioningState = "Failed"
	// Invalid specifies the invalid state for provisioning state.
	Invalid ProvisioningState = "Invalid"
	// Succeeded specifies the succeeded state for provisioning state.
	Succeeded ProvisioningState = "Succeeded"
)

// Account is contains information about an Azure Batch account.
type Account struct {
	autorest.Response `json:"-"`
	ID                *string             `json:"id,omitempty"`
	Name              *string             `json:"name,omitempty"`
	Type              *string             `json:"type,omitempty"`
	Location          *string             `json:"location,omitempty"`
	Tags              *map[string]*string `json:"tags,omitempty"`
	Properties        *AccountProperties  `json:"properties,omitempty"`
}

// AccountBaseProperties is the properties of a Batch account.
type AccountBaseProperties struct {
	AutoStorage *AutoStorageBaseProperties `json:"autoStorage,omitempty"`
}

// AccountCreateParameters is parameters supplied to the Create operation.
type AccountCreateParameters struct {
	Location   *string                `json:"location,omitempty"`
	Tags       *map[string]*string    `json:"tags,omitempty"`
	Properties *AccountBaseProperties `json:"properties,omitempty"`
}

// AccountKeys is a set of Azure Batch account keys.
type AccountKeys struct {
	autorest.Response `json:"-"`
	Primary           *string `json:"primary,omitempty"`
	Secondary         *string `json:"secondary,omitempty"`
}

// AccountListResult is values returned by the List operation.
type AccountListResult struct {
	autorest.Response `json:"-"`
	Value             *[]Account `json:"value,omitempty"`
	NextLink          *string    `json:"nextLink,omitempty"`
}

// AccountListResultPreparer prepares a request to retrieve the next set of results. It returns
// nil if no more results exist.
func (client AccountListResult) AccountListResultPreparer() (*http.Request, error) {
	if client.NextLink == nil || len(to.String(client.NextLink)) <= 0 {
		return nil, nil
	}
	return autorest.Prepare(&http.Request{},
		autorest.AsJSON(),
		autorest.AsGet(),
		autorest.WithBaseURL(to.String(client.NextLink)))
}

// AccountProperties is account specific properties.
type AccountProperties struct {
	AccountEndpoint              *string                `json:"accountEndpoint,omitempty"`
	ProvisioningState            ProvisioningState      `json:"provisioningState,omitempty"`
	AutoStorage                  *AutoStorageProperties `json:"autoStorage,omitempty"`
	CoreQuota                    *int32                 `json:"coreQuota,omitempty"`
	PoolQuota                    *int32                 `json:"poolQuota,omitempty"`
	ActiveJobAndJobScheduleQuota *int32                 `json:"activeJobAndJobScheduleQuota,omitempty"`
}

// AccountRegenerateKeyParameters is parameters supplied to the RegenerateKey
// operation.
type AccountRegenerateKeyParameters struct {
	KeyName AccountKeyType `json:"keyName,omitempty"`
}

// AccountUpdateParameters is parameters supplied to the Update operation.
type AccountUpdateParameters struct {
	Tags       *map[string]*string    `json:"tags,omitempty"`
	Properties *AccountBaseProperties `json:"properties,omitempty"`
}

// ActivateApplicationPackageParameters is parameters for an
// ApplicationOperations.ActivateApplicationPackage request.
type ActivateApplicationPackageParameters struct {
	Format *string `json:"format,omitempty"`
}

// AddApplicationParameters is parameters for an
// ApplicationOperations.AddApplication request.
type AddApplicationParameters struct {
	AllowUpdates *bool   `json:"allowUpdates,omitempty"`
	DisplayName  *string `json:"displayName,omitempty"`
}

// Application is contains information about an application in a Batch account.
type Application struct {
	autorest.Response `json:"-"`
	ID                *string               `json:"id,omitempty"`
	DisplayName       *string               `json:"displayName,omitempty"`
	Packages          *[]ApplicationPackage `json:"packages,omitempty"`
	AllowUpdates      *bool                 `json:"allowUpdates,omitempty"`
	DefaultVersion    *string               `json:"defaultVersion,omitempty"`
}

// ApplicationPackage is an application package which represents a particular
// version of an application.
type ApplicationPackage struct {
	autorest.Response  `json:"-"`
	ID                 *string      `json:"id,omitempty"`
	Version            *string      `json:"version,omitempty"`
	State              PackageState `json:"state,omitempty"`
	Format             *string      `json:"format,omitempty"`
	StorageURL         *string      `json:"storageUrl,omitempty"`
	StorageURLExpiry   *date.Time   `json:"storageUrlExpiry,omitempty"`
	LastActivationTime *date.Time   `json:"lastActivationTime,omitempty"`
}

// AutoStorageBaseProperties is the properties related to auto storage account.
type AutoStorageBaseProperties struct {
	StorageAccountID *string `json:"storageAccountId,omitempty"`
}

// AutoStorageProperties is contains information about the auto storage
// account associated with a Batch account.
type AutoStorageProperties struct {
	StorageAccountID *string    `json:"storageAccountId,omitempty"`
	LastKeySync      *date.Time `json:"lastKeySync,omitempty"`
}

// ListApplicationsResult is response to an
// ApplicationOperations.ListApplications request.
type ListApplicationsResult struct {
	autorest.Response `json:"-"`
	Value             *[]Application `json:"value,omitempty"`
	NextLink          *string        `json:"nextLink,omitempty"`
}

// ListApplicationsResultPreparer prepares a request to retrieve the next set of results. It returns
// nil if no more results exist.
func (client ListApplicationsResult) ListApplicationsResultPreparer() (*http.Request, error) {
	if client.NextLink == nil || len(to.String(client.NextLink)) <= 0 {
		return nil, nil
	}
	return autorest.Prepare(&http.Request{},
		autorest.AsJSON(),
		autorest.AsGet(),
		autorest.WithBaseURL(to.String(client.NextLink)))
}

// LocationQuota is quotas associated with a Batch region for a particular
// subscription.
type LocationQuota struct {
	autorest.Response `json:"-"`
	AccountQuota      *int32 `json:"accountQuota,omitempty"`
}

// Resource is a definition of an Azure resource.
type Resource struct {
	ID       *string             `json:"id,omitempty"`
	Name     *string             `json:"name,omitempty"`
	Type     *string             `json:"type,omitempty"`
	Location *string             `json:"location,omitempty"`
	Tags     *map[string]*string `json:"tags,omitempty"`
}

// UpdateApplicationParameters is parameters for an
// ApplicationOperations.UpdateApplication request.
type UpdateApplicationParameters struct {
	AllowUpdates   *bool   `json:"allowUpdates,omitempty"`
	DefaultVersion *string `json:"defaultVersion,omitempty"`
	DisplayName    *string `json:"displayName,omitempty"`
}
