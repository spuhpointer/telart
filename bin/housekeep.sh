#!/bin/bash

#
# @(#) Lets clean up logs every 4 hour
#

while [[ 1 == 1 ]]; do
        truncate --size 0 /home/telart/telart/log/*
        # 60 * 60 * 4 
        sleep 14400
done
