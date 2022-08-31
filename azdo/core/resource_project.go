package core

import (
	"azdo/azdo/client"
	"azdo/utils/converter"
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v6/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v6/operations"
	"log"
	"time"
)

func ResourceProject() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceProjectCreate,
		ReadContext:   resourceProjectRead,
		DeleteContext: resourceProjectDelete,
		UpdateContext: resourceProjectUpdate,
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"visibility": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"version_control": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"work_item_template": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"process_template_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceProjectCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clients := m.(*client.AggregatedClient)
	project, err := expandProject(clients, d, true)
	if err != nil {
		return diag.FromErr(err)
	}
	operationRef, err := clients.CoreClient.QueueCreateProject(ctx, core.QueueCreateProjectArgs{ProjectToCreate: project})

	stateConf := &resource.StateChangeConf{
		ContinuousTargetOccurence: 1,
		Delay:                     5 * time.Second,
		MinTimeout:                10 * time.Second,
		Pending: []string{
			string(operations.OperationStatusValues.InProgress),
			string(operations.OperationStatusValues.Queued),
			string(operations.OperationStatusValues.NotSet),
		},
		Target: []string{
			string(operations.OperationStatusValues.Failed),
			string(operations.OperationStatusValues.Succeeded),
			string(operations.OperationStatusValues.Cancelled)},
		Refresh: projectStatusRefreshFunc(clients, operationRef),
		Timeout: d.Timeout(schema.TimeoutCreate),
	}

	if _, err := stateConf.WaitForStateContext(clients.Ctx); err != nil {
		return diag.Errorf(" waiting for project ready. %v ", err)
	}
	if err != nil {
		return diag.FromErr(fmt.Errorf(" creating project: %v", err))
	}
	d.Set("name", *project.Name)
	return resourceProjectRead(ctx, d, m)
}

func resourceProjectRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clients := m.(*client.AggregatedClient)
	id := d.Id()
	name := d.Get("name").(string)
	identifier := id
	if identifier == "" {
		identifier = name
	}

	project, err := clients.CoreClient.GetProject(clients.Ctx, core.GetProjectArgs{
		ProjectId:           &identifier,
		IncludeCapabilities: converter.Bool(true),
		IncludeHistory:      converter.Bool(false),
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf(" reading project: %v", err))
	}
	processTemplateID := (*project.Capabilities)["processTemplate"]["templateTypeId"]
	processTemplateName, err := lookupProcessTemplateName(clients, processTemplateID)

	d.SetId(project.Id.String())
	d.Set("name", project.Name)
	d.Set("visibility", project.Visibility)
	d.Set("description", project.Description)
	d.Set("process_template_id", processTemplateID)
	d.Set("work_item_template", processTemplateName)
	d.Set("version_control", (*project.Capabilities)["versioncontrol"]["sourceControlType"])
	return nil
}

func String(value string) *string {
	return &value
}

func resourceProjectDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clients := m.(*client.AggregatedClient)
	id := d.Id()
	uuid, err := uuid.Parse(id)
	if err != nil {
		return diag.FromErr(fmt.Errorf(" reading project: %s", id))
	}

	_, deleteErr := clients.CoreClient.QueueDeleteProject(clients.Ctx, core.QueueDeleteProjectArgs{
		ProjectId: &uuid,
	})
	if deleteErr != nil {
		return diag.FromErr(fmt.Errorf(" deleting project: %s", err))
	}
	return nil
}

func resourceProjectUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var operationRef *operations.OperationReference
	clients := m.(*client.AggregatedClient)
	project, err := expandProject(clients, d, false)
	if err != nil {
		return diag.FromErr(errors.New("failed to get project data"))
	}
	requiresUpdate := false

	if !d.HasChange("name") {
		project.Name = nil
	} else {
		requiresUpdate = true
	}
	if !d.HasChange("description") {
		project.Description = nil
	} else {
		requiresUpdate = true
	}
	if !d.HasChange("visibility") {
		project.Visibility = nil
	} else {
		requiresUpdate = true
	}
	if requiresUpdate {
		operationRef, err = clients.CoreClient.UpdateProject(ctx, core.UpdateProjectArgs{ProjectUpdate: project, ProjectId: project.Id})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	stateConf := &resource.StateChangeConf{
		ContinuousTargetOccurence: 1,
		Delay:                     10 * time.Second,
		MinTimeout:                10 * time.Second,
		Pending: []string{
			string(operations.OperationStatusValues.InProgress),
			string(operations.OperationStatusValues.Queued),
			string(operations.OperationStatusValues.NotSet),
		},
		Target: []string{
			string(operations.OperationStatusValues.Failed),
			string(operations.OperationStatusValues.Succeeded),
			string(operations.OperationStatusValues.Cancelled)},
		Refresh: projectStatusRefreshFunc(clients, operationRef),
		Timeout: d.Timeout(schema.TimeoutUpdate),
	}

	if _, err := stateConf.WaitForStateContext(clients.Ctx); err != nil {
		return diag.Errorf(" waiting for project ready. %v ", err)
	}

	return nil
}

func projectStatusRefreshFunc(clients *client.AggregatedClient, operationRef *operations.OperationReference) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		ret, err := clients.OperationsClient.GetOperation(clients.Ctx, operations.GetOperationArgs{
			OperationId: operationRef.Id,
			PluginId:    operationRef.PluginId,
		})
		if err != nil {
			return nil, string(operations.OperationStatusValues.Failed), err
		}

		if *ret.Status != operations.OperationStatusValues.Succeeded {
			log.Printf("[DEBUG] Waiting for project operation success. Operation result %v", ret.DetailedMessage)
		}

		return ret, string(*ret.Status), nil
	}
}

func expandProject(clients *client.AggregatedClient, d *schema.ResourceData, forCreate bool) (*core.TeamProject, error) {
	workItemTemplate := d.Get("work_item_template").(string)
	processTemplateID, err := lookupWorkItemTemplate(clients, workItemTemplate)
	if err != nil {
		return nil, err
	}

	// an "error" is OK here as it is expected in the case that the ID is not set in the resource data
	var projectID *uuid.UUID
	parsedID, err := uuid.Parse(d.Id())
	if err == nil {
		projectID = &parsedID
	}

	visibility := core.ProjectVisibility(d.Get("visibility").(string))

	var capabilities *map[string]map[string]string
	if forCreate {
		capabilities = &map[string]map[string]string{
			"versioncontrol": {
				"sourceControlType": d.Get("version_control").(string),
			},
			"processTemplate": {
				"templateTypeId": processTemplateID,
			},
		}
	}

	project := &core.TeamProject{
		Id:           projectID,
		Name:         converter.String(d.Get("name").(string)),
		Description:  converter.String(d.Get("description").(string)),
		Visibility:   &visibility,
		Capabilities: capabilities,
	}

	return project, nil
}

// Removed Code Segment
//var projectID *uuid.UUID
//visibility := core.ProjectVisibility(d.Get("visibility").(string))
//workItemTemplate := d.Get("work_item_template").(string)
//processTemplateID, err := lookupWorkItemTemplate(clients, workItemTemplate)
//parsedID, err := uuid.Parse(d.Id())
//if err == nil {
//	projectID = &parsedID
//}
//capabilities := &map[string]map[string]string{
//	"versioncontrol": {
//		"sourceControlType": d.Get("version_control").(string),
//	},
//	"processTemplate": {
//		"templateTypeId": processTemplateID,
//	},
//}
//project := &core.TeamProject{
//	Id:           projectID,
//	Name:         String(d.Get("name").(string)),
//	Description:  String(d.Get("description").(string)),
//	Visibility:   &visibility,
//	Capabilities: capabilities,
//}
//
