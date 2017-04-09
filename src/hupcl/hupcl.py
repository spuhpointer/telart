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


GPIO.setup(2, GPIO.IN, pull_up_down=GPIO.PUD_DOWN)
presses = 1

svc = "PHONE%02d" % (tpgetnodeid())

#
# Send the event to phone service
#
def my_callback(channel):
	global presses
	print "callback called " + str(presses) + " times"

	
	Cmd = ""
	if GPIO.input(2):
		print "RISING"
		Cmd = "H"
	else:
		Cmd = "P"

	inp = UbfBuffer()
	inp['A_SRC_NODE'][0] = "%d" % tpgetnodeid()
	inp['A_CMD'][0] = "%s" % Cmd
	print inp
	res =  tpcall(svc, inp.as_dictionary(), TPNOTRAN)
	print res
	presses += 1

GPIO.add_event_detect(2, GPIO.BOTH, callback=my_callback, bouncetime=500)

# reset channel..
my_callback(1)

tplog(log_info, "Waiting")
while True:
	try:
		time.sleep(5)
	except KeyboardInterrupt:
        	GPIO.cleanup()

