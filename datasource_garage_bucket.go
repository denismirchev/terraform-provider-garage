package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceGarageBucket() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceGarageBucketRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The exact bucket ID to look up",
			},
			"global_alias": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Global alias of the bucket to look up",
			},
			"search": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Partial bucket ID or alias to search for",
			},
			"global_aliases": {
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Global aliases for this bucket",
			},
			"objects": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of objects in this bucket",
			},
			"bytes": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Total bytes used by objects in this bucket",
			},
			"website_access": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether website access is enabled",
			},
		},
	}
}

func dataSourceGarageBucketRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)

	req := client.Client.BucketAPI.GetBucketInfo(ctx)
	if id, ok := d.GetOk("id"); ok {
		req = req.Id(id.(string))
	}
	if alias, ok := d.GetOk("global_alias"); ok {
		req = req.GlobalAlias(alias.(string))
	}
	if search, ok := d.GetOk("search"); ok {
		req = req.Search(search.(string))
	}

	bucket, resp, err := req.Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return diag.FromErr(fmt.Errorf("bucket not found"))
		}
		return diag.FromErr(fmt.Errorf("failed to read bucket: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	d.SetId(bucket.Id)

	aliases := make([]interface{}, len(bucket.GlobalAliases))
	for i, alias := range bucket.GlobalAliases {
		aliases[i] = alias
	}
	if err := d.Set("global_aliases", aliases); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("objects", int(bucket.Objects)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("bytes", int(bucket.Bytes)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("website_access", bucket.WebsiteAccess); err != nil {
		return diag.FromErr(err)
	}
	if len(bucket.GlobalAliases) > 0 {
		if err := d.Set("global_alias", bucket.GlobalAliases[0]); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
