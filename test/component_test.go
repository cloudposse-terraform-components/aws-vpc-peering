package test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/cloudposse/test-helpers/pkg/atmos"
	helper "github.com/cloudposse/test-helpers/pkg/atmos/aws-component-helper"
	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/stretchr/testify/assert"
)

func TestComponent(t *testing.T) {
	// Define the AWS region to use for the tests
	awsRegion := "us-east-2"

	// Initialize the test fixture
	fixture := helper.NewFixture(t, "../", awsRegion, "test/fixtures")

	// Ensure teardown is executed after the test
	defer fixture.TearDown()
	fixture.SetUp(&atmos.Options{})

	// Define the test suite
	fixture.Suite("default", func(t *testing.T, suite *helper.Suite) {
		// Test phase: Validate the functionality of the ALB component
		suite.Test(t, "basic", func(t *testing.T, atm *helper.Atmos) {
			defer atm.GetAndDestroy("vpc/a", "default-test", map[string]interface{}{})
			atm.GetAndDeploy("vpc/a", "default-test", map[string]interface{}{})

			defer atm.GetAndDestroy("vpc/b", "default-test", map[string]interface{}{})
			vpcComponent := atm.GetAndDeploy("vpc/b", "default-test", map[string]interface{}{})

			vpcId := atm.Output(vpcComponent, "vpc_id")

			inputs := map[string]interface{}{
				"accepter_vpc": map[string]interface{}{
					"id": vpcId,
				},
			}

			defer atm.GetAndDestroy("vpc-peering/basic", "default-test", inputs)
			component := atm.GetAndDeploy("vpc-peering/basic", "default-test", inputs)
			assert.NotNil(t, component)

			type VpcPeering struct {
				AccepterAcceptStatus        string            `json:"accepter_accept_status"`
				AccepterConnectionID        string            `json:"accepter_connection_id"`
				AccepterSubnetRouteTableMap map[string]string `json:"accepter_subnet_route_table_map"`
				RequesterAcceptStatus       string            `json:"requester_accept_status"`
				RequesterConnectionID       string            `json:"requester_connection_id"`
			}
			var vpcPeering VpcPeering
			atm.OutputStruct(component, "vpc_peering", &vpcPeering)
			assert.Equal(t, vpcPeering.AccepterAcceptStatus, "active")
			assert.Equal(t, vpcPeering.RequesterAcceptStatus, "active")
			assert.Equal(t, 4, len(vpcPeering.AccepterSubnetRouteTableMap))

			client := aws.NewEc2Client(t, awsRegion)
			connections, err := client.DescribeVpcPeeringConnections(context.Background(), &ec2.DescribeVpcPeeringConnectionsInput{
				VpcPeeringConnectionIds: []string{vpcPeering.AccepterConnectionID},
			})
			assert.NoError(t, err)
			assert.Equal(t, 1, len(connections.VpcPeeringConnections))
			connection := connections.VpcPeeringConnections[0]
			assert.Equal(t, string(types.VpcPeeringConnectionStateReasonCodeActive), string(connection.Status.Code))
			assert.Equal(t, string(types.VpcPeeringConnectionStateReasonCodeActive), string(connection.Status.Code))
			assert.Equal(t, string(vpcPeering.AccepterConnectionID), string(*connection.VpcPeeringConnectionId))
		})
	})
}
