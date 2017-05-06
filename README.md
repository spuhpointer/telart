## Synopsis

Peer-to-peer, random dialer telephone system realization on [Enduro/X cluster middleware for C/C++/Go/Python](http://www.endurox.org). [Project description can be found here](https://publicwork.wordpress.com/2017/04/08/telephone-system-simulation-with-endurox-middleware/)
The project uses Enduro/X XATMI distributed processing and clustering to join in single cluster domain all involed nodes (in this
case Raspberry PI machines). Which offers services like PHONE, RING, PLAYBACK (busy, wait), MIC. The system reads the switch state
picked up, or hanged up the phone and does the random call to other nodes. If other node picks up the phone
the call is established and user can speak over.



## rc.local setup

$ cat /etc/rc.local 

```
#!/bin/sh -e
#
# rc.local
#
# This script is executed at the end of each multiuser runlevel.
# Make sure that the script will "exit 0" on success or any other
# value on error.
#
# In order to enable or disable this script just change the execution
# bits.
#
# By default this script does nothing.

# Print the IP address
_IP=$(hostname -I) || true
if [ "$_IP" ]; then
  printf "My IP address is %s\n" "$_IP"
fi

# Max Messages in Queue
echo 10000 > /proc/sys/fs/mqueue/msg_max

# Max message size (Currently Enduro/X supports only 32K as max)
echo 64000 > /proc/sys/fs/mqueue/msgsize_max

# Max number of queues for user
echo 10000 > /proc/sys/fs/mqueue/queues_max

# give some access to kmem
chmod g+rw /dev/gpiomem

echo "Before start" > /tmp/startup

# Start the phone app
su telart -c '/home/telart/telart/conf/TELART_START' >> /tmp/startup 2>&1

exit 0
```


