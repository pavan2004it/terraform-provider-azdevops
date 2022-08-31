terraform {

  required_providers {
    azdo = {
      version = "1.0.0"
      source  = "hashicorp.com/azdo/azdo"
    }
  }
}


data "azdo_projects" "all_projects" {}

output "project_details" {
  value = {
    for key,project in data.azdo_projects.all_projects.projects :
    key => project
  }
}

#data "azdotf_project" "sample" {
#  name = "Docker"
#}

#output "sample_details" {
#  value = data.azdotf_project.sample
#}

#resource "azdotf_project" "example" {
#  name               = "Sample Project"
#  visibility         = "private"
#  version_control    = "Git"
#  work_item_template = "Agile"
#  description        = "Managed by Pavan"
#}
#
#
#data "azdotf_project" "project_data"{
#  name = azdotf_project.example.name
#}
#
#output "project_identification_number" {
#  value = data.azdotf_project.project_data
#}