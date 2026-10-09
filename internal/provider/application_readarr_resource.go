package provider

import (
	"context"
	"strconv"

	"github.com/devopsarr/prowlarr-go/prowlarr"
	"github.com/devopsarr/terraform-provider-prowlarr/internal/helpers"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	applicationReadarrResourceName   = "application_readarr"
	applicationReadarrImplementation = "Readarr"
	applicationReadarrConfigContract = "ReadarrSettings"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &ApplicationReadarrResource{}
	_ resource.ResourceWithImportState = &ApplicationReadarrResource{}
)

func NewApplicationReadarrResource() resource.Resource {
	return &ApplicationReadarrResource{}
}

// ApplicationReadarrResource defines the application implementation.
type ApplicationReadarrResource struct {
	client *prowlarr.APIClient
	auth   context.Context
}

// ApplicationReadarr describes the application data model.
type ApplicationReadarr struct {
	SyncCategories  types.Set    `tfsdk:"sync_categories"`
	Tags            types.Set    `tfsdk:"tags"`
	Name            types.String `tfsdk:"name"`
	SyncLevel       types.String `tfsdk:"sync_level"`
	ProwlarrURL     types.String `tfsdk:"prowlarr_url"`
	BaseURL         types.String `tfsdk:"base_url"`
	APIKey          types.String `tfsdk:"api_key"`
	APIKeyWO        types.String `tfsdk:"api_key_wo"`
	APIKeyWOVersion types.Int64  `tfsdk:"api_key_wo_version"`
	ID              types.Int64  `tfsdk:"id"`
}

func (a ApplicationReadarr) toApplication() *Application {
	return &Application{
		SyncCategories: a.SyncCategories,
		Tags:           a.Tags,
		Name:           a.Name,
		SyncLevel:      a.SyncLevel,
		ProwlarrURL:    a.ProwlarrURL,
		BaseURL:        a.BaseURL,
		APIKey:         a.APIKey,
		ID:             a.ID,
		ConfigContract: types.StringValue(applicationReadarrConfigContract),
		Implementation: types.StringValue(applicationReadarrImplementation),
	}
}

func (a *ApplicationReadarr) fromApplication(application *Application) {
	a.Tags = application.Tags
	a.Name = application.Name
	a.ID = application.ID
	a.SyncLevel = application.SyncLevel
	a.SyncCategories = application.SyncCategories
	a.BaseURL = application.BaseURL
	a.ProwlarrURL = application.ProwlarrURL
	a.APIKey = application.APIKey
}

func (r *ApplicationReadarrResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + applicationReadarrResourceName
}

func (r *ApplicationReadarrResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "<!-- subcategory:Applications -->\nApplication Readarr resource.\nFor more information refer to [Application](https://wiki.servarr.com/prowlarr/settings#applications) and [Readarr](https://wiki.servarr.com/prowlarr/supported#readarr).",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Application name.",
				Required:            true,
			},
			"sync_level": schema.StringAttribute{
				MarkdownDescription: "Sync level.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("addOnly", "disabled", "fullSync"),
				},
			},
			"tags": schema.SetAttribute{
				MarkdownDescription: "List of associated tags.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.Int64Type,
			},
			"id": schema.Int64Attribute{
				MarkdownDescription: "Application ID.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			// Field values
			"base_url": schema.StringAttribute{
				MarkdownDescription: "Base URL.",
				Required:            true,
			},
			"prowlarr_url": schema.StringAttribute{
				MarkdownDescription: "Prowlarr URL.",
				Required:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "API key. Either this or `api_key_wo` is required. Kept in state; use `api_key_wo` to keep it out.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("api_key"), path.MatchRoot("api_key_wo")),
				},
			},
			"api_key_wo": schema.StringAttribute{
				MarkdownDescription: "API key, as `api_key`, but write-only: it is never stored in plan or state. It is sent on create and whenever `api_key_wo_version` changes. Requires Terraform 1.11 or later.",
				Optional:            true,
				Sensitive:           true,
				WriteOnly:           true,
			},
			"api_key_wo_version": schema.Int64Attribute{
				MarkdownDescription: "Version of `api_key_wo`. Change it to send a new `api_key_wo`.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.AlsoRequires(path.MatchRoot("api_key_wo")),
				},
			},
			"sync_categories": schema.SetAttribute{
				MarkdownDescription: "Sync categories.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.Int64Type,
			},
		},
	}
}

func (r *ApplicationReadarrResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if auth, client := resourceConfigure(ctx, req, resp); client != nil {
		r.client = client
		r.auth = auth
	}
}

func (r *ApplicationReadarrResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var application *ApplicationReadarr

	resp.Diagnostics.Append(req.Plan.Get(ctx, &application)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// The write-only API key is only in the configuration.
	var apiKeyWO types.String

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("api_key_wo"), &apiKeyWO)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create new ApplicationReadarr
	request := application.readWithAPIKeyWO(ctx, apiKeyWO, true, &resp.Diagnostics)

	response, _, err := r.client.ApplicationAPI.CreateApplications(r.auth).ApplicationResource(*request).Execute()
	if err != nil {
		resp.Diagnostics.AddError(helpers.ClientError, helpers.ParseClientError(helpers.Create, applicationReadarrResourceName, err))

		return
	}

	tflog.Trace(ctx, "created "+applicationReadarrResourceName+": "+strconv.Itoa(int(response.GetId())))
	// Generate resource state struct
	application.write(ctx, response, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &application)...)
}

func (r *ApplicationReadarrResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var application *ApplicationReadarr

	resp.Diagnostics.Append(req.State.Get(ctx, &application)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get ApplicationReadarr current value
	response, _, err := r.client.ApplicationAPI.GetApplicationsById(r.auth, int32(application.ID.ValueInt64())).Execute()
	if err != nil {
		resp.Diagnostics.AddError(helpers.ClientError, helpers.ParseClientError(helpers.Read, applicationReadarrResourceName, err))

		return
	}

	tflog.Trace(ctx, "read "+applicationReadarrResourceName+": "+strconv.Itoa(int(response.GetId())))
	// Map response body to resource schema attribute
	application.write(ctx, response, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &application)...)
}

func (r *ApplicationReadarrResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Get plan values
	var application *ApplicationReadarr

	resp.Diagnostics.Append(req.Plan.Get(ctx, &application)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// The write-only API key is only in the configuration. It is sent when its
	// version changes; otherwise the app is told to keep the one it has.
	var (
		apiKeyWO     types.String
		stateVersion types.Int64
	)

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("api_key_wo"), &apiKeyWO)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("api_key_wo_version"), &stateVersion)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Update ApplicationReadarr
	request := application.readWithAPIKeyWO(ctx, apiKeyWO, !application.APIKeyWOVersion.Equal(stateVersion), &resp.Diagnostics)

	response, _, err := r.client.ApplicationAPI.UpdateApplications(r.auth, strconv.Itoa(int(request.GetId()))).ApplicationResource(*request).Execute()
	if err != nil {
		resp.Diagnostics.AddError(helpers.ClientError, helpers.ParseClientError(helpers.Update, applicationReadarrResourceName, err))

		return
	}

	tflog.Trace(ctx, "updated "+applicationReadarrResourceName+": "+strconv.Itoa(int(response.GetId())))
	// Generate resource state struct
	application.write(ctx, response, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &application)...)
}

func (r *ApplicationReadarrResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var ID int64

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &ID)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete ApplicationReadarr current value
	_, err := r.client.ApplicationAPI.DeleteApplications(r.auth, int32(ID)).Execute()
	if err != nil {
		resp.Diagnostics.AddError(helpers.ClientError, helpers.ParseClientError(helpers.Delete, applicationReadarrResourceName, err))

		return
	}

	tflog.Trace(ctx, "deleted "+applicationReadarrResourceName+": "+strconv.Itoa(int(ID)))
	resp.State.RemoveResource(ctx)
}

func (r *ApplicationReadarrResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	helpers.ImportStatePassthroughIntID(ctx, path.Root("id"), req, resp)
	tflog.Trace(ctx, "imported "+applicationReadarrResourceName+": "+req.ID)
}

func (a *ApplicationReadarr) write(ctx context.Context, application *prowlarr.ApplicationResource, diags *diag.Diagnostics) {
	genericApplication := a.toApplication()
	genericApplication.write(ctx, application, diags)
	a.fromApplication(genericApplication)
}

func (a *ApplicationReadarr) read(ctx context.Context, diags *diag.Diagnostics) *prowlarr.ApplicationResource {
	return a.toApplication().read(ctx, diags)
}

// readWithAPIKeyWO is read, with the API key taken from the write-only
// api_key_wo when it is set: its value when send is true, otherwise the
// masked value, which the app answers by keeping the key it has.
func (a *ApplicationReadarr) readWithAPIKeyWO(ctx context.Context, apiKeyWO types.String, send bool, diags *diag.Diagnostics) *prowlarr.ApplicationResource {
	application := a.toApplication()

	if !apiKeyWO.IsNull() && !apiKeyWO.IsUnknown() {
		if send {
			application.APIKey = apiKeyWO
		} else {
			application.APIKey = types.StringValue(helpers.SensitiveValue)
		}
	}

	return application.read(ctx, diags)
}
