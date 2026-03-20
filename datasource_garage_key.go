package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceGarageKey() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceGarageKeyRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The exact access key ID to look up",
			},
			"search": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Partial key ID or name to search for",
			},
			"access_key_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The access key ID",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the access key",
			},
			"allow_create_bucket": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the key is allowed to create buckets",
			},
			"expired": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the key has expired",
			},
		},
	}
}

func dataSourceGarageKeyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)

	req := client.Client.AccessKeyAPI.GetKeyInfo(ctx)
	if id, ok := d.GetOk("id"); ok {
		req = req.Id(id.(string))
	}
	if search, ok := d.GetOk("search"); ok {
		req = req.Search(search.(string))
	}

	key, resp, err := req.Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return diag.FromErr(fmt.Errorf("key not found"))
		}
		return diag.FromErr(fmt.Errorf("failed to read key: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	d.SetId(key.AccessKeyId)
	if err := d.Set("access_key_id", key.AccessKeyId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("name", key.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("allow_create_bucket", key.Permissions.GetCreateBucket()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("expired", key.Expired); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
