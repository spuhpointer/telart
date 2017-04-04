#ifndef __TELART_H
#define __TELART_H

/*---------------------------Externs------------------------------------*/
/*---------------------------Macros-------------------------------------*/


/* List of commands, for A_CMD, also used as states in statemachine... */

#define CMD_HUP_OUR 		'H' /* hup phone, our 			*/
#define CMD_HUP_THEIR 		'h' /* hup phone, their			*/
#define CMD_PICK_UP 		'P' /* pick up phone			*/
#define CMD_SIGNAL_WAIT 	'W' /* geneate wait signal		*/
#define CMD_SIGNAL_WAIT_STOP 	'w' /* stop the wait signal		*/
#define CMD_SIGNAL_BUSY 	'B' /* Generate busy signal		*/
#define CMD_SIGNAL_BUSY_STOP 	'b' /* Stop the wait signal		*/
#define CMD_ESTABLISH_CALL 	'E' /* Establish phone connection	*/
#define CMD_RING_BELL 		'R' /* Ring the bell			*/
#define CMD_RING_BELL_STOP 	'r' /* STOP to ring the bell		*/
#define CMD_MIC_XMIT_START 	'M' /* MIC transmission start		*/
#define CMD_MIC_XMIT_STOP 	'm' /* MIC STOP transmission		*/
#define CMD_TIMEOUT 	        'T' /* Generic Time-out command		*/
#define CMD_FOUND 	        'F' /* Found target     		*/
#define CMD_SYSERR 	        'R' /* System error occurred   		*/

/*---------------------------Enums--------------------------------------*/
/*---------------------------Typedefs-----------------------------------*/
/*---------------------------Globals------------------------------------*/
/*---------------------------Statics------------------------------------*/
/*---------------------------Prototypes---------------------------------*/

#endif
