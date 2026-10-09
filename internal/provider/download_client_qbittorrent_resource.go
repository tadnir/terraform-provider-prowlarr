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
	downloadClientQbittorrentResourceName   = "download_client_qbittorrent"
	downloadClientQbittorrentImplementation = "QBittorrent"
	downloadClientQbittorrentConfigContract = "QBittorrentSettings"
	downloadClientQbittorrentProtocol       = "torrent"
)

var downloadClientQbittorrentInitialStates = []int64{0, 1, 2}

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &DownloadClientQbittorrentResource{}
	_ resource.ResourceWithImportState = &DownloadClientQbittorrentResource{}
)

func NewDownloadClientQbittorrentResource() resource.Resource {
	return &DownloadClientQbittorrentResource{}
}

// DownloadClientQbittorrentResource defines the download client implementation.
type DownloadClientQbittorrentResource struct {
	client *prowlarr.APIClient
	auth   context.Context
}

// DownloadClientQbittorrent describes the download client data model.
type DownloadClientQbittorrent struct {
	Tags            types.Set    `tfsdk:"tags"`
	Categories      types.Set    `tfsdk:"categories"`
	Name            types.String `tfsdk:"name"`
	Host            types.String `tfsdk:"host"`
	URLBase         types.String `tfsdk:"url_base"`
	Username        types.String `tfsdk:"username"`
	Password        types.String `tfsdk:"password"`
	APIKey          types.String `tfsdk:"api_key"`
	APIKeyWO        types.String `tfsdk:"api_key_wo"`
	APIKeyWOVersion types.Int64  `tfsdk:"api_key_wo_version"`
	Category        types.String `tfsdk:"category"`
	ItemPriority    types.Int64  `tfsdk:"item_priority"`
	Priority        types.Int64  `tfsdk:"priority"`
	Port            types.Int64  `tfsdk:"port"`
	ID              types.Int64  `tfsdk:"id"`
	InitialState    types.Int64  `tfsdk:"initial_state"`
	UseSsl          types.Bool   `tfsdk:"use_ssl"`
	Enable          types.Bool   `tfsdk:"enable"`
}

func (d DownloadClientQbittorrent) toDownloadClient() *DownloadClient {
	return &DownloadClient{
		Tags:           d.Tags,
		Categories:     d.Categories,
		Name:           d.Name,
		Host:           d.Host,
		URLBase:        d.URLBase,
		Username:       d.Username,
		Password:       d.Password,
		APIKey:         d.APIKey,
		Category:       d.Category,
		ItemPriority:   d.ItemPriority,
		Priority:       d.Priority,
		Port:           d.Port,
		ID:             d.ID,
		InitialState:   d.InitialState,
		UseSsl:         d.UseSsl,
		Enable:         d.Enable,
		Implementation: types.StringValue(downloadClientQbittorrentImplementation),
		ConfigContract: types.StringValue(downloadClientQbittorrentConfigContract),
		Protocol:       types.StringValue(downloadClientQbittorrentProtocol),
	}
}

func (d *DownloadClientQbittorrent) fromDownloadClient(client *DownloadClient) {
	d.Tags = client.Tags
	d.Categories = client.Categories
	d.Name = client.Name
	d.Host = client.Host
	d.URLBase = client.URLBase
	d.Username = client.Username
	d.Password = client.Password
	d.APIKey = client.APIKey
	d.Category = client.Category
	d.ItemPriority = client.ItemPriority
	d.Priority = client.Priority
	d.Port = client.Port
	d.ID = client.ID
	d.InitialState = client.InitialState
	d.UseSsl = client.UseSsl
	d.Enable = client.Enable
}

func (r *DownloadClientQbittorrentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + downloadClientQbittorrentResourceName
}

func (r *DownloadClientQbittorrentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "<!-- subcategory:Download Clients -->\nDownload Client qBittorrent resource.\nFor more information refer to [Download Client](https://wiki.servarr.com/prowlarr/settings#download-clients) and [qBittorrent](https://wiki.servarr.com/prowlarr/supported#qbittorrent).",
		Attributes: map[string]schema.Attribute{
			"enable": schema.BoolAttribute{
				MarkdownDescription: "Enable flag.",
				Optional:            true,
				Computed:            true,
			},
			"priority": schema.Int64Attribute{
				MarkdownDescription: "Priority.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Download Client name.",
				Required:            true,
			},
			"tags": schema.SetAttribute{
				MarkdownDescription: "List of associated tags.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.Int64Type,
			},
			"categories": schema.SetNestedAttribute{
				MarkdownDescription: "List of mapped categories.",
				Optional:            true,
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: DownloadClientResource{}.getClientCategorySchema().Attributes,
				},
			},
			"id": schema.Int64Attribute{
				MarkdownDescription: "Download Client ID.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			// Field values
			"use_ssl": schema.BoolAttribute{
				MarkdownDescription: "Use SSL flag.",
				Optional:            true,
				Computed:            true,
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: "Port.",
				Optional:            true,
				Computed:            true,
			},
			"item_priority": schema.Int64Attribute{
				MarkdownDescription: "Older Movie priority. `0` Last, `1` First.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.OneOf(0, 1),
				},
			},
			"initial_state": schema.Int64Attribute{
				MarkdownDescription: "Initial state, with Stop support. `0` Start, `1` ForceStart, `2` Pause.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.OneOf(downloadClientQbittorrentInitialStates...),
				},
			},
			"host": schema.StringAttribute{
				MarkdownDescription: "host.",
				Optional:            true,
				Computed:            true,
			},
			"url_base": schema.StringAttribute{
				MarkdownDescription: "Base URL.",
				Optional:            true,
				Computed:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username.",
				Optional:            true,
				Computed:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "API key, instead of username and password. Requires qBittorrent 5.2 or later. Kept in state; use `api_key_wo` to keep it out.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("api_key_wo")),
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
			"category": schema.StringAttribute{
				MarkdownDescription: "Category.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func (r *DownloadClientQbittorrentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if auth, client := resourceConfigure(ctx, req, resp); client != nil {
		r.client = client
		r.auth = auth
	}
}

func (r *DownloadClientQbittorrentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var client *DownloadClientQbittorrent

	resp.Diagnostics.Append(req.Plan.Get(ctx, &client)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// The write-only API key is only in the configuration.
	var apiKeyWO types.String

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("api_key_wo"), &apiKeyWO)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create new DownloadClientQbittorrent
	request := client.readWithAPIKeyWO(ctx, apiKeyWO, true, &resp.Diagnostics)

	response, _, err := r.client.DownloadClientAPI.CreateDownloadClient(r.auth).DownloadClientResource(*request).Execute()
	if err != nil {
		resp.Diagnostics.AddError(helpers.ClientError, helpers.ParseClientError(helpers.Create, downloadClientQbittorrentResourceName, err))

		return
	}

	tflog.Trace(ctx, "created "+downloadClientQbittorrentResourceName+": "+strconv.Itoa(int(response.GetId())))
	// Generate resource state struct
	client.write(ctx, response, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &client)...)
}

func (r *DownloadClientQbittorrentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var client DownloadClientQbittorrent

	resp.Diagnostics.Append(req.State.Get(ctx, &client)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get DownloadClientQbittorrent current value
	response, _, err := r.client.DownloadClientAPI.GetDownloadClientById(r.auth, int32(client.ID.ValueInt64())).Execute()
	if err != nil {
		resp.Diagnostics.AddError(helpers.ClientError, helpers.ParseClientError(helpers.Read, downloadClientQbittorrentResourceName, err))

		return
	}

	tflog.Trace(ctx, "read "+downloadClientQbittorrentResourceName+": "+strconv.Itoa(int(response.GetId())))
	// Map response body to resource schema attribute
	client.write(ctx, response, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &client)...)
}

func (r *DownloadClientQbittorrentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Get plan values
	var client *DownloadClientQbittorrent

	resp.Diagnostics.Append(req.Plan.Get(ctx, &client)...)

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

	// Update DownloadClientQbittorrent
	request := client.readWithAPIKeyWO(ctx, apiKeyWO, !client.APIKeyWOVersion.Equal(stateVersion), &resp.Diagnostics)

	response, _, err := r.client.DownloadClientAPI.UpdateDownloadClient(r.auth, strconv.Itoa(int(request.GetId()))).DownloadClientResource(*request).Execute()
	if err != nil {
		resp.Diagnostics.AddError(helpers.ClientError, helpers.ParseClientError(helpers.Update, downloadClientQbittorrentResourceName, err))

		return
	}

	tflog.Trace(ctx, "updated "+downloadClientQbittorrentResourceName+": "+strconv.Itoa(int(response.GetId())))
	// Generate resource state struct
	client.write(ctx, response, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &client)...)
}

func (r *DownloadClientQbittorrentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var ID int64

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &ID)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete DownloadClientQbittorrent current value
	_, err := r.client.DownloadClientAPI.DeleteDownloadClient(r.auth, int32(ID)).Execute()
	if err != nil {
		resp.Diagnostics.AddError(helpers.ClientError, helpers.ParseClientError(helpers.Delete, downloadClientQbittorrentResourceName, err))

		return
	}

	tflog.Trace(ctx, "deleted "+downloadClientQbittorrentResourceName+": "+strconv.Itoa(int(ID)))
	resp.State.RemoveResource(ctx)
}

func (r *DownloadClientQbittorrentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	helpers.ImportStatePassthroughIntID(ctx, path.Root("id"), req, resp)
	tflog.Trace(ctx, "imported "+downloadClientQbittorrentResourceName+": "+req.ID)
}

func (d *DownloadClientQbittorrent) write(ctx context.Context, downloadClient *prowlarr.DownloadClientResource, diags *diag.Diagnostics) {
	genericDownloadClient := d.toDownloadClient()
	genericDownloadClient.write(ctx, downloadClient, diags)
	d.fromDownloadClient(genericDownloadClient)
}

func (d *DownloadClientQbittorrent) read(ctx context.Context, diags *diag.Diagnostics) *prowlarr.DownloadClientResource {
	return d.toDownloadClient().read(ctx, diags)
}

// readWithAPIKeyWO is read, with the API key taken from the write-only
// api_key_wo when it is set: its value when send is true, otherwise the
// masked value, which the app answers by keeping the key it has.
func (d *DownloadClientQbittorrent) readWithAPIKeyWO(ctx context.Context, apiKeyWO types.String, send bool, diags *diag.Diagnostics) *prowlarr.DownloadClientResource {
	client := d.toDownloadClient()

	if !apiKeyWO.IsNull() && !apiKeyWO.IsUnknown() {
		if send {
			client.APIKey = apiKeyWO
		} else {
			client.APIKey = types.StringValue(helpers.SensitiveValue)
		}
	}

	return client.read(ctx, diags)
}
