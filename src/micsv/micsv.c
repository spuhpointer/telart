#include <string.h>
#include <stdio.h>
#include <stdlib.h>
#include <memory.h>
#include <math.h>
#include <errno.h>
#include <signal.h>
#include <sys/stat.h>
#include <sys/types.h>
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


#define PROGSECTION		"micsv"	/* configuration section */
#define CONFIG_SERVER		"@CCONF"
#define KEY_VAL_BUFFSZ		1024


/*---------------------------Enums--------------------------------------*/
/*---------------------------Typedefs-----------------------------------*/
/*---------------------------Globals------------------------------------*/
/*---------------------------Statics------------------------------------*/
/*---------------------------Prototypes---------------------------------*/


/**
 * Service entry
 * @return SUCCEED/FAIL
 */
void MIC (TPSVCINFO *p_svc)
{
	int ret = SUCCEED;
	FILE *fp=NULL;
	char buf[32000];
	char cmd[256];
	long revent=0;
	
	size_t rd;

	UBFH *p_ub = (UBFH *)p_svc->data;

	tplogprintubf(log_info, "Got request", p_ub);
	
        sprintf(cmd, "arecord -r2000 --buffer-time=500 -f cd -t wav");
        
        TP_LOG(log_info, "Executing: [%s]", cmd);
        
        
        if (NULL==(fp = popen(cmd, "r")))
        {
                TP_LOG(log_error, "Failed to open [%s]: %s",
                        cmd, strerror(errno));
                ret=FAIL;
                goto out;
        }

        /* Check the process name in output... */
        while ((rd=read(fileno(fp), buf, sizeof(buf))) > 0)
        {
                TP_DUMP(6, "Read audio data block", buf, rd);
                TP_LOG(log_info, "Read audio data block %d", rd);
		
		if (NULL==(p_ub=(UBFH *)tprealloc((char *)p_ub, sizeof(buf)+1024)))
		{
			TP_LOG(log_error, "Failed to realloc: %s", 
			       tpstrerror(tperrno));
			ret=FAIL;
			goto out;
		}
		
		/* send the data to conversational service */
		
		if (SUCCEED!=Bchg(p_ub, A_DATA, 0, buf, (int)rd))
		{
			TP_LOG(log_error, "Failed to set A_DATA: %s", 
			       Bstrerror(Berror));
			ret=FAIL;
			goto out;
		}
		
		if (SUCCEED!=tpsend(p_svc->cd, (char *)p_ub, 0, TPNOBLOCK, &revent))
		{
			TP_LOG(log_error, "tpsend failed: %s", tpstrerror(tperrno));
			if (revent!=0)
			{
				switch (revent)
				{
					case TPEV_DISCONIMM:
						TP_LOG(log_error, "got: TPEV_DISCONIMM");
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
		
		TP_LOG(log_info, "Packet delivered ok");
        }
        
out:
	/* close */
	if (fp!=NULL)
	{
		pclose(fp);
	}

	tpreturn(  ret==SUCCEED?TPSUCCESS:TPFAIL,
		0L,
		(char *)p_ub,
		0L,
		0L);
}

/**
 * Consume any sigchilds...
 */
static int periodic(void)
{
	int ret = SUCCEED;
	
	pid_t chldpid;
	int stat_loc;
	struct rusage rusage;

	memset(&rusage, 0, sizeof(rusage));
	
	while (0<(chldpid = wait3(&stat_loc, WNOHANG|WUNTRACED, &rusage)))
	{
		NDRX_LOG(log_warn, "sigchld: PID: %d exit status: %d",
					chldpid, stat_loc);
	}	
	
	return ret;
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
	
	/* ignore sigchilds... as we will consume them by wait */
	signal(SIGCHLD, SIG_IGN);
	
	/* tpext_addperiodcb(5, periodic); */
	
	/* Install periodic callback... */

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
		
		if (0==strcmp(key, "someparam1"))
		{
			TP_LOG(log_debug, "Got param1: [%s]", val);
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
	
	
	/* Advertise our service according to our cluster node id */
	sprintf(svcnm, "MIC%02ld", tpgetnodeid());
	if (SUCCEED!=tpadvertise(svcnm, MIC))
	{
		TP_LOG(log_error, "Failed to initialize MIC!");
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

