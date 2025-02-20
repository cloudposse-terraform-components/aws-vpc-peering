package test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/cloudposse/test-helpers/pkg/atmos"
	helper "github.com/cloudposse/test-helpers/pkg/atmos/component-helper"
	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/stretchr/testify/assert"
)

type VpcPeering struct {
	AccepterAcceptStatus        string            `json:"accepter_accept_status"`
	AccepterConnectionID        string            `json:"accepter_connection_id"`
	AccepterSubnetRouteTableMap map[string]string `json:"accepter_subnet_route_table_map"`
	RequesterAcceptStatus       string            `json:"requester_accept_status"`
	RequesterConnectionID       string            `json:"requester_connection_id"`
}

type ComponentSuite struct {
	helper.TestSuite
}

func (s *ComponentSuite) TestBasic() {
	const component = "vpc-peering/basic"
	const stack = "default-test"
	const awsRegion = "us-east-2"

	vpcOptions := s.GetAtmosOptions("vpc/b", stack, nil)

	vpcId := atmos.Output(s.T(), vpcOptions, "vpc_id")

	inputs := map[string]interface{}{
		"accepter_vpc": map[string]interface{}{
			"id": vpcId,
		},
	}

	defer s.DestroyAtmosComponent(s.T(), component, stack, &inputs)
	options, _ := s.DeployAtmosComponent(s.T(), component, stack, &inputs)
	assert.NotNil(s.T(), options)

	var vpcPeering VpcPeering
	atmos.OutputStruct(s.T(), options, "vpc_peering", &vpcPeering)
	assert.Equal(s.T(), vpcPeering.AccepterAcceptStatus, "active")
	assert.Equal(s.T(), 4, len(vpcPeering.AccepterSubnetRouteTableMap))

	client := aws.NewEc2Client(s.T(), awsRegion)
	connections, err := client.DescribeVpcPeeringConnections(context.Background(), &ec2.DescribeVpcPeeringConnectionsInput{
		VpcPeeringConnectionIds: []string{vpcPeering.AccepterConnectionID},
	})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 1, len(connections.VpcPeeringConnections))
	connection := connections.VpcPeeringConnections[0]
	assert.EqualValues(s.T(), types.VpcPeeringConnectionStateReasonCodeActive, connection.Status.Code)
	assert.EqualValues(s.T(), vpcPeering.AccepterConnectionID, *connection.VpcPeeringConnectionId)

	s.DriftTest(component, stack, &inputs)
}

func (s *ComponentSuite) TestEnabledFlag() {
	const component = "vpc-peering/disabled"
	const stack = "default-test"
	const awsRegion = "us-east-2"

	vpcOptions := s.GetAtmosOptions("vpc/b", stack, nil)
	vpcId := atmos.Output(s.T(), vpcOptions, "vpc_id")

	inputs := map[string]interface{}{
		"accepter_vpc": map[string]interface{}{
			"id": vpcId,
		},
	}

	s.VerifyEnabledFlag(component, stack, &inputs)
}

func TestRunSuite(t *testing.T) {
	suite := new(ComponentSuite)
	suite.AddDependency(t, "vpc/a", "default-test", nil)
	suite.AddDependency(t, "vpc/b", "default-test", nil)

	helper.Run(t, suite)
}
