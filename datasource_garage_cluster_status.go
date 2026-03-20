package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceGarageClusterStatus() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceGarageClusterStatusRead,
		Schema: map[string]*schema.Schema{
			"layout_version": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Current version number of the cluster layout",
			},
			"nodes": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of cluster nodes",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"zone": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"capacity": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"tags": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"is_up": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"last_seen_secs_ago": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceGarageClusterStatusRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)

	status, resp, err := client.Client.ClusterAPI.GetClusterStatus(ctx).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get cluster status: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if err := d.Set("layout_version", int(status.GetLayoutVersion())); err != nil {
		return diag.FromErr(err)
	}

	nodesResp := status.GetNodes()
	nodes := make([]map[string]interface{}, 0, len(nodesResp))
	for _, n := range nodesResp {
		role, hasRole := n.GetRoleOk()

		tags := []interface{}{}
		zone := ""
		capacity := 0
		if hasRole && role != nil {
			roleTags := role.GetTags()
			tags = make([]interface{}, len(roleTags))
			for i, t := range roleTags {
				tags[i] = t
			}
			zone = role.GetZone()
			capacity = int(role.GetCapacity())
		}

		nodeMap := map[string]interface{}{
			"id":                 n.GetId(),
			"zone":               zone,
			"capacity":           capacity,
			"tags":               tags,
			"is_up":              n.GetIsUp(),
			"last_seen_secs_ago": int(n.GetLastSeenSecsAgo()),
		}
		nodes = append(nodes, nodeMap)
	}

	if err := d.Set("nodes", nodes); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("cluster-status-v%d", status.GetLayoutVersion()))
	return nil
}
