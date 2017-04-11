#!/usr/bin/python
import time
import string
import sys
import types
import fileinput
import re
import os

import RPi.GPIO as GPIO
import time
GPIO.setmode(GPIO.BCM)

from endurox.atmi import *
from endurox.ubfbuffer import *


INPUT_PIN = 2

GPIO.setup(INPUT_PIN, GPIO.IN, pull_up_down=GPIO.PUD_DOWN)
presses = 1

svc = "PHONE%02d" % (tpgetnodeid())

#
# Send the event to phone service
#
def call(Cmd):
	global presses

	tplog(log_info, "Calling %s with command: %s " % (Cmd, svc))
	inp = UbfBuffer()
	inp['A_SRC_NODE'][0] = "%d" % tpgetnodeid()
	inp['A_CMD'][0] = "%s" % Cmd
	print inp
	res =  tpcall(svc, inp.as_dictionary(), TPNOTRAN)
	print res

#GPIO.add_event_detect(INPUT_PIN, GPIO.BOTH)

# reset channel..
#my_callback(1)

tplog(log_info, "Starting to scan...")
currentCmd = ""
lastVal = GPIO.input(INPUT_PIN)
tplog(log_info, "last val = %s"%lastVal)
same = 0
while True:
	
	try:
		# Software deboucner
		# 500ms needs to be same signal, if so
		# then issue the command
		time.sleep(0.1)

		if lastVal == GPIO.input(INPUT_PIN):
			same+=1
		else:
			lastVal = GPIO.input(INPUT_PIN)
			same = 0

		if same > 3 and lastVal==True and currentCmd!="H":
			currentCmd="H"
			call(currentCmd)
		elif same > 3 and lastVal==False and currentCmd!="P":
			currentCmd="P"
			call(currentCmd)
			
	except KeyboardInterrupt:
        	GPIO.cleanup()

