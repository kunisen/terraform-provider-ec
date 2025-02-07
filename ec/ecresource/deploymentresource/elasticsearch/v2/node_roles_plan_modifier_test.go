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

package v2_test

import (
	"context"
	"testing"

	deploymentv2 "github.com/elastic/terraform-provider-ec/ec/ecresource/deploymentresource/deployment/v2"
	v2 "github.com/elastic/terraform-provider-ec/ec/ecresource/deploymentresource/elasticsearch/v2"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func Test_nodeRolesPlanModifier(t *testing.T) {
	type args struct {
		attributeState  []string
		attributePlan   []string
		deploymentState *deploymentv2.Deployment
		deploymentPlan  deploymentv2.Deployment
	}
	tests := []struct {
		name            string
		args            args
		expected        []string
		expectedUnknown bool
	}{
		{
			name: "it should keep current plan value if it's defined",
			args: args{
				attributePlan: []string{
					"data_content",
					"data_hot",
					"ingest",
					"master",
				},
			},
			expected: []string{
				"data_content",
				"data_hot",
				"ingest",
				"master",
			},
		},

		{
			name:            "it should not use state if state doesn't have `version`",
			args:            args{},
			expectedUnknown: true,
		},

		{
			name: "it should not use state if plan changed deployment template`",
			args: args{
				deploymentState: &deploymentv2.Deployment{
					DeploymentTemplateId: "aws-io-optimized-v2",
				},
				deploymentPlan: deploymentv2.Deployment{
					DeploymentTemplateId: "aws-storage-optimized-v3",
				},
			},
			expectedUnknown: true,
		},

		{
			name: "it should not use state if plan version is less than 7.10.0 but the attribute state is not null`",
			args: args{
				attributeState: []string{"data_hot"},
				deploymentState: &deploymentv2.Deployment{
					DeploymentTemplateId: "aws-io-optimized-v2",
				},
				deploymentPlan: deploymentv2.Deployment{
					DeploymentTemplateId: "aws-io-optimized-v2",
					Version:              "7.9.0",
				},
			},
			expectedUnknown: true,
		},

		{
			name: "it should not use state if plan version is changed over 7.10.0 and the attribute state is not null`",
			args: args{
				attributeState: []string{"data_hot"},
				deploymentState: &deploymentv2.Deployment{
					DeploymentTemplateId: "aws-io-optimized-v2",
					Version:              "7.9.0",
				},
				deploymentPlan: deploymentv2.Deployment{
					DeploymentTemplateId: "aws-io-optimized-v2",
					Version:              "7.10.1",
				},
			},
			expectedUnknown: true,
		},

		{
			name: "it should use state if plan version is changed over 7.10.0 and the attribute state is null`",
			args: args{
				attributeState: nil,
				deploymentState: &deploymentv2.Deployment{
					DeploymentTemplateId: "aws-io-optimized-v2",
					Version:              "7.9.0",
				},
				deploymentPlan: deploymentv2.Deployment{
					DeploymentTemplateId: "aws-io-optimized-v2",
					Version:              "7.10.1",
				},
			},
			expected: nil,
		},

		{
			name: "it should use state if both plan and state versions is or higher than 7.10.0 and the attribute state is not null`",
			args: args{
				attributeState: []string{"data_hot"},
				deploymentState: &deploymentv2.Deployment{
					DeploymentTemplateId: "aws-io-optimized-v2",
					Version:              "7.10.0",
				},
				deploymentPlan: deploymentv2.Deployment{
					DeploymentTemplateId: "aws-io-optimized-v2",
					Version:              "7.10.0",
				},
			},
			expected: []string{"data_hot"},
		},

		{
			name: "it should not use state if both plan and state versions is or higher than 7.10.0 and the attribute state is null`",
			args: args{
				attributeState: nil,
				deploymentState: &deploymentv2.Deployment{
					DeploymentTemplateId: "aws-io-optimized-v2",
					Version:              "7.10.0",
				},
				deploymentPlan: deploymentv2.Deployment{
					DeploymentTemplateId: "aws-io-optimized-v2",
					Version:              "7.10.0",
				},
			},
			expectedUnknown: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifier := v2.UseNodeRolesDefault()

			// attributeConfig value is not used in the plan modifer
			// it just should be known
			attributeConfigValue := attrValueFromGoTypeValue(t, []string{}, types.SetType{ElemType: types.StringType})

			attributeStateValue := attrValueFromGoTypeValue(t, tt.args.attributeState, types.SetType{ElemType: types.StringType})

			deploymentStateValue := tftypesValueFromGoTypeValue(t, tt.args.deploymentState, deploymentv2.DeploymentSchema().Type())

			deploymentPlanValue := tftypesValueFromGoTypeValue(t, tt.args.deploymentPlan, deploymentv2.DeploymentSchema().Type())

			req := tfsdk.ModifyAttributePlanRequest{
				AttributeConfig: attributeConfigValue,
				AttributeState:  attributeStateValue,
				State: tfsdk.State{
					Raw:    deploymentStateValue,
					Schema: deploymentv2.DeploymentSchema(),
				},
				Plan: tfsdk.Plan{
					Raw:    deploymentPlanValue,
					Schema: deploymentv2.DeploymentSchema(),
				},
			}

			// the default plan value is `Unknown` ("known after apply")
			// the plan modifier either keeps this value or uses the current state
			// if test doesn't specify plan value, let's use the default (`Unknown`) value that is used by TF during plan modifier execution
			attributePlanValue := unknownValueFromAttrType(t, types.SetType{ElemType: types.StringType})
			if tt.args.attributePlan != nil {
				attributePlanValue = attrValueFromGoTypeValue(t, tt.args.attributePlan, types.SetType{ElemType: types.StringType})
			}

			resp := tfsdk.ModifyAttributePlanResponse{AttributePlan: attributePlanValue}

			modifier.Modify(context.Background(), req, &resp)

			assert.Nil(t, resp.Diagnostics)

			if tt.expectedUnknown {
				assert.True(t, resp.AttributePlan.IsUnknown(), "attributePlan should be unknown")
				return
			}

			var attributePlan []string

			diags := tfsdk.ValueAs(context.Background(), resp.AttributePlan, &attributePlan)

			assert.Nil(t, diags)

			assert.Equal(t, tt.expected, attributePlan)
		})
	}
}
