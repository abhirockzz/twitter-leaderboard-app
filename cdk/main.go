package main

import (
	"fmt"
	"strconv"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsmemorydb"
	"github.com/aws/jsii-runtime-go"

	"github.com/aws/constructs-go/constructs/v10"
)

type CdkStackProps struct {
	awscdk.StackProps
}

func main() {
	app := awscdk.NewApp(nil)

	NewCdkStack1(app, "stack1", &CdkStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	NewCdkStack2(app, "stack2", &CdkStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	NewCdkStack3(app, "stack3", &CdkStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	app.Synth(nil)
}

const (
	memoryDBNodeType                  = "db.t4g.small"
	accessString                      = "on ~* &* +@all"
	numMemoryDBShards                 = 1
	numMemoryDBReplicaPerShard        = 1
	memoryDBDefaultParameterGroupName = "default.memorydb-redis6"
	memoryDBRedisEngineVersion        = "6.2"
	memoryDBRedisPort                 = 6379

	tweetIngestionFunctionName     = "tweet-ingest-function"
	hashtagLeaderboardFunctionName = "hashtag-leaderboard-function"

	tweetIngestionFunctionPath     = "../tweet-ingest/"
	hashtagLeaderboardFunctionPath = "../leaderboard-function/"
)

var vpc awsec2.Vpc

var memorydbCluster awsmemorydb.CfnCluster
var user awsmemorydb.CfnUser

var memorydbSecurityGroup awsec2.SecurityGroup
var twitterIngestFunctionSecurityGroup awsec2.SecurityGroup
var twitterLeaderboardFunctionSecurityGroup awsec2.SecurityGroup

func NewCdkStack1(scope constructs.Construct, id string, props *CdkStackProps) awscdk.Stack {

	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// vpc
	vpc = awsec2.NewVpc(stack, jsii.String("demo-vpc-481"), nil)

	// memorydb auth info
	authInfo := map[string]interface{}{"Type": "password", "Passwords": []string{getMemorydbPassword()}}

	// memorydb user and acl
	user = awsmemorydb.NewCfnUser(stack, jsii.String("demo-memorydb-user"), &awsmemorydb.CfnUserProps{UserName: jsii.String("demo-user-8"), AccessString: jsii.String(accessString), AuthenticationMode: authInfo})

	acl := awsmemorydb.NewCfnACL(stack, jsii.String("demo-memorydb-acl"), &awsmemorydb.CfnACLProps{AclName: jsii.String("demo-memorydb-acl-579"), UserNames: &[]*string{user.UserName()}})

	acl.AddDependsOn(user)

	// memory subnet group
	var subnetIDsForSubnetGroup []*string

	for _, sn := range *vpc.PrivateSubnets() {
		subnetIDsForSubnetGroup = append(subnetIDsForSubnetGroup, sn.SubnetId())
	}

	subnetGroup := awsmemorydb.NewCfnSubnetGroup(stack, jsii.String("demo-memorydb-subnetgroup"), &awsmemorydb.CfnSubnetGroupProps{SubnetGroupName: jsii.String("demo-memorydb-subnetgroup-729"), SubnetIds: &subnetIDsForSubnetGroup})

	// memory security group
	memorydbSecurityGroup = awsec2.NewSecurityGroup(stack, jsii.String("memorydb-demo-sg"), &awsec2.SecurityGroupProps{Vpc: vpc, SecurityGroupName: jsii.String("memorydb-demo-sg-235"), AllowAllOutbound: jsii.Bool(true)})

	// memory security cluster
	memorydbCluster = awsmemorydb.NewCfnCluster(stack, jsii.String("demo-memorydb-cluster"), &awsmemorydb.CfnClusterProps{ClusterName: jsii.String("demo-memorydb-cluster-362"), NodeType: jsii.String(memoryDBNodeType), AclName: acl.AclName(), NumShards: jsii.Number(numMemoryDBShards), EngineVersion: jsii.String(memoryDBRedisEngineVersion), Port: jsii.Number(memoryDBRedisPort), SubnetGroupName: subnetGroup.SubnetGroupName(), NumReplicasPerShard: jsii.Number(numMemoryDBReplicaPerShard), TlsEnabled: jsii.Bool(true), SecurityGroupIds: &[]*string{memorydbSecurityGroup.SecurityGroupId()}, ParameterGroupName: jsii.String(memoryDBDefaultParameterGroupName)})

	memorydbCluster.AddDependsOn(user)
	memorydbCluster.AddDependsOn(acl)
	memorydbCluster.AddDependsOn(subnetGroup)

	twitterIngestFunctionSecurityGroup = awsec2.NewSecurityGroup(stack, jsii.String("twitterIngestFunctionSecurityGroup"), &awsec2.SecurityGroupProps{Vpc: vpc, SecurityGroupName: jsii.String("twitterIngestFunctionSecurityGroup-905"), AllowAllOutbound: jsii.Bool(true)})

	twitterLeaderboardFunctionSecurityGroup = awsec2.NewSecurityGroup(stack, jsii.String("twitterLeaderboardFunctionSecurityGroup"), &awsec2.SecurityGroupProps{Vpc: vpc, SecurityGroupName: jsii.String("twitterLeaderboardFunctionSecurityGroup-619"), AllowAllOutbound: jsii.Bool(true)})

	memorydbSecurityGroup.AddIngressRule(awsec2.Peer_SecurityGroupId(twitterIngestFunctionSecurityGroup.SecurityGroupId(), nil), awsec2.Port_Tcp(jsii.Number(6379)), jsii.String("for tweets ingest lambda function to access memorydb"), jsii.Bool(false))

	memorydbSecurityGroup.AddIngressRule(awsec2.Peer_SecurityGroupId(twitterLeaderboardFunctionSecurityGroup.SecurityGroupId(), nil), awsec2.Port_Tcp(jsii.Number(6379)), jsii.String("for leaderboard lambda function to access memorydb"), jsii.Bool(false))

	return stack
}

func NewCdkStack2(scope constructs.Construct, id string, props *CdkStackProps) awscdk.Stack {

	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	memoryDBEndpointURL := fmt.Sprintf("%s:%s", *memorydbCluster.AttrClusterEndpointAddress(), strconv.Itoa(int(*memorydbCluster.Port())))

	// environment variable and sec group for Lambda function
	lambdaEnvVars := &map[string]*string{"MEMORYDB_ENDPOINT": jsii.String(memoryDBEndpointURL), "MEMORYDB_USER": user.UserName(), "MEMORYDB_PASSWORD": jsii.String(getMemorydbPassword()), "TWITTER_API_KEY": jsii.String(getTwitterAPIKey()), "TWITTER_API_SECRET": jsii.String(getTwitterAPISecret()), "TWITTER_ACCESS_TOKEN": jsii.String(getTwitterAccessToken()), "TWITTER_ACCESS_TOKEN_SECRET": jsii.String(getTwitterAccessTokenSecret())}

	//function is created with - env vars, vpc, subnet, sec group and deployed as docker image
	awslambda.NewDockerImageFunction(stack, jsii.String("lambda-memorydb-func"), &awslambda.DockerImageFunctionProps{FunctionName: jsii.String(tweetIngestionFunctionName), Environment: lambdaEnvVars, Timeout: awscdk.Duration_Seconds(jsii.Number(20)), Code: awslambda.DockerImageCode_FromImageAsset(jsii.String(tweetIngestionFunctionPath), nil), Vpc: vpc, VpcSubnets: &awsec2.SubnetSelection{Subnets: vpc.PrivateSubnets()}, SecurityGroups: &[]awsec2.ISecurityGroup{twitterIngestFunctionSecurityGroup}})

	return stack
}

func NewCdkStack3(scope constructs.Construct, id string, props *CdkStackProps) awscdk.Stack {

	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	memoryDBEndpointURL := fmt.Sprintf("%s:%s", *memorydbCluster.AttrClusterEndpointAddress(), strconv.Itoa(int(*memorydbCluster.Port())))

	// environment variable for Lambda function
	lambdaEnvVars := &map[string]*string{"MEMORYDB_ENDPOINT": jsii.String(memoryDBEndpointURL), "MEMORYDB_USERNAME": user.UserName(), "MEMORYDB_PASSWORD": jsii.String(getMemorydbPassword())}

	//create func with docker iamge along with vpc, subnet and sec group config for memorydb access
	function := awslambda.NewDockerImageFunction(stack, jsii.String("twitter-hashtag-leaderboard"), &awslambda.DockerImageFunctionProps{FunctionName: jsii.String(hashtagLeaderboardFunctionName), Environment: lambdaEnvVars, Code: awslambda.DockerImageCode_FromImageAsset(jsii.String(hashtagLeaderboardFunctionPath), nil), Timeout: awscdk.Duration_Seconds(jsii.Number(5)), Vpc: vpc, VpcSubnets: &awsec2.SubnetSelection{Subnets: vpc.PrivateSubnets()}, SecurityGroups: &[]awsec2.ISecurityGroup{twitterLeaderboardFunctionSecurityGroup}})

	funcURL := awslambda.NewFunctionUrl(stack, jsii.String("func-url"), &awslambda.FunctionUrlProps{AuthType: awslambda.FunctionUrlAuthType_NONE, Function: function})

	awscdk.NewCfnOutput(stack, jsii.String("Function URL"), &awscdk.CfnOutputProps{Value: funcURL.Url()})

	return stack
}

// env determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *awscdk.Environment {
	return nil
}
