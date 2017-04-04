#include <string.h>
#include <stdio.h>
#include <stdlib.h>
#include <memory.h>
#include <math.h>
#include <errno.h>
#include <signal.h>
#include <sys/resource.h>

#include <ndebug.h>
#include <atmi.h>


#include <ubf.h>
#include <Exfields.h>
#include <telart.fd.h>
/*---------------------------Externs------------------------------------*/
/*---------------------------Macros-------------------------------------*/

#ifndef SUCCEED
#define SUCCEED			0
#endif

#ifndef	FAIL
#define FAIL			-1
#endif


#define PROGSECTION		"playbacksv"	/* configuration section */
#define CONFIG_SERVER		"@CCONF"
#define KEY_VAL_BUFFSZ		1024



#define STDIN_PIPE  0

/*---------------------------Enums--------------------------------------*/
/*---------------------------Typedefs-----------------------------------*/
/*---------------------------Globals------------------------------------*/
/*---------------------------Statics------------------------------------*/

static char M_command[PATH_MAX]={0};
static char M_busy[PATH_MAX]={0};
static char M_wait[PATH_MAX]={0};

static char *M_play; /* what to play... */

/*---------------------------Prototypes---------------------------------*/

/* TODO: Have a sig-child handler... */


/**
 * Run playback
 */
static pid_t run_play(void)
{
	pid_t child_pid;
	
	if (0==(child_pid=fork()))
	{
		/* this is child process... */
		char *argv[]={ (char *) M_command, M_play, 0};

		TP_LOG(log_info, "Executing: [%s]", argv[0]);		
		
		if (SUCCEED!=execv(argv[0], argv))
		{
			TP_LOG(log_error, "Failed to exec: %s", strerror(errno));
			exit(FAIL);
		}
	}
	
	return child_pid;
}
/**
 * Service entry
 * @return SUCCEED/FAIL
 */
void PLAYBACK (TPSVCINFO *p_svc)
{
	int ret = SUCCEED;
	FILE *fp=NULL;
	char buf[32000];
	char cmd;
	long revent=0;
	int was_sig = 1;
	pid_t child_pid, sigc;
	int stat_loc;
	struct rusage rusage;
	
	BFLDLEN rd;

	UBFH *p_ub = (UBFH *)p_svc->data;

	tplogprintubf(log_info, "Got request", p_ub);
	
	/* get the command */
	
	if (SUCCEED!=Bget(p_ub, A_CMD, 0, &cmd, 0L))
	{
		TP_LOG(log_error, "Failed to get A_CMD: %s", Bstrerror(Berror));
		ret=FAIL;
		goto out;
	}
	
	TP_LOG(log_info, "Got command: %c", cmd);
	
	switch (cmd)
	{
		case 'B':
			M_play = M_busy;
			break;
		case 'W':
			M_play = M_wait;
			break;
			
		default:
			TP_LOG(log_error, "Invalid command received %c!", cmd);
			ret=FAIL;
			goto out;
			break;
	}
	
        /* Check the process name in output... */
        while (SUCCEED==ret)
        {
		/* loop the player... */
		if (was_sig && FAIL==(child_pid = run_play()))
		{
			TP_LOG(log_info, "Failed to fork: [%s]", strerror(errno));
			ret=FAIL;
			goto out;
		}
		
		was_sig = 0;
		
		/* to play back at some interval we need to receive messages... */
		/* Receive command stop play, or timeout - terminat the child...
		 * thus terminate the playback...
		 */
		if (SUCCEED!=tprecv(p_svc->cd, (char **)&p_ub, 0L, 0L, &revent))
		{
			TP_LOG(log_error, "tpsend failed: %s", tpstrerror(tperrno));
			if (revent!=0)
			{
				switch (revent)
				{
					case TPEV_DISCONIMM:
						TP_LOG(log_error, "got: TPEV_DISCONIMM "
							"- SUCCEED");
						goto out; /* SUCCEED... */
						break;
					case TPEV_SVCERR:
						TP_LOG(log_error, "got: TPEV_SVCERR");
						break;
					case TPEV_SVCFAIL:
						TP_LOG(log_error, "got: TPEV_SVCFAIL");
						break;
				}
			}
			
			ret=FAIL;
			goto out;
		}
		
		/* check the child exit... */
		
		while ((sigc=wait3(&stat_loc, WNOHANG|WUNTRACED, &rusage)) > 0)
		{
			TP_LOG(log_info, "Got SIGCHLD...")
			was_sig = 1;
			if (sigc == child_pid)
			{
				child_pid=0;
			}
		}
        }
        
out:
	if (child_pid>0)
	{
		kill(child_pid, SIGINT);
	}

	tpreturn(  ret==SUCCEED?TPSUCCESS:TPFAIL,
		0L,
		(char *)p_ub,
		0L,
		0L);
}

/**
 * Initialize the application
 * @param argc	argument count
 * @param argv	argument values
 * @return SUCCEED/FAIL
 */
int init(int argc, char** argv)
{
	int ret = SUCCEED;

	UBFH *p_ub = NULL;
	char config_tag[128];
	long rsplen;
	BFLDLEN sz;
	int occ;
	int i;
	char key[KEY_VAL_BUFFSZ]={0};
	char val[KEY_VAL_BUFFSZ]={0};
	char *cctag;
	char svcnm[MAXTIDENT+1];
	
	
	TP_LOG(log_info, "Initializing...");

        /* signal(SIGCHLD, SIG_IGN); */

	if (SUCCEED!=tpinit(NULL))
	{
		TP_LOG(log_error, "Failed to Initialize: %s", 
			tpstrerror(tperrno));
		ret = FAIL;
		goto out;
	}
	

	/* Download configuration */
	
	if (NULL==(p_ub = (UBFH *)tpalloc("UBF", NULL, 1024)))
	{
		TP_LOG(log_error, "Failed to alloc:%s",  tpstrerror(tperrno));
		ret=FAIL;
		goto out;
	}
	
	cctag = getenv("NDRX_CCTAG");
	if (NULL!=cctag)
	{
		snprintf(config_tag, sizeof(config_tag), "%s/%s", 
			PROGSECTION, cctag);
	}
	else
	{
		/* NO subsection configured */
		snprintf(config_tag, sizeof(config_tag), "%s", PROGSECTION);
	}
	
	if ( (SUCCEED!=Bchg(p_ub, EX_CC_CMD, 0, "g", 0L))
		|| (SUCCEED!=Bchg(p_ub, EX_CC_LOOKUPSECTION, 0,  config_tag, 0L)))
	{
		TP_LOG(log_error, "Failed to set EX_CC_CMD/EX_CC_LOOKUPSECTION: %s", 
			Bstrerror(Berror));
		ret = FAIL;
		goto out;
	}
	
	if (FAIL==tpcall(CONFIG_SERVER, (char *)p_ub, 0L, (char **)&p_ub, &rsplen, TPNOTIME))
	{
		TP_LOG(log_error, "Failed to call %s: %s", 
			 CONFIG_SERVER,tpstrerror(tperrno));
		ret=FAIL;
		goto out;
	}
	
	
	tplogprintubf(log_info, "Got configuration", p_ub);

	occ = Boccur(p_ub, EX_CC_KEY);
	
	for (i=0; i<occ; i++)
	{
		sz = sizeof(key);
		if (SUCCEED!=CBget(p_ub, EX_CC_KEY, i, key, &sz, BFLD_STRING))
		{
			TP_LOG(log_error, "Failed to get EX_CC_KEY[%d]: %s", i,
			     Bstrerror(Berror));
			ret=FAIL;
			goto out;
		}
		
		sz = sizeof(val);
		if (SUCCEED!=CBget(p_ub, EX_CC_VALUE, i, val, &sz, BFLD_STRING))
		{
			TP_LOG(log_error, "Failed to get EX_CC_VALUE[%d]: %s", i,
			     Bstrerror(Berror));
			ret=FAIL;
			goto out;
		}
		
		TP_LOG(log_debug, "Got key: [%s] = [%s]",
			key, val);
		
		if (0==strcmp(key, "command"))
		{
			TP_LOG(log_debug, "Got command: [%s]", val);
			strncpy((char *)M_command, val, sizeof(M_command));
			M_command[sizeof(M_command)-1] = 0;
		}
		else if (0==strcmp(key, "busy"))
		{
			TP_LOG(log_debug, "Got busy: [%s]", val);
			
			strncpy((char *)M_busy, val, sizeof(M_busy));
			M_busy[sizeof(M_busy)-1] = 0;
		}
		else if (0==strcmp(key, "wait"))
		{
			TP_LOG(log_debug, "Got wait: [%s]", val);
			
			strncpy((char *)M_wait, val, sizeof(M_wait));
			M_busy[sizeof(M_wait)-1] = 0;
		}
		else
		{
			TP_LOG(log_debug, "Unknown setting [%s] - ignoring...",
				key
			);
		}
	}
	
	if (!M_command[0])
	{
		TP_LOG(log_error, "Missing 'command' argument!");
		ret=FAIL;
		goto out;
	}
	
	
	if (!M_busy[0])
	{
		TP_LOG(log_error, "Missing 'busy' argument!");
		ret=FAIL;
		goto out;
	}
	
	if (!M_wait[0])
	{
		TP_LOG(log_error, "Missing 'wait' argument!");
		ret=FAIL;
		goto out;
	}
	
	/* Advertise our service according to our cluster node id */
	sprintf(svcnm, "PLAYBACK%02ld", tpgetnodeid());
	if (SUCCEED!=tpadvertise(svcnm, PLAYBACK))
	{
		TP_LOG(log_error, "Failed to initialize PLAYBACK!");
		ret=FAIL;
		goto out;
	}	
	
out:

	if (NULL!=p_ub)
	{
		tpfree((char *)p_ub);
	}
	
	return ret;
}

/**
 * Terminate the application
 */
void uninit(void)
{
	TP_LOG(log_info, "Uninitializing...");
}

/**
 * Server program main entry
 * @param argc	argument count
 * @param argv	argument values
 * @return SUCCEED/FAIL
 */
int main(int argc, char** argv)
{
	/* Launch the Enduro/x thread */
	return ndrx_main_integra(argc, argv, init, uninit, 0);
}

