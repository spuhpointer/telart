#!/usr/bin/python
import sys
import os
import string
from endurox.atmi import *

class server:
	def RING(self, arg):
		tplog(log_debug, "connect to RING ...")
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
			return TPFAIL
	
		except:
			tb = traceback.format_exec()
			#tplog(log_error,"got exception => %s" % tb)
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
	mainloop(sys.argv, srv, None)

# Local Variables: 
# mode:python 
# End: 
