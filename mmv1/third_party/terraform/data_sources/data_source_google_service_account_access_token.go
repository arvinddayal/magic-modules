package google

import (
	"fmt"
	"log"

	"strings"

	transport_tpg "github.com/hashicorp/terraform-provider-google/google/transport"
	"github.com/hashicorp/terraform-provider-google/google/verify"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	iamcredentials "google.golang.org/api/iamcredentials/v1"
)

func DataSourceGoogleServiceAccountAccessToken() *schema.Resource {

	return &schema.Resource{
		Read: dataSourceGoogleServiceAccountAccessTokenRead,
		Schema: map[string]*schema.Schema{
			"target_service_account": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: verify.ValidateRegexp("(" + strings.Join(PossibleServiceAccountNames, "|") + ")"),
			},
			"access_token": {
				Type:      schema.TypeString,
				Sensitive: true,
				Computed:  true,
			},
			"scopes": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					StateFunc: func(v interface{}) string {
						return canonicalizeServiceScope(v.(string))
					},
				},
				// ValidateFunc is not yet supported on lists or sets.
			},
			"delegates": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: verify.ValidateRegexp(ServiceAccountLinkRegex),
				},
			},
			"lifetime": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateDuration(), // duration <=3600s; TODO: support validateDuration(min,max)
				Default:      "3600s",
			},
		},
	}
}

func dataSourceGoogleServiceAccountAccessTokenRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*transport_tpg.Config)
	userAgent, err := generateUserAgentString(d, config.UserAgent)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Acquire Service Account AccessToken for %s", d.Get("target_service_account").(string))

	service := config.NewIamCredentialsClient(userAgent)

	name := fmt.Sprintf("projects/-/serviceAccounts/%s", d.Get("target_service_account").(string))
	tokenRequest := &iamcredentials.GenerateAccessTokenRequest{
		Lifetime:  d.Get("lifetime").(string),
		Delegates: convertStringSet(d.Get("delegates").(*schema.Set)),
		Scope:     canonicalizeServiceScopes(convertStringSet(d.Get("scopes").(*schema.Set))),
	}
	at, err := service.Projects.ServiceAccounts.GenerateAccessToken(name, tokenRequest).Do()
	if err != nil {
		return err
	}

	d.SetId(name)
	if err := d.Set("access_token", at.AccessToken); err != nil {
		return fmt.Errorf("Error setting access_token: %s", err)
	}

	return nil
}
