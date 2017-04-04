#include <string.h>
#include <stdio.h>
#include <stdlib.h>
#include <memory.h>
#include <math.h>
#include <errno.h>
#include <signal.h>

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


#define PROGSECTION		"liveplaysv"	/* configuration section */
#define CONFIG_SERVER		"@CCONF"
#define KEY_VAL_BUFFSZ		1024



#define STDIN_PIPE  0

/*---------------------------Enums--------------------------------------*/
/*---------------------------Typedefs-----------------------------------*/
/*---------------------------Globals------------------------------------*/
/*---------------------------Statics------------------------------------*/

static char M_command[PATH_MAX]={0};
/*---------------------------Prototypes---------------------------------*/

/* TODO: Have a sig-child handler... */

/**
 * Service entry
 * @return SUCCEED/FAIL
 */
void LIVEPLAY (TPSVCINFO *p_svc)
{
	int ret = SUCCEED;
	FILE *fp=NULL;
	char buf[32000];
	char cmd[256];
	long revent=0;
	int child_stdin_pipe[2];
	pid_t child_pid;
	
	BFLDLEN rd;

	UBFH *p_ub = (UBFH *)p_svc->data;

	tplogprintubf(log_info, "Got request", p_ub);
	
	if (SUCCEED!=pipe(child_stdin_pipe))
	{
		TP_LOG(log_error, "Failed to pipe: %s", strerror(errno));
		ret=FAIL;
		goto out;
	}
	
	if (0==(child_pid=fork()))
	{
		/* this is child process... */
		char *argv[]={ (char *) M_command, "-", 0};

		/* char *argv[]={ "hexdump", "-o", 0}; */
		
		TP_LOG(log_info, "Executing: [%s]", argv[0]);
		
		/* copy stdin to read end of stdin pipe */
		/* dup2(0, child_stdin_pipe[0]); */
		
		close(child_stdin_pipe[1]);
		
		dup2(child_stdin_pipe[0], STDIN_FILENO);
		
		if (SUCCEED!=execv(argv[0], argv))
		{
			TP_LOG(log_error, "Failed to exec: %s", strerror(errno));
			exit(FAIL);
		}
		
#if 0
		while (FAIL!=(rd=read(child_stdin_pipe[0], buf, sizeof(buf))))
		{
			TP_DUMP(log_debug, "Got audio block", buf, rd);
		}
		  
		TP_LOG(log_error, "Failed to read: %s", strerror(errno));
#endif
	}
	
	if (FAIL==child_pid)
	{
		TP_LOG(log_info, "Failed to fork: [%s]", strerror(errno));
		ret=FAIL;
		goto out;
	}
	
	/* this is parent... */
	close(child_stdin_pipe[0]); /* not used by parent */

        /* Check the process name in output... */
        while (SUCCEED==ret)
        {
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
		
		/* send the data to conversational service */
		rd = sizeof(buf);
		if (SUCCEED!=Bget(p_ub, A_DATA, 0, buf, &rd))
		{
			TP_LOG(log_error, "Failed to get A_DATA: %s", 
			       Bstrerror(Berror));
			ret=FAIL;
			goto out;
		}
		
                TP_DUMP(log_debug, "Recevied audio block", buf, rd);
		
		if (FAIL==write(child_stdin_pipe[1], buf, rd))
		{
			TP_LOG(log_error, "Failed to write to aplay: %s",
			       strerror(errno));
			ret=FAIL;
			goto out;
		}
        }
        
out:

	close(child_stdin_pipe[1]);
	kill(child_pid, SIGINT);

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
        signal(SIGCHLD, SIG_IGN);

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
		else if (0==strcmp(key, "someparam2"))
		{
			TP_LOG(log_debug, "Got param2: [%s]", val);
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
	
	
	/* Advertise our service according to our cluster node id */
	sprintf(svcnm, "LIVEPLAY%02ld", tpgetnodeid());
	if (SUCCEED!=tpadvertise(svcnm, LIVEPLAY))
	{
		TP_LOG(log_error, "Failed to initialize LIVEPLAY!");
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

