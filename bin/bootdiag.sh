!#/bin/bash

# Diag buzzer on
cat << EOF | ud
SRVCNM	PHONE0$A_NODE
A_CMD	D
A_SRC_NODE	$A_NODE
EOF

sleep 1

# diag buzzer off
cat << EOF | ud
SRVCNM	PHONE0$A_NODE
A_CMD	d
A_SRC_NODE	$A_NODE
EOF

sleep 99999999
