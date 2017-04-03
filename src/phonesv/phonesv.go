package main

import (
	"fmt"
	"os"
	u "ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

const (
	SUCCEED     = atmi.SUCCEED
	FAIL        = atmi.FAIL
	PROGSECTION = "phonesv"

	CALL_MODE_NONE    = 0
	CALL_MODE_ACTIVE  = 1
	CALL_MODE_APSSIVE = 2

	/*
	 * Active stages:
	 */
	CALL_STAGE_NONE   = 0 /* no call in progress */
	CALL_STAGE_SEARCH = 1 /* search for free phone */
	CALL_STAGE_RING   = 2 /* righ their bell our, play: ---.--- */
	CALL_STAGE_BUSY   = 3 /* no phone found, play: -.-.- */
	CALL_IN_CALL      = 4 /* call established active/passive */
	CALL_THEIR_HUP    = 5 /* active/passive, other end hup */
)

var MInCall bool = false
var MCallMode int = CALL_MODE_NONE
var MStage = CALL_STAGE_NONE

/* TODO: */
var MMinNode = 1  /* search in random from... */
var MMaxNode = 20 /* search in random to... */

//Redirec the voice from MIC to PHONE
//@param
func GoVoice(fromMic int, toPhone int) {

	var revent int64

	micSvc := fmt.Sprintf("MIC%02d", fromMic)
	phoneSvc := fmt.Sprintf("LIVEPLAY%02d", toPhone)

	ret := SUCCEED

	ac, errA := atmi.NewATMICtx()

	if nil != errA {
		fmt.Fprintf(os.Stderr, "Failed to allocate new context: %s",
			errA.Message())
		os.Exit(atmi.FAIL)
	}

	//Return to the caller
	defer func() {

		ac.TpLogError("Voice terminates with  %d", ret)
	}()

	//Allocate configuration buffer
	buf, errB := ac.NewUBF(16 * 1024)
	if nil != errB {
		ac.TpLogError("Failed to allocate buffer: [%s]", errB.Error())
	}

	//Allocate data buffer (UBF)
	cdM, errA := ac.TpConnect(micSvc, buf.GetBuf(),
		atmi.TPNOTRAN|atmi.TPRECVONLY)

	defer ac.TpDiscon(cdM)

	cdP, errA := ac.TpConnect(phoneSvc, buf.GetBuf(),
		atmi.TPNOTRAN|atmi.TPSENDONLY)

	defer ac.TpDiscon(cdP)

	//Establish connection
	for MInCall {

		//Get mic data
		if errA := ac.TpRecv(cdM, buf.GetBuf(), 0, &revent); nil != errA {

			ac.TpLogError("Failed to receive mic data: %s",
				errA.Message())

			ret = FAIL
			return
		}

		buf.TpLogPrintUBF(atmi.LOG_DEBUG, "Transfering")

		//Send audio data to playback... data
		if errA := ac.TpSend(cdP, buf.GetBuf(), 0, &revent); nil != errA {

			ac.TpLogError("Failed to send sound data: %s",
				errA.Message())

			ret = FAIL
			return

		}
	}

}

//PHONE service
//@param ac ATMI Context
//@param svc Service call information
func PHONE(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := SUCCEED

	//Return to the caller
	defer func() {

		ac.TpLogCloseReqFile()
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, &svc.Data, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, &svc.Data, 0)
		}
	}()

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	//Print the buffer to stdout
	//fmt.Println("Incoming request:")
	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Incoming request:")

	//Add test field to buffer
	hup, errB := ub.BGetInt(u.A_HUP, 0)

	if nil != errB {
		ac.TpLogError("BGetInt() Got error: %s", errB.Error())
		ret = FAIL
		return
	}

	switch hup {

	case 0:
		ac.TpLogInfo("Terminating call...")
		MInCall = false
		break

	case 1:
		ac.TpLogInfo("Starting call...")
		MInCall = true
		go GoVoice(19, 19)
		break

	default:
		ac.TpLogError("Invalid command: %d", hup)

	}

	return
}

//Server init, called when process is booted
//@param ac ATMI Context
func Init(ac *atmi.ATMICtx) int {

	ac.TpLogWarn("Doing server init...")
	if err := ac.TpInit(); err != nil {
		return FAIL
	}

	//Get the configuration

	//Allocate configuration buffer
	buf, err := ac.NewUBF(16 * 1024)
	if nil != err {
		ac.TpLogError("Failed to allocate buffer: [%s]", err.Error())
		return FAIL
	}

	buf.BChg(u.EX_CC_CMD, 0, "g")
	buf.BChg(u.EX_CC_LOOKUPSECTION, 0, fmt.Sprintf("%s/%s", PROGSECTION, os.Getenv("NDRX_CCTAG")))

	if _, err := ac.TpCall("@CCONF", buf, 0); nil != err {
		ac.TpLogError("ATMI Error %d:[%s]\n", err.Code(), err.Message())
		return FAIL
	}

	//Dump to log the config read
	buf.TpLogPrintUBF(atmi.LOG_DEBUG, "Got configuration.")

	occs, _ := buf.BOccur(u.EX_CC_KEY)

	// Load in the config...
	for occ := 0; occ < occs; occ++ {
		ac.TpLogDebug("occ %d", occ)
		fldName, err := buf.BGetString(u.EX_CC_KEY, occ)

		if nil != err {
			ac.TpLogError("Failed to get field "+
				"%d occ %d", u.EX_CC_KEY, occ)
			return FAIL
		}

		ac.TpLogDebug("Got config field [%s]", fldName)

		switch fldName {

		case "mykey1":
			myval, _ := buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%s] ", fldName, myval)
			break

		default:

			break
		}
	}
	//Advertize service
	if err := ac.TpAdvertise("PHONE", "PHONE", PHONE); err != nil {
		ac.TpLogError("Failed to Advertise: ATMI Error %d:[%s]\n", err.Code(), err.Message())
		return atmi.FAIL
	}

	return SUCCEED
}

//Server shutdown
//@param ac ATMI Context
func Uninit(ac *atmi.ATMICtx) {
	ac.TpLogWarn("Server is shutting down...")
	MInCall = false
	MCallMode = CALL_MODE_NONE
	MStage = CALL_STAGE_NONE
}

//Executable main entry point
func main() {
	//Have some context
	ac, err := atmi.NewATMICtx()

	if nil != err {
		fmt.Fprintf(os.Stderr, "Failed to allocate new context: %s", err)
		os.Exit(atmi.FAIL)
	} else {
		//Run as server
		if err = ac.TpRun(Init, Uninit); nil != err {
			ac.TpLogError("Exit with failure")
			os.Exit(atmi.FAIL)
		} else {
			ac.TpLogInfo("Exit with success")
			os.Exit(atmi.SUCCEED)
		}
	}
}
