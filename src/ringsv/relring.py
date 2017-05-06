#!/usr/bin/python
import sys
import os
import string
import time

from endurox.atmi import *
import RPi.GPIO as GPIO	 #import the GPIO library

class server:
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

			o=0.03 # contact on seconds
			w=0.01 # wait seconds (between ticks)
			w2=0.1 # Wait at end (sequence 2)
			# protect us for 5 second non response.
			while time.time() - lastRcv < 5 :

				tplog(log_debug, "About to run bell sequence")
				pitches=[-1,0,1,0,-1,0,1,0,-1,0,1,0,-1,0,1,0,-1,0,1,0,-1,0,1,0,-1,0,1,0,-1,0,1,0,-1,0,1,0,-1,0,-1,0,1,0,-1,0,1,0,-1,0,1,2,2,2,2,2]
				for p in pitches:

					if p == -1:
						# Safety net
						GPIO.output(self.buzzer_pin3, False)
						GPIO.output(self.buzzer_pin4, False)

						GPIO.output(self.buzzer_pin1, True)
						GPIO.output(self.buzzer_pin2, True)
						time.sleep(o)
					elif p == 1:
						GPIO.output(self.buzzer_pin1, False)
						GPIO.output(self.buzzer_pin2, False)
						
						GPIO.output(self.buzzer_pin3, True)
						GPIO.output(self.buzzer_pin4, True)
						time.sleep(o)
					else:
						GPIO.output(self.buzzer_pin3, False)
						GPIO.output(self.buzzer_pin4, False)

						GPIO.output(self.buzzer_pin1, False)
						GPIO.output(self.buzzer_pin2, False)
						if p == 2:
							time.sleep(w2)
						else:
							time.sleep(w)
						
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
			tpdiscon(handle)
			return TPFAIL
						
					
		except RuntimeError, e:
			exception = "got exception: %s" % e
			tplog(log_error, exception)
			return TPFAIL
	
		except Exception, e:
			tb = traceback.format_exec()
			tplog(log_error,"got exception %s => %s" % (str(e), tb))
			return TPFAIL
		finally:
			tplog(log_info, "Finally down")
			GPIO.output(self.buzzer_pin1, False)
			GPIO.output(self.buzzer_pin2, False)
			GPIO.output(self.buzzer_pin3, False)
			GPIO.output(self.buzzer_pin4, False)
			
	
	def init(self, arguments):
		# Setup GPIO
		GPIO.setmode(GPIO.BCM)	
		self.buzzer_pin1 = 14
		self.buzzer_pin2 = 15
		self.buzzer_pin3 = 18
		self.buzzer_pin4 = 23

		GPIO.setup(self.buzzer_pin1, GPIO.IN)
		GPIO.setup(self.buzzer_pin1, GPIO.OUT)

		GPIO.setup(self.buzzer_pin2, GPIO.IN)
		GPIO.setup(self.buzzer_pin2, GPIO.OUT)

		GPIO.setup(self.buzzer_pin3, GPIO.IN)
		GPIO.setup(self.buzzer_pin3, GPIO.OUT)

		GPIO.setup(self.buzzer_pin4, GPIO.IN)
		GPIO.setup(self.buzzer_pin4, GPIO.OUT)

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
