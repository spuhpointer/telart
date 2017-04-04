package telart

const (

	/* List of commands, for A_CMD, also used as states in statemachine...
	 * This mirrors the C side header telart.h
	 */

	CMD_HUP_OUR          = 'H' /* our hup phone 			*/
        CMD_HUP_THEIR        = 'h' /* Their HUP                         */
	CMD_PICK_UP          = 'P' /* pick up phone			*/
	CMD_SIGNAL_WAIT      = 'W' /* geneate wait signal		*/
	CMD_SIGNAL_WAIT_STOP = 'w' /* stop the wait signal		*/
	CMD_SIGNAL_BUSY      = 'B' /* Generate busy signal		*/
	CMD_SIGNAL_BUSY_STOP = 'b' /* Stop the wait signal		*/
	CMD_ESTABLISH_CALL   = 'E' /* Establish phone connection	*/
	CMD_RING_BELL        = 'R' /* Ring the bell			*/
	CMD_RING_BELL_STOP   = 'r' /* STOP to ring the bell		*/
	CMD_MIC_XMIT_START   = 'M' /* MIC transmission start		*/
	CMD_MIC_XMIT_STOP    = 'm' /* MIC STOP transmission		*/
	CMD_TIMEOUT          = 'T' /* Generic Time-out command		*/
	CMD_FOUND            = 'F' /* Found target              	*/
	CMD_SYSERR           = 'R' /* System error occurred            	*/

)
