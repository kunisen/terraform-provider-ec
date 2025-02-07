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

package trafficfilterresource

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/elastic/cloud-sdk-go/pkg/api/deploymentapi/trafficfilterapi"
	"github.com/elastic/cloud-sdk-go/pkg/client/deployments_traffic_filter"

	"github.com/elastic/terraform-provider-ec/ec/internal/util"
)

// Delete will delete an existing deployment traffic filter ruleset
func (r Resource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	if !resourceReady(r, &response.Diagnostics) {
		return
	}

	var state modelV0

	diags := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	res, err := trafficfilterapi.Get(trafficfilterapi.GetParams{
		API: r.client, ID: state.ID.Value, IncludeAssociations: true,
	})
	if err != nil {
		if !util.TrafficFilterNotFound(err) {
			response.Diagnostics.AddError(err.Error(), err.Error())
		}
		return
	}

	for _, assoc := range res.Associations {
		if err := trafficfilterapi.DeleteAssociation(trafficfilterapi.DeleteAssociationParams{
			API:        r.client,
			ID:         state.ID.Value,
			EntityID:   *assoc.ID,
			EntityType: *assoc.EntityType,
		}); err != nil {
			if !associationDeleted(err) {
				response.Diagnostics.AddError(err.Error(), err.Error())
				return
			}
		}
	}

	if err := trafficfilterapi.Delete(trafficfilterapi.DeleteParams{
		API: r.client, ID: state.ID.Value,
	}); err != nil {
		if !ruleDeleted(err) {
			response.Diagnostics.AddError(err.Error(), err.Error())
			return
		}
	}
}

func associationDeleted(err error) bool {
	var notFound *deployments_traffic_filter.DeleteTrafficFilterRulesetAssociationNotFound
	return errors.As(err, &notFound)
}

func ruleDeleted(err error) bool {
	var notFound *deployments_traffic_filter.DeleteTrafficFilterRulesetNotFound
	return errors.As(err, &notFound)
}
