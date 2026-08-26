// Copyright (c) 2026 Russell Obets
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/darkhonor/terraform-provider-technitium/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &AppRecordResource{}
	_ resource.ResourceWithImportState = &AppRecordResource{}
)

func NewAppRecordResource() resource.Resource {
	return &AppRecordResource{}
}

// AppRecordResource manages Technitium's singular APP RRset at a DNS name.
// APP records are deliberately separate from RecordResource: the server
// enforces one APP RRset per name and its delete API removes that RRset without
// accepting RDATA identity fields.
type AppRecordResource struct {
	client *client.Client
}

type AppRecordResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Zone         types.String `tfsdk:"zone"`
	Name         types.String `tfsdk:"name"`
	TTL          types.Int64  `tfsdk:"ttl"`
	AppName      types.String `tfsdk:"app_name"`
	ClassPath    types.String `tfsdk:"class_path"`
	RecordData   types.String `tfsdk:"record_data"`
	LastModified types.String `tfsdk:"last_modified"`
}

func (r *AppRecordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_record"
}

func (r *AppRecordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Technitium APP record. The referenced DNS application must already be installed on the server.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "APP record identifier (zone::name).",
				Computed:    true,
			},
			"zone": schema.StringAttribute{
				Description: "Parent zone name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Fully qualified domain name for the APP record.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ttl": schema.Int64Attribute{
				Description: "Time to live in seconds.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(3600),
			},
			"app_name": schema.StringAttribute{
				Description: "Installed Technitium DNS application name, for example Split Horizon.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"class_path": schema.StringAttribute{
				Description: "Application record handler class path, for example SplitHorizon.SimpleAddress.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"record_data": schema.StringAttribute{
				Description: "Opaque record data consumed by the selected application handler. Use jsonencode for handlers that expect JSON.",
				Required:    true,
			},
			"last_modified": schema.StringAttribute{
				Description: "Timestamp of last modification.",
				Computed:    true,
			},
		},
	}
}

func (r *AppRecordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	providerData, ok := req.ProviderData.(*TechnitiumProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *TechnitiumProviderData, got: %T", req.ProviderData))
		return
	}
	r.client = providerData.Client
}

func (r *AppRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AppRecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	record, err := r.client.RecordAdd(ctx, plan.Name.ValueString(), plan.Zone.ValueString(), "APP",
		int(plan.TTL.ValueInt64()), false, appRecordParams(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating APP record", err.Error())
		return
	}

	plan.ID = types.StringValue(buildAppRecordID(plan.Zone.ValueString(), plan.Name.ValueString()))
	plan.LastModified = types.StringValue(record.LastModified)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AppRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AppRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	records, err := r.client.RecordGet(ctx, state.Name.ValueString(), state.Zone.ValueString())
	if err != nil {
		if isRecordAlreadyGone(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading APP record", err.Error())
		return
	}

	for _, record := range records {
		if record.Type != "APP" {
			continue
		}

		state.TTL = types.Int64Value(int64(record.TTL))
		state.AppName = types.StringValue(recordRDataString(record, "appName"))
		state.ClassPath = types.StringValue(recordRDataString(record, "classPath"))
		state.RecordData = types.StringValue(recordRDataString(record, "data"))
		state.LastModified = types.StringValue(record.LastModified)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *AppRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AppRecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.RecordUpdate(ctx, plan.Name.ValueString(), plan.Zone.ValueString(), "APP",
		int(plan.TTL.ValueInt64()), appRecordParams(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating APP record", err.Error())
		return
	}

	plan.ID = types.StringValue(buildAppRecordID(plan.Zone.ValueString(), plan.Name.ValueString()))
	records, err := r.client.RecordGet(ctx, plan.Name.ValueString(), plan.Zone.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading APP record after update", err.Error())
		return
	}
	for _, record := range records {
		if record.Type == "APP" {
			plan.LastModified = types.StringValue(record.LastModified)
			break
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AppRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AppRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.RecordDelete(ctx, state.Name.ValueString(), state.Zone.ValueString(), "APP", nil); err != nil && !isRecordAlreadyGone(err) {
		resp.Diagnostics.AddError("Error deleting APP record", err.Error())
	}
}

func (r *AppRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	zone, name, ok := strings.Cut(req.ID, "::")
	if !ok || zone == "" || name == "" || strings.Contains(name, "::") {
		resp.Diagnostics.AddError("Invalid import ID", "Import ID must be in format zone::name (for example, example.com::www.example.com).")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone"), zone)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
}

func appRecordParams(model *AppRecordResourceModel) map[string]string {
	return map[string]string{
		"appName":    model.AppName.ValueString(),
		"classPath":  model.ClassPath.ValueString(),
		"recordData": model.RecordData.ValueString(),
	}
}

func buildAppRecordID(zone, name string) string {
	return zone + "::" + name
}

func recordRDataString(record client.Record, key string) string {
	if value, ok := record.RData[key]; ok {
		return fmt.Sprintf("%v", value)
	}
	return ""
}
