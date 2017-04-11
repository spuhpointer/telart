#!/usr/bin/python
import sys
import os
import string
import time

from endurox.atmi import *
import RPi.GPIO as GPIO	 #import the GPIO library

#class Buzzer(object):
class server:
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

	def play(self):
		tune=4
		GPIO.setmode(GPIO.BCM)
		GPIO.setup(self.buzzer_pin, GPIO.OUT)
		x=0

		print("Playing tune ",tune)
		pitches=[1047, 988,659]
		duration=[0.1,0.1,0.2]
		for p in pitches:
			self.buzz(p, duration[x])	#feed the pitch and duration to the func$
			time.sleep(duration[x] *0.5)
			x+=1

	def RING(self, arg):
		tplog(log_debug, "connect to RING ... - starting to ring...")
		try:
			tplog(log_debug, "Starting... RunBuzzer")
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
			lastRcv = time.time()
			# protect us for 5 second non response.
			while time.time() - lastRcv < 5 :
				#evt, rec = tprecv(handle)

				#if rec is not None:

				tune=4
				GPIO.setmode(GPIO.BCM)
				GPIO.setup(self.buzzer_pin, GPIO.OUT)
				x=0

				print("Playing tune ",tune)
				pitches=[1047, 988,659, 1,1,1,1,1]
				duration=[0.1,0.1,0.2,0.1,0.1,0.1,0.1,0.1]
				for p in pitches:
					self.buzz(p, duration[x])	#feed the pitch and duration to the func$
					time.sleep(duration[x] *0.5)
					try:
						# flush the queues, if have msgs in...
						while True:
							evt, rec = tprecv(handle,TPNOBLOCK)
							tplog(log_debug, "Got Tick")
							lastRcv = time.time()
					except RuntimeError, e:
						exception = "got exception: %s" % e
						if exception.find("TPMINVAL") == -1:
							raise
						
					
		except RuntimeError, e:
			exception = "got exception: %s" % e
			tplog(log_error, exception)
			return TPFAIL
	
		except Exception, e:
			tb = traceback.format_exec()
			tplog(log_error,"got exception %s => %s" % (str(e), tb))
			return TPFAIL
	
	def init(self, arguments):
		# Setup GPIO
		GPIO.setmode(GPIO.BCM)	
		self.buzzer_pin = 22 #set to GPIO pin 22
		GPIO.setup(self.buzzer_pin, GPIO.IN)
		GPIO.setup(self.buzzer_pin, GPIO.OUT)
		print("buzzer ready")

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
	#global RingThread
	#global RunRing
	#RunRing = True
	#thread.start_new_thread( RunBuzzer, ("Thread-1", ) )
	#thread = Thread(target = RunBuzzer, args = (10, ))
	#thread.start()
	mainloop(sys.argv, srv, None)
	# Terminate ring thread
	#RingThread=False
	#thread.join()

# Local Variables: 
# mode:python 
# End: 
