package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceGarageBuckets() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceGarageBucketsRead,
		Schema: map[string]*schema.Schema{
			"buckets": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of all buckets",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"global_aliases": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"local_aliases": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"alias": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"access_key_id": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceGarageBucketsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)

	buckets, resp, err := client.Client.BucketAPI.ListBuckets(ctx).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to list buckets: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	bucketList := make([]map[string]interface{}, 0, len(buckets))
	for _, bucket := range buckets {
		globalAliases := make([]interface{}, len(bucket.GlobalAliases))
		for i, alias := range bucket.GlobalAliases {
			globalAliases[i] = alias
		}

		localAliases := make([]interface{}, len(bucket.LocalAliases))
		for i, localAlias := range bucket.LocalAliases {
			localAliases[i] = map[string]interface{}{
				"alias":         localAlias.Alias,
				"access_key_id": localAlias.AccessKeyId,
			}
		}

		bucketList = append(bucketList, map[string]interface{}{
			"id":             bucket.Id,
			"global_aliases": globalAliases,
			"local_aliases":  localAliases,
		})
	}

	if err := d.Set("buckets", bucketList); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("garage_buckets")
	return nil
}
