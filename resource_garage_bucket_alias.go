package main

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceGarageBucketAlias() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGarageBucketAliasCreate,
		ReadContext:   resourceGarageBucketAliasRead,
		DeleteContext: resourceGarageBucketAliasDelete,

		Schema: map[string]*schema.Schema{
			"bucket_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The bucket ID",
			},
			"alias": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The alias name",
			},
			"access_key_id": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Access key ID for local aliases (required for local aliases, omit for global)",
			},
		},
	}
}

func resourceGarageBucketAliasCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	bucketID := d.Get("bucket_id").(string)
	alias := d.Get("alias").(string)
	accessKeyID := d.Get("access_key_id").(string)

	aliasEnum := garageBucketAliasEnum(bucketID, alias, accessKeyID)

	_, resp, err := client.Client.BucketAliasAPI.AddBucketAlias(ctx).BucketAliasEnum(aliasEnum).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to add bucket alias: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	d.SetId(garageBucketAliasID(bucketID, alias, accessKeyID))

	return resourceGarageBucketAliasRead(ctx, d, m)
}

func resourceGarageBucketAliasRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	bucketID := d.Get("bucket_id").(string)
	alias := d.Get("alias").(string)
	accessKeyID := d.Get("access_key_id").(string)

	if accessKeyID != "" {
		key, resp, err := client.Client.AccessKeyAPI.GetKeyInfo(ctx).Id(accessKeyID).Execute()
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				d.SetId("")
				return nil
			}
			return diag.FromErr(fmt.Errorf("failed to read key: %w", err))
		}
		defer func() {
			if resp.Body != nil {
				_ = resp.Body.Close()
			}
		}()

		for _, bucket := range key.Buckets {
			if bucket.Id != bucketID {
				continue
			}
			if slices.Contains(bucket.LocalAliases, alias) {
				d.SetId(garageBucketAliasID(bucketID, alias, accessKeyID))
				return nil
			}
			break
		}

		d.SetId("")
		return nil
	}

	bucket, resp, err := client.Client.BucketAPI.GetBucketInfo(ctx).Id(bucketID).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to read bucket: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if slices.Contains(bucket.GlobalAliases, alias) {
		d.SetId(garageBucketAliasID(bucketID, alias, accessKeyID))
		return nil
	}

	d.SetId("")
	return nil
}

func resourceGarageBucketAliasDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	bucketID := d.Get("bucket_id").(string)
	alias := d.Get("alias").(string)
	accessKeyID := d.Get("access_key_id").(string)

	aliasEnum := garageBucketAliasEnum(bucketID, alias, accessKeyID)

	_, resp, err := client.Client.BucketAliasAPI.RemoveBucketAlias(ctx).BucketAliasEnum(aliasEnum).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to remove bucket alias: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	d.SetId("")
	return nil
}

func garageBucketAliasEnum(bucketID, alias, accessKeyID string) garage.BucketAliasEnum {
	if accessKeyID != "" {
		localAlias := garage.NewBucketAliasEnumOneOf1(accessKeyID, bucketID, alias)
		return garage.BucketAliasEnumOneOf1AsBucketAliasEnum(localAlias)
	}

	globalAlias := garage.NewBucketAliasEnumOneOf(bucketID, alias)
	return garage.BucketAliasEnumOneOfAsBucketAliasEnum(globalAlias)
}

func garageBucketAliasID(bucketID, alias, accessKeyID string) string {
	if accessKeyID != "" {
		return fmt.Sprintf("%s/local:%s:%s", bucketID, accessKeyID, alias)
	}

	return fmt.Sprintf("%s/global:%s", bucketID, alias)
}
