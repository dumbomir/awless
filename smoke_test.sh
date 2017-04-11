#!/usr/bin/env bash
set -e

TMP_FILE=./create-basic-infra.awl
TMP_USERDATA_FILE=./tmp-user-data.sh

/bin/cat > $TMP_USERDATA_FILE <<EOF
#!/bin/bash
echo "success" > /tmp/awless-ssh-userdata-success.txt
EOF

BIN=./awless

echo "building awless"
go build

echo "flushing awless logs..."
$BIN log --delete

ORIG_REGION=`$BIN config get aws.region`
ORIG_IMAGE=`$BIN config get instance.image`

REGION="us-west-1"
AMI="ami-165a0876"

echo "Setting region $REGION, ami $AMI"
$BIN config set aws.region $REGION
$BIN config set instance.image $AMI

SUFFIX=integ-test-`date +%s`
INSTANCE_NAME=inst-$SUFFIX
VPC_NAME=vpc-$SUFFIX
SUBNET_NAME=subnet-$SUFFIX
KEY_NAME=awless-integ-test-key

/bin/cat > $TMP_FILE <<EOF
testvpc = create vpc cidr={vpc-cidr} name=$VPC_NAME
testsubnet = create subnet cidr={sub-cidr} vpc=\$testvpc name=$SUBNET_NAME
gateway = create internetgateway
attach internetgateway id=\$gateway vpc=\$testvpc
update subnet id=\$testsubnet public=true
rtable = create routetable vpc=\$testvpc
attach routetable id=\$rtable subnet=\$testsubnet
create route cidr=0.0.0.0/0 gateway=\$gateway table=\$rtable
sgroup = create securitygroup vpc=\$testvpc description="authorize SSH from the Internet" name=ssh-from-internet
update securitygroup id=\$sgroup inbound=authorize protocol=tcp cidr=0.0.0.0/0 portrange=22
testkeypair = create keypair name=$KEY_NAME
testinstance = create instance subnet=\$testsubnet image={instance.image} type=t2.nano count={instance.count} key=\$testkeypair name=$INSTANCE_NAME userdata=$TMP_USERDATA_FILE group=\$sgroup
create tag resource=\$testinstance key=Env value=Testing
EOF

$BIN run ./$TMP_FILE vpc-cidr=10.0.0.0/24 sub-cidr=10.0.0.0/25 -e -f

ALIAS="\@$INSTANCE_NAME"
eval "$BIN check instance id=$ALIAS state=running timeout=20 -f"

echo "Instance is running. Waiting 20s for system boot"
sleep 20 

SSH_CONNECT=`$BIN ssh $INSTANCE_NAME --print-cli`
echo "Connecting to instance with $SSH_CONNECT"
RESULT=`$SSH_CONNECT -o StrictHostKeychecking=no 'cat /tmp/awless-ssh-userdata-success.txt'`

if [ "$RESULT" != "success" ]; then
	echo "FAIL to read correct token in remote file after ssh to instance"
	exit -1
fi

echo "Reading token in remote file on instance with success"

REVERT_ID=`$BIN log | grep RevertID | cut -d , -f2 | cut -d : -f2`
$BIN revert $REVERT_ID -e -f

echo "Clean up and reverting back to region '$ORIG_REGION' and ami '$ORIG_IMAGE'"

$BIN config set aws.region $ORIG_REGION
$BIN config set instance.image $ORIG_IMAGE

rm $TMP_FILE $TMP_USERDATA_FILE
rm -f ~/.awless/keys/$KEY_NAME.pem
rm $BIN
