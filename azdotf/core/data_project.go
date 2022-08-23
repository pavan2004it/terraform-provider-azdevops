package core

import (
	"azdoTf/azdotf/client"
	"azdoTf/utils"
	"azdoTf/utils/converter"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v6/core"
	"log"
)

func DataProject() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataProjectRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"project_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"visibility": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"version_control": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"process_template_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"work_item_template": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataProjectRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clients := m.(*client.AggregatedClient)

	name := d.Get("name").(string)
	id := d.Get("project_id").(string)

	if name == "" && id == "" {
		return diag.FromErr(fmt.Errorf("Either project_id or name must be set "))
	}

	identifier := id
	if identifier == "" {
		identifier = name
	}

	project, err := clients.CoreClient.GetProject(ctx, core.GetProjectArgs{
		ProjectId:           &identifier,
		IncludeCapabilities: converter.Bool(true),
		IncludeHistory:      converter.Bool(false),
	})

	if err != nil {
		if utils.ResponseWasNotFound(err) {
			return diag.FromErr(fmt.Errorf("Project with name %s or ID %s does not exist ", name, id))
		}
		return diag.FromErr(fmt.Errorf("Error looking up project with Name %s or ID %s, %+v ", name, id, err))
	}

	processTemplateID := (*project.Capabilities)["processTemplate"]["templateTypeId"]
	processtemplateName, err := lookupProcessTemplateName(clients, processTemplateID)

	d.SetId(project.Id.String())
	d.Set("name", project.Name)
	d.Set("visibility", project.Visibility)
	d.Set("description", project.Description)
	d.Set("process_template_id", processTemplateID)
	d.Set("work_item_template", processtemplateName)
	d.Set("version_control", (*project.Capabilities)["versioncontrol"]["sourceControlType"])
	d.Set("project_id", project.Id.String())

	return nil
}

func lookupWorkItemTemplate(clients *client.AggregatedClient, templateId string) (string, error) {
	processes, err := clients.CoreClient.GetProcesses(clients.Ctx, core.GetProcessesArgs{})
	if err != nil {
		log.Fatal(err)
	}
	for _, process := range *processes {
		if *process.Name == "Agile" {
			return process.Id.String(), nil
		}
	}

	return "", fmt.Errorf("no process template found")
}

func lookupProcessTemplateName(clients *client.AggregatedClient, templateID string) (string, error) {
	id, err := uuid.Parse(templateID)
	if err != nil {
		return "", fmt.Errorf("error parsing Work Item Template ID, got %s: %v", templateID, err)
	}

	process, err := clients.CoreClient.GetProcessById(clients.Ctx, core.GetProcessByIdArgs{
		ProcessId: &id,
	})

	if err != nil {
		return "", fmt.Errorf("error looking up template by ID: %v", err)
	}

	return *process.Name, nil
}
