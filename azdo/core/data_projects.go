package core

import (
	"azdo/azdo/client"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v6/core"
)

// DataProject schema and implementation for project data source

func DataProjects() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataProjectsRead,
		Schema: map[string]*schema.Schema{
			"projects": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"project_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"description": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

// Introducing a read method here which is almost the same code a in resource_project.go
// but this follows the `A little copying is better than a little dependency.` GO proverb.
func dataProjectsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	clients := m.(*client.AggregatedClient)
	var projects []core.TeamProjectReference
	outputs := make([]map[string]interface{}, 0)
	projectValues, err := clients.CoreClient.GetProjects(ctx, core.GetProjectsArgs{})
	projects = projectValues.Value
	for _, project := range projects {
		output := make(map[string]interface{})
		if project.Name != nil {
			output["name"] = *project.Name
		}
		if project.Description != nil {
			output["description"] = project.Description
		}
		if project.Id != nil {
			output["project_id"] = project.Id.String()
		}
		outputs = append(outputs, output)
	}
	h := sha1.New()
	d.SetId("projects#" + base64.URLEncoding.EncodeToString(h.Sum(nil)))
	err = d.Set("projects", outputs)
	if err != nil {
		return diag.FromErr(err)
	}
	//d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return nil
}
