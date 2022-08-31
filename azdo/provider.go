package azdo

import (
	"azdo/azdo/client"
	core2 "azdo/azdo/core"
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"azdo_project": core2.ResourceProject(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"azdo_projects": core2.DataProjects(),
			"azdo_project":  core2.DataProject(),
		},
		Schema: map[string]*schema.Schema{
			"org_service_url": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("AZDO_ORG_SERVICE_URL", nil),
				Description: "The url of the Azure DevOps instance which should be used.",
			},
			"personal_access_token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("AZDO_PERSONAL_ACCESS_TOKEN", nil),
				Description: "The personal access token which should be used.",
				Sensitive:   true,
			},
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	p := schema.Provider{}
	terraformVersion := &p.TerraformVersion
	if *terraformVersion == "" {
		// Terraform 0.12 introduced this field to the protocol
		// We can therefore assume that if it's missing it's 0.10 or 0.11
		*terraformVersion = "0.11+compatible"
	}

	azdoclient, err := client.GetAzdoClient(d.Get("personal_access_token").(string), d.Get("org_service_url").(string), *terraformVersion)

	return azdoclient, diag.FromErr(err)
}
