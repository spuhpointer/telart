#!/usr/bin/perl


#$ENV{"TELART_BRIDGE_1"} = 'AAA';


#print "export TELART_BRIDGE_1=AAAA\n";


#
# @(#) Script detects IP address, first priority 192.168.X.X
# Second priority 10.150.70.X
# Then configures the bridges
# The nodeid is detected from last ip addres digit.
# the other nodes are expected to be in same range.
# for example: 10.150.70.54
# Our nodeid: 4, others are in: 10.150.70.50..10.150.70.59
# or example: 192.168.0.5
# Our nodeid: 5, others are in: 192.168.0.0..192.168.0.9
#
# We (our bridge) are going into passive mode (wait for tcp call)
# and doing active-connect to other nodes.
#


#
# Read ip config
#
(@ifconfig=`ifconfig`) || die("Cannot call ifconfig!");

$infos = join(",", @ifconfig);

$infos =~ s/\R//g;

#print "echo Got ip infos: [$infos]\n";


################################################################################
# Extract our addresses
################################################################################

#IP 1
($ip) = ($infos =~ /inet addr\:192\.168\.0\.([0-9]+)/);

if ($ip eq "")
{
# IP 2 attempt
	($ip) = ($infos =~/inet addr\:10\.150\.70\.([0-9]+)/);
	
	if ($ip ne "")
	{
		$ip = "10.150.70.$ip";
	}
} 
else
{
	$ip = "192.168.0.$ip";
}

if ($ip eq "")
{
	die("Cannot detect IP configuration!");
}

################################################################################
# Detect segment...
################################################################################
$segment = $ip;
$ourNode = chop($segment);

# Get the our node id.

#
# have some echo in front so that bash can take it over!
#
print "echo Appliance IP: [$ip]\n";
print "echo Segment: [$segment]\n";
print "echo Our Node ID: [$ourNode]\n";
if ($ourNode == 0)
{
	die("Node 0 not supported - change ip please!");
}

print "export A_NODE=$ourNode\n";

################################################################################
# Build our bridge, the list of env vars:
#
# TELART_BRIDGE_1...TELART_BRIDGE_9
# if the index is the same as our node id, then setup it as active
# and listent to 0.0.0.0
# - even nodes are passive (do listen)
# - odd nodes are active (do connect)
#
################################################################################

$max_node = 9;
$con = 1;

################################################################################
# Generate outgoing connections
################################################################################
for ($i=1; $i<=$max_node; $i++)
{
	print "export A_BRI_$i\_MIN=0\n";
	print "export A_BRI_$i\_APPOPT=-\n";
	print "export A_BRI_$i\_SYSOPT=-\n";
}

################################################################################
# Generate outgoing connections
################################################################################
for ($i=$ourNode; $i<$max_node; $i++)
{
	$theirNode = $i+1;
	
	print "export A_BRI_$con\_SYSOPT=\"-e \${NDRX_APPHOME}/log/tpbridge_$theirNode.log -r\"\n";
	print "export A_BRI_$con\_APPOPT=\"-f -n$theirNode -r -i $segment$theirNode -p 2100$ourNode -tA -z30\"\n";
	print "export A_BRI_$con\_MIN=1\n";
	
	$con++;
}

################################################################################
# Generate incoming connections (wait for connection...)
################################################################################
for ($i=$ourNode; $i>1; $i--)
{
	$theirNode = $i-1;
	
	print "export A_BRI_$con\_SYSOPT=\"-e \${NDRX_APPHOME}/log/tpbridge_$theirNode.log -r\"\n";
	print "export A_BRI_$con\_APPOPT=\"-f -n$theirNode -r -i 0.0.0.0 -p 2100$theirNode -tP -z30\"\n";
	print "export A_BRI_$con\_MIN=1\n";
	
	$con++;
}

