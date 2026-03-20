package main

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"

	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceGarageBucket() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGarageBucketCreate,
		ReadContext:   resourceGarageBucketRead,
		UpdateContext: resourceGarageBucketUpdate,
		DeleteContext: resourceGarageBucketDelete,
		Importer:      &schema.ResourceImporter{},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The bucket ID (computed if not provided)",
			},
			"global_alias": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Global alias for the bucket (this appears as the name in garage bucket list)",
			},
			"bytes": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Total number of bytes used by objects in this bucket",
			},
			"objects": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of objects in this bucket",
			},
			"expiration_days": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Number of days after which objects in this bucket will be automatically deleted. Set to 0 to disable expiration.",
			},
			"max_size": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Maximum size quota for this bucket",
			},
			"max_objects": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Maximum number of objects quota for this bucket",
			},
			"website_access_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether website access is enabled for this bucket",
			},
			"website_access_index_document": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Which document to serve as index page for this bucket",
			},
			"website_access_error_document": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Which document to serve as error page for this bucket",
			},
			"website_config": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Website access configuration for the bucket",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Whether website access is enabled",
						},
						"index_document": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Index document for website access (required when enabled)",
						},
						"error_document": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Error document for website access",
						},
					},
				},
			},
			"quotas": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Quotas for the bucket",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"max_size": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Maximum bucket size in bytes",
						},
						"max_objects": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Maximum number of objects",
						},
					},
				},
			},
		},
	}
}

func resourceGarageBucketCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	globalAlias := d.Get("global_alias").(string)

	bucketInfo := garage.NewCreateBucketRequest()
	if globalAlias != "" {
		bucketInfo.SetGlobalAlias(globalAlias)
	}

	bucket, resp, err := client.Client.BucketAPI.CreateBucket(ctx).CreateBucketRequest(*bucketInfo).Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create bucket: %w", err))
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	d.SetId(bucket.Id)
	if err := d.Set("id", bucket.Id); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("bytes", bucket.Bytes); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("objects", bucket.Objects); err != nil {
		return diag.FromErr(err)
	}
	if len(bucket.GlobalAliases) > 0 {
		if err := d.Set("global_alias", bucket.GlobalAliases[0]); err != nil {
			return diag.FromErr(err)
		}
	}

	// Set quotas if specified
	quotas := garage.NewApiBucketQuotas()
	doUpdate := false

	if val, ok := d.GetOk("max_size"); ok {
		quotas.SetMaxSize(int64(val.(int)))
		doUpdate = true
	}
	if val, ok := d.GetOk("max_objects"); ok {
		quotas.SetMaxObjects(int64(val.(int)))
		doUpdate = true
	}

	// Set website access if specified
	websiteAccessEnabled := d.Get("website_access_enabled").(bool)
	websiteAccess := garage.NewUpdateBucketWebsiteAccess(websiteAccessEnabled)
	if websiteAccessEnabled {
		doUpdate = true
		if val, ok := d.GetOk("website_access_index_document"); ok {
			websiteAccess.SetIndexDocument(val.(string))
		} else {
			websiteAccess.SetIndexDocument("index.html")
		}
		if val, ok := d.GetOk("website_access_error_document"); ok {
			websiteAccess.SetErrorDocument(val.(string))
		}
	}

	if doUpdate {
		bucketUpdate := garage.NewUpdateBucketRequestBody()
		bucketUpdate.SetQuotas(*quotas)
		bucketUpdate.SetWebsiteAccess(*websiteAccess)
		_, resp, err := client.Client.BucketAPI.UpdateBucket(ctx).Id(bucket.Id).UpdateBucketRequestBody(*bucketUpdate).Execute()
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to update bucket: %w", err))
		}
		defer func() {
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
		}()
	}

	// Set expiration policy if specified
	if expirationDays, ok := d.GetOk("expiration_days"); ok && expirationDays.(int) > 0 {
		if err := setBucketLifecyclePolicy(ctx, client, bucket.Id, expirationDays.(int)); err != nil {
			return diag.FromErr(fmt.Errorf("failed to set expiration policy: %w", err))
		}
		if err := d.Set("expiration_days", expirationDays.(int)); err != nil {
			return diag.FromErr(err)
		}
	}

	if wc, ok := d.GetOk("website_config"); ok && len(wc.([]interface{})) > 0 {
		wcMap := wc.([]interface{})[0].(map[string]interface{})
		websiteAccess := garage.NewUpdateBucketWebsiteAccess(wcMap["enabled"].(bool))
		if idx := wcMap["index_document"].(string); idx != "" {
			websiteAccess.SetIndexDocument(idx)
		}
		if errDoc := wcMap["error_document"].(string); errDoc != "" {
			websiteAccess.SetErrorDocument(errDoc)
		}
		updateBody := garage.NewUpdateBucketRequestBody()
		updateBody.SetWebsiteAccess(*websiteAccess)

		_, updateResp, err := client.Client.BucketAPI.UpdateBucket(ctx).Id(bucket.Id).UpdateBucketRequestBody(*updateBody).Execute()
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to set website config: %w", err))
		}
		defer func() {
			if updateResp.Body != nil {
				_ = updateResp.Body.Close()
			}
		}()
	}

	if q, ok := d.GetOk("quotas"); ok && len(q.([]interface{})) > 0 {
		qMap := q.([]interface{})[0].(map[string]interface{})
		quotas := garage.NewApiBucketQuotas()
		if maxSz := qMap["max_size"].(int); maxSz > 0 {
			quotas.SetMaxSize(int64(maxSz))
		}
		if maxObj := qMap["max_objects"].(int); maxObj > 0 {
			quotas.SetMaxObjects(int64(maxObj))
		}
		updateBody := garage.NewUpdateBucketRequestBody()
		updateBody.SetQuotas(*quotas)

		_, updateResp, err := client.Client.BucketAPI.UpdateBucket(ctx).Id(bucket.Id).UpdateBucketRequestBody(*updateBody).Execute()
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to set quotas: %w", err))
		}
		defer func() {
			if updateResp.Body != nil {
				_ = updateResp.Body.Close()
			}
		}()
	}

	return resourceGarageBucketRead(ctx, d, m)
}

func resourceGarageBucketRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	bucketID := d.Id()

	bucket, resp, err := client.Client.BucketAPI.GetBucketInfo(ctx).Id(bucketID).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to read bucket: %w", err))
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if err := d.Set("id", bucket.Id); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("bytes", bucket.Bytes); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("objects", bucket.Objects); err != nil {
		return diag.FromErr(err)
	}
	quotas := bucket.GetQuotas()
	if val, ok := quotas.GetMaxSizeOk(); ok {
		if err := d.Set("max_size", val); err != nil {
			return diag.FromErr(err)
		}
	}
	if val, ok := quotas.GetMaxObjectsOk(); ok {
		if err := d.Set("max_objects", val); err != nil {
			return diag.FromErr(err)
		}
	}
	var websiteAccess garage.GetBucketInfoWebsiteResponse
	if bucket.WebsiteAccess {
		websiteAccess = bucket.GetWebsiteConfig()
	}
	if err := d.Set("website_access_enabled", bucket.WebsiteAccess); err != nil {
		return diag.FromErr(err)
	}
	_, indexDocumentSet := d.GetOk("website_access_index_document")
	newIndexDocument := websiteAccess.GetIndexDocument()
	// "nil" is mapped to the "index.html" default when updating the resource
	// Account for the default here to prevent updating the resource again
	if indexDocumentSet || newIndexDocument != "index.html" {
		if err := d.Set("website_access_index_document", newIndexDocument); err != nil {
			return diag.FromErr(err)
		}
	}
	if err := d.Set("website_access_error_document", websiteAccess.GetErrorDocument()); err != nil {
		return diag.FromErr(err)
	}

	if len(bucket.GlobalAliases) > 0 {
		if err := d.Set("global_alias", bucket.GlobalAliases[0]); err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set("global_alias", ""); err != nil {
			return diag.FromErr(err)
		}
	}

	// Read expiration policy if it exists
	expirationDays, err := getBucketLifecyclePolicy(ctx, client, bucket.Id)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read expiration policy: %w", err))
	}
	if expirationDays > 0 {
		if err := d.Set("expiration_days", expirationDays); err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set("expiration_days", 0); err != nil {
			return diag.FromErr(err)
		}
	}

	if _, ok := d.GetOk("website_config"); ok {
		websiteConfig := bucket.GetWebsiteConfig()
		if err := d.Set("website_config", []map[string]interface{}{
			{
				"enabled":        bucket.WebsiteAccess,
				"index_document": (&websiteConfig).GetIndexDocument(),
				"error_document": (&websiteConfig).GetErrorDocument(),
			},
		}); err != nil {
			return diag.FromErr(err)
		}
	}

	if _, ok := d.GetOk("quotas"); ok {
		if err := d.Set("quotas", []map[string]interface{}{
			{
				"max_size":    int(bucket.Quotas.GetMaxSize()),
				"max_objects": int(bucket.Quotas.GetMaxObjects()),
			},
		}); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourceGarageBucketUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	bucketID := d.Id()

	// Handle quota changes
	doUpdate := false
	var quotas *garage.ApiBucketQuotas
	if d.HasChanges("max_size", "max_objects") {
		doUpdate = true
		quotas = garage.NewApiBucketQuotas()
		if val, ok := d.GetOk("max_size"); ok {
			quotas.SetMaxSize(int64(val.(int)))
		}
		if val, ok := d.GetOk("max_objects"); ok {
			quotas.SetMaxObjects(int64(val.(int)))
		}
	}

	// Handle website access changes
	var websiteAccess *garage.UpdateBucketWebsiteAccess
	if d.HasChanges("website_access_enabled", "website_access_index_document", "website_access_error_document") {
		doUpdate = true
		websiteAccessEnabled := d.Get("website_access_enabled").(bool)
		websiteAccess = garage.NewUpdateBucketWebsiteAccess(websiteAccessEnabled)
		if websiteAccessEnabled {
			if val, ok := d.GetOk("website_access_index_document"); ok {
				websiteAccess.SetIndexDocument(val.(string))
			} else {
				websiteAccess.SetIndexDocument("index.html")
			}
			if val, ok := d.GetOk("website_access_error_document"); ok {
				websiteAccess.SetErrorDocument(val.(string))
			}
		}
	}

	if doUpdate {
		bucketUpdate := garage.NewUpdateBucketRequestBody()
		if quotas != nil {
			bucketUpdate.SetQuotas(*quotas)
		}
		if websiteAccess != nil {
			bucketUpdate.SetWebsiteAccess(*websiteAccess)
		}
		_, resp, err := client.Client.BucketAPI.UpdateBucket(ctx).Id(bucketID).UpdateBucketRequestBody(*bucketUpdate).Execute()
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to update bucket: %w", err))
		}
		defer func() {
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
		}()
	}

	// Handle expiration policy changes
	if d.HasChange("expiration_days") {
		expirationDays := d.Get("expiration_days").(int)
		if expirationDays > 0 {
			if err := setBucketLifecyclePolicy(ctx, client, bucketID, expirationDays); err != nil {
				return diag.FromErr(fmt.Errorf("failed to update expiration policy: %w", err))
			}
		} else {
			// Remove lifecycle policy if expiration_days is 0 or removed
			if err := deleteBucketLifecyclePolicy(ctx, client, bucketID); err != nil {
				return diag.FromErr(fmt.Errorf("failed to remove expiration policy: %w", err))
			}
		}
	}

	if d.HasChange("website_config") {
		websiteAccess := garage.NewUpdateBucketWebsiteAccess(false)
		wc := d.Get("website_config").([]interface{})
		if len(wc) > 0 {
			wcMap := wc[0].(map[string]interface{})
			websiteAccess.SetEnabled(wcMap["enabled"].(bool))
			if idx := wcMap["index_document"].(string); idx != "" {
				websiteAccess.SetIndexDocument(idx)
			} else {
				websiteAccess.SetIndexDocumentNil()
			}
			if errDoc := wcMap["error_document"].(string); errDoc != "" {
				websiteAccess.SetErrorDocument(errDoc)
			} else {
				websiteAccess.SetErrorDocumentNil()
			}
		} else {
			websiteAccess.SetIndexDocumentNil()
			websiteAccess.SetErrorDocumentNil()
		}
		updateBody := garage.NewUpdateBucketRequestBody()
		updateBody.SetWebsiteAccess(*websiteAccess)

		_, resp, err := client.Client.BucketAPI.UpdateBucket(ctx).Id(bucketID).UpdateBucketRequestBody(*updateBody).Execute()
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to update website config: %w", err))
		}
		defer func() {
			if resp.Body != nil {
				_ = resp.Body.Close()
			}
		}()
	}

	if d.HasChange("quotas") {
		q := d.Get("quotas").([]interface{})
		updateBody := garage.NewUpdateBucketRequestBody()
		if len(q) > 0 {
			qMap := q[0].(map[string]interface{})
			quotas := garage.NewApiBucketQuotas()
			if maxSz := qMap["max_size"].(int); maxSz > 0 {
				quotas.SetMaxSize(int64(maxSz))
			} else {
				quotas.SetMaxSizeNil()
			}
			if maxObj := qMap["max_objects"].(int); maxObj > 0 {
				quotas.SetMaxObjects(int64(maxObj))
			} else {
				quotas.SetMaxObjectsNil()
			}
			updateBody.SetQuotas(*quotas)
		} else {
			updateBody.SetQuotasNil()
		}

		_, resp, err := client.Client.BucketAPI.UpdateBucket(ctx).Id(bucketID).UpdateBucketRequestBody(*updateBody).Execute()
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to update quotas: %w", err))
		}
		defer func() {
			if resp.Body != nil {
				_ = resp.Body.Close()
			}
		}()
	}

	return resourceGarageBucketRead(ctx, d, m)
}

func resourceGarageBucketDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*GarageClient)
	bucketID := d.Id()

	resp, err := client.Client.BucketAPI.DeleteBucket(ctx).Id(bucketID).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to delete bucket: %w", err))
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	d.SetId("")
	return nil
}

// S3 Lifecycle Configuration structures
type LifecycleConfiguration struct {
	XMLName xml.Name `xml:"LifecycleConfiguration"`
	Rules   []Rule   `xml:"Rule"`
}

type Rule struct {
	ID         string      `xml:"ID"`
	Status     string      `xml:"Status"`
	Filter     *Filter     `xml:"Filter,omitempty"`
	Expiration *Expiration `xml:"Expiration,omitempty"`
}

type Filter struct {
	Prefix string `xml:"Prefix"`
}

type Expiration struct {
	Days int `xml:"Days"`
}

// setBucketLifecyclePolicy sets the lifecycle expiration policy for a bucket using S3-compatible API
func setBucketLifecyclePolicy(ctx context.Context, client *GarageClient, bucketID string, expirationDays int) error {
	// Get bucket info to find the bucket alias/name
	bucket, resp, err := client.Client.BucketAPI.GetBucketInfo(ctx).Id(bucketID).Execute()
	if err != nil {
		return fmt.Errorf("failed to get bucket info: %w", err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	// Get bucket name (alias)
	bucketName := bucketID
	if len(bucket.GlobalAliases) > 0 {
		bucketName = bucket.GlobalAliases[0]
	}

	// Construct S3 endpoint URL (Garage S3 API typically uses port 3900)
	// Replace admin port (3903) with S3 port (3900)
	s3URL := fmt.Sprintf("%s://%s/%s?lifecycle",
		client.Client.GetConfig().Scheme,
		replacePort(client.Client.GetConfig().Host, 3900),
		bucketName)

	// Create lifecycle configuration XML
	lifecycleConfig := LifecycleConfiguration{
		Rules: []Rule{
			{
				ID:     "expire-after-days",
				Status: "Enabled",
				Expiration: &Expiration{
					Days: expirationDays,
				},
			},
		},
	}

	xmlData, err := xml.MarshalIndent(lifecycleConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lifecycle config: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "PUT", s3URL, bytes.NewReader(xmlData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/xml")
	// Use admin token for authorization (Garage may accept this for S3 API too)
	if authHeader, ok := client.Client.GetConfig().DefaultHeader["Authorization"]; ok {
		req.Header.Set("Authorization", authHeader)
	}

	// Execute request
	httpClient := &http.Client{}
	resp, err = httpClient.Do(req)
	if err != nil || resp == nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// getBucketLifecyclePolicy retrieves the lifecycle expiration policy for a bucket
func getBucketLifecyclePolicy(ctx context.Context, client *GarageClient, bucketID string) (int, error) {
	// Get bucket info to find the bucket alias/name
	bucket, resp, err := client.Client.BucketAPI.GetBucketInfo(ctx).Id(bucketID).Execute()
	if err != nil {
		return 0, fmt.Errorf("failed to get bucket info: %w", err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	// Get bucket name (alias)
	bucketName := bucketID
	if len(bucket.GlobalAliases) > 0 {
		bucketName = bucket.GlobalAliases[0]
	}

	// Construct S3 endpoint URL
	s3URL := fmt.Sprintf("%s://%s/%s?lifecycle",
		client.Client.GetConfig().Scheme,
		replacePort(client.Client.GetConfig().Host, 3900),
		bucketName)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", s3URL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Use admin token for authorization
	if authHeader, ok := client.Client.GetConfig().DefaultHeader["Authorization"]; ok {
		req.Header.Set("Authorization", authHeader)
	}

	// Execute request
	httpClient := &http.Client{}
	resp, err = httpClient.Do(req)
	if err != nil || resp == nil {
		return 0, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if resp.StatusCode == http.StatusNotFound {
		return 0, nil // No lifecycle policy set
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse XML response
	var lifecycleConfig LifecycleConfiguration
	if err := xml.NewDecoder(resp.Body).Decode(&lifecycleConfig); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract expiration days from first rule
	if len(lifecycleConfig.Rules) > 0 && lifecycleConfig.Rules[0].Expiration != nil {
		return lifecycleConfig.Rules[0].Expiration.Days, nil
	}

	return 0, nil
}

// deleteBucketLifecyclePolicy removes the lifecycle policy from a bucket
func deleteBucketLifecyclePolicy(ctx context.Context, client *GarageClient, bucketID string) error {
	// Get bucket info to find the bucket alias/name
	bucket, resp, err := client.Client.BucketAPI.GetBucketInfo(ctx).Id(bucketID).Execute()
	if err != nil {
		return fmt.Errorf("failed to get bucket info: %w", err)
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	// Get bucket name (alias)
	bucketName := bucketID
	if len(bucket.GlobalAliases) > 0 {
		bucketName = bucket.GlobalAliases[0]
	}

	// Construct S3 endpoint URL
	s3URL := fmt.Sprintf("%s://%s/%s?lifecycle",
		client.Client.GetConfig().Scheme,
		replacePort(client.Client.GetConfig().Host, 3900),
		bucketName)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "DELETE", s3URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Use admin token for authorization
	if authHeader, ok := client.Client.GetConfig().DefaultHeader["Authorization"]; ok {
		req.Header.Set("Authorization", authHeader)
	}

	// Execute request
	httpClient := &http.Client{}
	resp, err = httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// replacePort replaces the port in a host string (e.g., "127.0.0.1:3903" -> "127.0.0.1:3900")
func replacePort(host string, newPort int) string {
	// Simple implementation - if host contains a port, replace it
	// Otherwise, append the new port
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			return host[:i+1] + strconv.Itoa(newPort)
		}
	}
	return host + ":" + strconv.Itoa(newPort)
}
