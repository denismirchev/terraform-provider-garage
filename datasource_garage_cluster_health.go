package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceGarageClusterHealth() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceGarageClusterHealthRead,
		Schema: map[string]*schema.Schema{
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Cluster health status (healthy, degraded, or unavailable)",
			},
			"connected_nodes": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of currently connected nodes",
			},
			"known_nodes": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of known nodes",
			},
			"storage_nodes": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of storage nodes in the cluster layout",
			},
			"storage_nodes_up": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of storage nodes currently connected",
			},
			"partitions": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Total number of data partitions",
			},
			"partitions_quorum": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Partitions with quorum available",
			},
			"partitions_all_ok": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Partitions with all storage nodes available",
			},
		},
	}
}

func dataSourceGarageClusterHealthRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)

	health, resp, err := client.Client.ClusterAPI.GetClusterHealth(ctx).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get cluster health: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	vals := map[string]interface{}{
		"status":            health.GetStatus(),
		"connected_nodes":   int(health.GetConnectedNodes()),
		"known_nodes":       int(health.GetKnownNodes()),
		"storage_nodes":     int(health.GetStorageNodes()),
		"storage_nodes_up":  int(health.GetStorageNodesUp()),
		"partitions":        int(health.GetPartitions()),
		"partitions_quorum": int(health.GetPartitionsQuorum()),
		"partitions_all_ok": int(health.GetPartitionsAllOk()),
	}
	for k, v := range vals {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(fmt.Sprintf("cluster-health-%s", health.GetStatus()))
	return nil
}
