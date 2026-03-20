package main

import (
	"context"
	"fmt"
	"net/http"

	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceGarageKey() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGarageKeyCreate,
		ReadContext:   resourceGarageKeyRead,
		UpdateContext: resourceGarageKeyUpdate,
		DeleteContext: resourceGarageKeyDelete,
		Importer:      &schema.ResourceImporter{},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the access key",
			},
			"allow_create_bucket": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether the key is allowed to create buckets",
			},
			"access_key_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The access key ID",
			},
			"secret_access_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The secret access key (only available on create)",
			},
		},
	}
}

func resourceGarageKeyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	name := d.Get("name").(string)

	keyBody := garage.NewUpdateKeyRequestBody()
	keyBody.SetName(name)

	key, resp, err := client.Client.AccessKeyAPI.CreateKey(ctx).Body(*keyBody).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create key: %w", err))
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	d.SetId(key.AccessKeyId)
	if err := d.Set("access_key_id", key.AccessKeyId); err != nil {
		return diag.FromErr(err)
	}
	if d.Get("allow_create_bucket").(bool) {
		keyPerm := garage.NewKeyPerm()
		keyPerm.SetCreateBucket(true)

		updateBody := garage.NewUpdateKeyRequestBody()
		updateBody.SetAllow(*keyPerm)

		_, resp, err := client.Client.AccessKeyAPI.UpdateKey(ctx).Id(key.AccessKeyId).UpdateKeyRequestBody(*updateBody).Execute()
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to update key permissions: %w", err))
		}
		defer func() {
			if resp.Body != nil {
				_ = resp.Body.Close()
			}
		}()
	}
	if key.SecretAccessKey.IsSet() {
		secret := key.SecretAccessKey.Get()
		if secret != nil {
			if err := d.Set("secret_access_key", *secret); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	return nil
}

func resourceGarageKeyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	keyID := d.Id()

	key, resp, err := client.Client.AccessKeyAPI.GetKeyInfo(ctx).Id(keyID).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to read key: %w", err))
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if err := d.Set("access_key_id", key.AccessKeyId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("name", key.Name); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("allow_create_bucket", key.Permissions.GetCreateBucket()); err != nil {
		return diag.FromErr(err)
	}
	// Note: secret_access_key is not available on read, only on create

	return nil
}

func resourceGarageKeyUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	keyID := d.Id()

	updateBody := garage.NewUpdateKeyRequestBody()

	if d.HasChange("name") {
		name := d.Get("name").(string)
		updateBody.SetName(name)
	}
	if d.HasChange("allow_create_bucket") {
		allow := d.Get("allow_create_bucket").(bool)
		perm := garage.NewKeyPerm()
		perm.SetCreateBucket(allow)
		updateBody.SetAllow(*perm)
	}

	_, resp, err := client.Client.AccessKeyAPI.UpdateKey(ctx).Id(keyID).UpdateKeyRequestBody(*updateBody).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update key: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	return resourceGarageKeyRead(ctx, d, m)
}

func resourceGarageKeyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	keyID := d.Id()

	resp, err := client.Client.AccessKeyAPI.DeleteKey(ctx).Id(keyID).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete key: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	d.SetId("")
	return nil
}
