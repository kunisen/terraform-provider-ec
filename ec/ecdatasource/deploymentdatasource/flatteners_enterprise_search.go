// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package deploymentdatasource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/elastic/cloud-sdk-go/pkg/models"

	"github.com/elastic/terraform-provider-ec/ec/internal/converters"
	"github.com/elastic/terraform-provider-ec/ec/internal/util"
)

// flattenEnterpriseSearchResources takes in EnterpriseSearch resource models and returns its
// flattened form.
func flattenEnterpriseSearchResources(ctx context.Context, in []*models.EnterpriseSearchResourceInfo) (types.List, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	var result = make([]enterpriseSearchResourceInfoModelV0, 0, len(in))

	for _, res := range in {
		model := enterpriseSearchResourceInfoModelV0{
			Topology: types.List{ElemType: types.ObjectType{AttrTypes: enterpriseSearchTopologyAttrTypes()}},
		}

		if res.ElasticsearchClusterRefID != nil {
			model.ElasticsearchClusterRefID = types.String{Value: *res.ElasticsearchClusterRefID}
		}

		if res.RefID != nil {
			model.RefID = types.String{Value: *res.RefID}
		}

		if res.Info != nil {
			if res.Info.Healthy != nil {
				model.Healthy = types.Bool{Value: *res.Info.Healthy}
			}

			if res.Info.ID != nil {
				model.ResourceID = types.String{Value: *res.Info.ID}
			}

			if res.Info.Status != nil {
				model.Status = types.String{Value: *res.Info.Status}
			}

			if !util.IsCurrentEssPlanEmpty(res) {
				var plan = res.Info.PlanInfo.Current.Plan

				if plan.EnterpriseSearch != nil {
					model.Version = types.String{Value: plan.EnterpriseSearch.Version}
				}

				var diags diag.Diagnostics
				model.Topology, diags = flattenEnterpriseSearchTopology(ctx, plan)
				diagnostics.Append(diags...)
			}

			if res.Info.Metadata != nil {
				model.HttpEndpoint, model.HttpsEndpoint = converters.ExtractEndpointsToTypes(res.Info.Metadata)
			}
		}

		result = append(result, model)
	}

	var target types.List
	diagnostics.Append(tfsdk.ValueFrom(ctx, result, types.ListType{
		ElemType: types.ObjectType{
			AttrTypes: enterpriseSearchResourceInfoAttrTypes(),
		},
	}, &target)...)

	return target, diagnostics
}

func flattenEnterpriseSearchTopology(ctx context.Context, plan *models.EnterpriseSearchPlan) (types.List, diag.Diagnostics) {
	var result = make([]enterpriseSearchTopologyModelV0, 0, len(plan.ClusterTopology))
	for _, topology := range plan.ClusterTopology {
		var model enterpriseSearchTopologyModelV0

		if isEsSizePopulated(topology) && *topology.Size.Value == 0 {
			continue
		}

		model.InstanceConfigurationID = types.String{Value: topology.InstanceConfigurationID}

		if isEsSizePopulated(topology) {
			model.Size = types.String{Value: util.MemoryToState(*topology.Size.Value)}
			model.SizeResource = types.String{Value: *topology.Size.Resource}
		}

		model.ZoneCount = types.Int64{Value: int64(topology.ZoneCount)}

		if topology.NodeType != nil {
			if topology.NodeType.Appserver != nil {
				model.NodeTypeAppserver = types.Bool{Value: *topology.NodeType.Appserver}
			}

			if topology.NodeType.Connector != nil {
				model.NodeTypeConnector = types.Bool{Value: *topology.NodeType.Connector}
			}

			if topology.NodeType.Worker != nil {
				model.NodeTypeWorker = types.Bool{Value: *topology.NodeType.Worker}
			}
		}

		result = append(result, model)
	}

	var target types.List
	diags := tfsdk.ValueFrom(ctx, result, types.ListType{
		ElemType: types.ObjectType{
			AttrTypes: enterpriseSearchTopologyAttrTypes(),
		},
	}, &target)

	return target, diags
}

func isEsSizePopulated(topology *models.EnterpriseSearchTopologyElement) bool {
	if topology.Size != nil && topology.Size.Value != nil {
		return true
	}

	return false
}
