#!/usr/bin/python
import time
import string
import sys
import types
import fileinput
import re
import os

# use the next line if you use /WS client connections
# remember to set WSNADDR for the client and to add a WSL to the 
# ubbconfig

# from endurox.atmiws import *

from endurox.atmi import *
from endurox.ubfbuffer import *


svc = "PHONE%02d" % (tpgetnodeid())

tplog(log_debug, "Doing %s call " % svc)

inp = UbfBuffer()
inp['A_SRC_NODE'][0] = tpgetnodeid()
inp['A_CMD'][0] = "P"

print inp

res =  tpcall(svc, inp.as_dictionary(), TPNOTRAN)

print res


