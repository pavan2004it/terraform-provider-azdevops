package client

import (
	"context"
	"fmt"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v6"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v6/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v6/operations"
	"log"
	"strings"
)

type AggregatedClient struct {
	OrganizationURL  string
	CoreClient       core.Client
	OperationsClient operations.Client
	Ctx              context.Context
}

func GetAzdoClient(azdoPAT string, organizationURL string, tfVersion string) (*AggregatedClient, error) {
	ctx := context.Background()

	if strings.EqualFold(azdoPAT, "") {
		return nil, fmt.Errorf("the personal access token is required")
	}

	if strings.EqualFold(organizationURL, "") {
		return nil, fmt.Errorf("the url of the Azure DevOps is required")
	}

	connection := azuredevops.NewPatConnection(organizationURL, azdoPAT)

	// client for these APIs (includes CRUD for AzDO projects...):
	//	https://docs.microsoft.com/en-us/rest/api/azure/devops/core/?view=azure-devops-rest-5.1

	coreClient, err := core.NewClient(ctx, connection)
	if err != nil {
		log.Printf("getAzdoClient(): core.NewClient failed.")
		return nil, err
	}

	operationsClient := operations.NewClient(ctx, connection)

	aggregatedClient := &AggregatedClient{
		OrganizationURL:  organizationURL,
		CoreClient:       coreClient,
		OperationsClient: operationsClient,
		Ctx:              ctx,
	}

	log.Printf("getAzdoClient(): Created core, build, operations, and serviceendpoint clients successfully!")
	return aggregatedClient, nil
}
