#ifndef __TELART_H
#define __TELART_H

/*---------------------------Externs------------------------------------*/
/*---------------------------Macros-------------------------------------*/


/* List of commands, for A_CMD, also used as states in statemachine... */

#define CMD_HUP_OUR 		'H' /* hup phone, our 			*/
#define CMD_HUP_THEIR 		'h' /* hup phone, their			*/
#define CMD_PICK_OUR 		'P' /* pick up phone, our		*/
#define CMD_PICK_THEIR 		'p' /* pick up phone, their		*/
#define CMD_SIGNAL_WAIT 	'W' /* geneate wait signal		*/
#define CMD_SIGNAL_WAIT_STOP 	'w' /* stop the wait signal		*/
#define CMD_SIGNAL_BUSY 	'B' /* Generate busy signal		*/
#define CMD_SIGNAL_BUSY_STOP 	'b' /* Stop the wait signal		*/
#define CMD_RING_BELL 		'R' /* Ring the bell			*/
#define CMD_RING_BELL_STOP 	'r' /* STOP to ring the bell		*/
#define CMD_MIC_XMIT_START 	'M' /* MIC transmission start		*/
#define CMD_MIC_XMIT_STOP 	'm' /* MIC STOP transmission		*/
#define CMD_TIMEOUT 	        'T' /* Generic Time-out command		*/
#define CMD_FOUND 	        'F' /* Found target     		*/
#define CMD_SYSERR 	        'R' /* System error occurred   		*/
#define CMD_EXIT 	        'X' /* System exit   		        */
#define CMD_DIAG_RING 	        'D' /* Diagnostic ring 		        */
#define CMD_DIAG_RINGOFF        'd' /* Diagnostic ring, off		*/

/*---------------------------Enums--------------------------------------*/
/*---------------------------Typedefs-----------------------------------*/
/*---------------------------Globals------------------------------------*/
/*---------------------------Statics------------------------------------*/
/*---------------------------Prototypes---------------------------------*/

#endif
