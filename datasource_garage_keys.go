package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceGarageKeys() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceGarageKeysRead,
		Schema: map[string]*schema.Schema{
			"keys": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of all access keys",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceGarageKeysRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)

	keys, resp, err := client.Client.AccessKeyAPI.ListKeys(ctx).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to list keys: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	keyList := make([]map[string]interface{}, 0, len(keys))
	for _, k := range keys {
		keyList = append(keyList, map[string]interface{}{
			"id":   k.Id,
			"name": k.Name,
		})
	}

	if err := d.Set("keys", keyList); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("garage_keys")

	return nil
}
