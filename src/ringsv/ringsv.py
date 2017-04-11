#!/usr/bin/python
import sys
import os
import string
import thread
from threading import Thread
from endurox.atmi import *

import RPi.GPIO as GPIO	 #import the GPIO library
import time							 #import the time library

RunRing=False
RingThread=True

class Buzzer(object):
	def __init__(self):
		GPIO.setmode(GPIO.BCM)	
		self.buzzer_pin = 22 #set to GPIO pin 22
		GPIO.setup(self.buzzer_pin, GPIO.IN)
		GPIO.setup(self.buzzer_pin, GPIO.OUT)
		print("buzzer ready")

	def __del__(self):
		class_name = self.__class__.__name__
		print (class_name, "finished")

	def buzz(self,pitch, duration):	 #create the function buzz and feed it the pitch and duration)
 
		if(pitch==0):
			time.sleep(duration)
			return
		period = 1.0 / pitch		 #in physics, the period (sec/cyc) is the inverse of the frequency (cyc/sec)
		delay = period / 2		 #calcuate the time for half of the wave	
		cycles = int(duration * pitch)	 #the number of waves to produce is the duration times the frequency

		for i in range(cycles):		#start a loop from 0 to the variable cycles calculated above
			GPIO.output(self.buzzer_pin, True)	 #set pin 18 to high
			time.sleep(delay)		#wait with pin 18 high
			GPIO.output(self.buzzer_pin, False)		#set pin 18 to low
			time.sleep(delay)		#wait with pin 18 low

	def play(self, tune):
		GPIO.setmode(GPIO.BCM)
		GPIO.setup(self.buzzer_pin, GPIO.OUT)
		x=0

		print("Playing tune ",tune)
		if(tune==1):
			pitches=[262,294,330,349,392,440,494,523, 587, 659,698,784,880,988,1047]
			duration=0.1
			for p in pitches:
				self.buzz(p, duration)	#feed the pitch and duration to the function, buzz
				time.sleep(duration *0.5)
			for p in reversed(pitches):
				self.buzz(p, duration)
				time.sleep(duration *0.5)

		elif(tune==4):
			pitches=[1047, 988,659]
			duration=[0.1,0.1,0.2]
			for p in pitches:
				self.buzz(p, duration[x])	#feed the pitch and duration to the func$

				if (RunRing==False):
					break

				time.sleep(duration[x] *0.5)
				x+=1
	#GPIO.setup(self.buzzer_pin, GPIO.IN)

def RunBuzzer(threadName):
	#a = input("Enter Tune number 1-5:")
	tplog(log_debug, "Into RunBuzzer")
	buzzer = Buzzer()
	while RingThread:
		while RunRing:
			buzzer.play(int(4))
		# Have some sleep
		time.sleep(0.1)

class server:
	def RING(self, arg):
		tplog(log_debug, "connect to RING ... - starting to ring...")
		RunRing = True
		try:
			tplog(log_debug, "Starting... RunBuzzer")
			#thread.start_new_thread( RunBuzzer, ("Thread-1", ) )
			
		except:
			log(log_error, "Failed to start Buzzer");

		try:
			handle = self.cd
			tplog(log_debug, "   cd = %i"  % self.cd)
                except:
			tplog(log_debug, "no cd given")

		try:
			tplog(log_debug, "   arg = %s" % arg)
		except:
			tplog(log_debug, "no arg given")


		try:
			tplog(log_debug, "   name = %s" % self.name)
		except:
			tplog(log_debug, "no name given")


		try:
			while 1:
				evt, rec = tprecv(handle)
		      
				# Having some issues with data buffer
				# the rec needs to be converted to UBF...
				# but for now it is not signficant
				# we just need a tick for a ring...
				tplog(log_debug,"Ring tick received... ")
		except RuntimeError, e:
			exception = "got exception: %s" % e
			tplog(log_error, exception)
			RunRing = False
			return TPFAIL
	
		except:
			tb = traceback.format_exec()
			#tplog(log_error,"got exception => %s" % tb)
			RunRing = False
			return TPFAIL
	
	def init(self, arguments):
                svc = "RING%02d" % (tpgetnodeid())
		tpadvertise(svc, "RING");
		
	def cleanup(self):
		userlog("cleanup in recv_py called!")


srv = server()

#srv.RING(1)

def exithandler():
	print "Ring service terminating..."
sys.exitfunc = exithandler

if __name__ == '__main__': 
	thread = Thread(target = RunBuzzer, args = (10, ))
	thread.start()
	mainloop(sys.argv, srv, None)
	# Terminate ring thread
	RingThread=False
	thread.join()

# Local Variables: 
# mode:python 
# End: 
