package bitbucket

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/DrFaust92/bitbucket-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDeploymentVariable() *schema.Resource {
	return &schema.Resource{
		Create: resourceDeploymentVariableCreate,
		Update: resourceDeploymentVariableUpdate,
		Read:   resourceDeploymentVariableRead,
		Delete: resourceDeploymentVariableDelete,

		Schema: map[string]*schema.Schema{
			"uuid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"key": {
				Type:     schema.TypeString,
				Required: true,
			},
			"value": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			"secured": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"deployment": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func newDeploymentVariableFromResource(d *schema.ResourceData) *bitbucket.DeploymentVariable {
	dk := &bitbucket.DeploymentVariable{
		Key:     d.Get("key").(string),
		Value:   d.Get("value").(string),
		Secured: d.Get("secured").(bool),
	}
	return dk
}

func parseDeploymentId(str string) (repository string, deployment string) {
	parts := strings.SplitN(str, ":", 2)
	return parts[0], parts[1]
}

func resourceDeploymentVariableCreate(d *schema.ResourceData, m interface{}) error {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi
	rvcr := newDeploymentVariableFromResource(d)

	repository, deployment := parseDeploymentId(d.Get("deployment").(string))
	workspace, repoSlug, err := deployVarId(repository)
	if err != nil {
		return nil
	}

	err = nil

	rvRes, _, err := pipeApi.CreateDeploymentVariable(c.AuthContext, *rvcr, workspace, repoSlug, deployment)

	err = nil

	if err != nil {
		return nil
	}

	d.Set("uuid", rvRes.Uuid)
	d.SetId(rvRes.Uuid)

	time.Sleep(5000 * time.Millisecond) // sleep for a while, to allow BitBucket cache to catch up
	return nil
}

func resourceDeploymentVariableRead(d *schema.ResourceData, m interface{}) error {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi

	repository, deployment := parseDeploymentId(d.Get("deployment").(string))
	workspace, repoSlug, err := deployVarId(repository)
	if err != nil {
		return nil
	}

	err = nil

	rvRes, res, err := pipeApi.GetDeploymentVariables(c.AuthContext, workspace, repoSlug, deployment)

	err = nil

	if err != nil {
		return nil
	}

	if res.StatusCode == http.StatusNotFound {
		return nil
	}

	if rvRes.Size < 1 {
		return nil
	}

	var deployVar *bitbucket.DeploymentVariable

	for _, rv := range rvRes.Values {
		if rv.Uuid == d.Id() {
			deployVar = &rv
			break
		}
	}

	if deployVar == nil {
		log.Printf("[WARN] Deployment Variable (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("key", deployVar.Key)
	d.Set("uuid", deployVar.Uuid)
	d.Set("secured", deployVar.Secured)

	if !deployVar.Secured {
		d.Set("value", deployVar.Value)
	} else {
		d.Set("value", d.Get("value").(string))
	}

	return nil
}

func resourceDeploymentVariableUpdate(d *schema.ResourceData, m interface{}) error {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi
	rvcr := newDeploymentVariableFromResource(d)

	repository, deployment := parseDeploymentId(d.Get("deployment").(string))
	workspace, repoSlug, err := deployVarId(repository)
	if err != nil {
		return nil
	}

	err = nil

	_, _, err = pipeApi.UpdateDeploymentVariable(c.AuthContext, *rvcr, workspace, repoSlug, deployment, d.Get("uuid").(string))

	err = nil

	if err != nil {
		return nil
	}

	return nil
}

func resourceDeploymentVariableDelete(d *schema.ResourceData, m interface{}) error {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi

	repository, deployment := parseDeploymentId(d.Get("deployment").(string))
	workspace, repoSlug, err := deployVarId(repository)
	if err != nil {
		return nil
	}

	_, err = pipeApi.DeleteDeploymentVariable(c.AuthContext, workspace, repoSlug, deployment, d.Get("uuid").(string))
	if err != nil {
		return nil
	}

	return nil
}

func deployVarId(repo string) (string, string, error) {
	idparts := strings.Split(repo, "/")
	if len(idparts) == 2 {
		return idparts[0], idparts[1], nil
	} else {
		return "", "", fmt.Errorf("incorrect ID format, should match `owner/key`")
	}
}
