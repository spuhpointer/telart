package main

import (
	"fmt"
	t "include"
	"os"
	"sync"
	"time"
	u "ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

const (
	SUCCEED     = atmi.SUCCEED
	FAIL        = atmi.FAIL
	PROGSECTION = "phonesv"

	/*
	 * Active stages:
	 */
	SIdle      = "Idle"         /* Idle state */
	SActivFind = "ActFind"      /* Find the target phone */
	SAllBusy   = "ActivAllBusy" /* All phones are busy */
	SActivRing = "ActRing"      /* Ring the target phone */
	SActivConv = "ActConv"      /* Active conversation */
	SPasivRing = "PasivRing"    /* We go the ring */
	SPasivConv = "PasivConv"    /* We go into conversion */
)

type TransitionFunc func(ac *atmi.ATMICtx) error

type Transition struct {
	cmd        rune           /* Command, see t.CMD_ */
	f1         TransitionFunc /* transision func 1 */
	f2         TransitionFunc /* transision func 2 */
	f3         TransitionFunc /* transision func 3 */
	next_state string         /* Next state */
}

type State struct {
	state       string /* state, see S* */
	voice       bool   /* run voice */
	ring        bool   /* Ring the bell on taret system */
	playBusy    bool   /* Play busy? */
	playWait    bool   /* Play wait at state */
	tout        int    /* timeout */
	transitions []Transition
}

var Machine = []State{
	/* Active states: we do the call: */
	State{
		state: SIdle, voice: false, ring: false, playBusy: false, playWait: false, tout: -1,
		transitions: []Transition{
			Transition{cmd: t.CMD_HUP_OUR, f1: nil, f2: nil, f3: nil, next_state: SIdle},
			Transition{cmd: t.CMD_PICK_UP, f1: GoFindFreePhone, f2: nil, f3: nil, next_state: SActivFind},
			/* They send us ring the bell - if idle, accept... */
			Transition{cmd: t.CMD_RING_BELL, f1: nil, f2: nil, f3: nil, next_state: SPasivRing},
		},
	},
	State{
		state: SActivFind, voice: false, ring: false, playBusy: false, playWait: true, tout: 90,
		transitions: []Transition{
			Transition{cmd: t.CMD_TIMEOUT, f1: nil, f2: nil, f3: nil, next_state: SAllBusy},
			Transition{cmd: t.CMD_FOUND, f1: nil, f2: nil, f3: nil, next_state: SActivRing},
			Transition{cmd: t.CMD_RING_BELL, f1: SetAnswerBusy, f2: nil, f3: nil, next_state: SActivFind},
		},
	},
	State{
		state: SActivRing, voice: false, ring: false, playBusy: false, playWait: true, tout: 90,
		transitions: []Transition{
			Transition{cmd: t.CMD_TIMEOUT, f1: nil, f2: nil, f3: nil, next_state: SAllBusy},
			/* they send us establish... */
			Transition{cmd: t.CMD_ESTABLISH_CALL, f1: nil, f2: nil, f3: nil, next_state: SActivConv},
			Transition{cmd: t.CMD_RING_BELL, f1: SetAnswerBusy, f2: nil, f3: nil, next_state: SActivRing},
			Transition{cmd: t.CMD_HUP_OUR, f1: SendHUP, f2: nil, f3: nil, next_state: SAllBusy},
		},
	},
	State{
		state: SActivConv, voice: false, ring: false, playBusy: false, playWait: false, tout: 600,
		transitions: []Transition{
			Transition{cmd: t.CMD_TIMEOUT, f1: SendHUP, f2: nil, f3: nil, next_state: SAllBusy},
			Transition{cmd: t.CMD_HUP_OUR, f1: SendHUP, f2: nil, f3: nil, next_state: SIdle},
			Transition{cmd: t.CMD_HUP_THEIR, f1: nil, f2: nil, f3: nil, next_state: SAllBusy},
			Transition{cmd: t.CMD_RING_BELL, f1: SetAnswerBusy, f2: nil, f3: nil, next_state: SActivConv},
		},
	},
	State{
		state: SAllBusy, voice: false, ring: false, playBusy: true, playWait: false, tout: -1,
		transitions: []Transition{
			Transition{cmd: t.CMD_HUP_OUR, f1: nil, f2: nil, f3: nil, next_state: SIdle},
			Transition{cmd: t.CMD_RING_BELL, f1: SetAnswerBusy, f2: nil, f3: nil, next_state: SAllBusy},
		},
	},

	/* passive states: we receive the call: */
	State{
		state: SPasivRing, voice: false, ring: true, playBusy: false, playWait: false, tout: 90,
		transitions: []Transition{
			Transition{cmd: t.CMD_TIMEOUT, f1: SendTimeOut, f2: nil, f3: nil, next_state: SIdle},
			Transition{cmd: t.CMD_PICK_UP, f1: SendEstablish, f2: nil, f3: nil, next_state: SPasivConv},
			Transition{cmd: t.CMD_HUP_THEIR, f1: nil, f2: nil, f3: nil, next_state: SIdle},
			Transition{cmd: t.CMD_RING_BELL, f1: SetAnswerBusy, f2: nil, f3: nil, next_state: SPasivRing},
		},
	},
	State{
		state: SPasivConv, voice: false, ring: true, playBusy: false, playWait: false, tout: 600,
		transitions: []Transition{
			Transition{cmd: t.CMD_TIMEOUT, f1: SendHUP, f2: nil, f3: nil, next_state: SAllBusy},
			Transition{cmd: t.CMD_HUP_OUR, f1: SendHUP, f2: nil, f3: nil, next_state: SIdle},
			Transition{cmd: t.CMD_HUP_THEIR, f1: nil, f2: nil, f3: nil, next_state: SIdle},
		},
	},
}

var MOurNode long   /* our call end */
var MTheirNode long /* their call end... */

/* voice our MIC to their Phone */
var MVoice bool = false

/* Playback of sounds in our phone */
var MBusy bool = false
var MWait bool = false

/* Do the ring */

var MRing bool = false

var MState = SIdle
var MSysError bool = false
var MTimeout bool = false /* Is current state timed out... */

/* TODO: */
var MMinNode = 1  /* search in random from... */
var MMaxNode = 20 /* search in random to... */

var MAnswer rune

var MachineLock = &sync.Mutex{}

//Search for free phone
func GoFindFreePhone(ac *atmi.ATMICtx) error {

	return nil
}

//Send timeout command to their node
func SendTimeOut(ac *atmi.ATMICtx) error {

	return nil
}

func SendHUP(ac *atmi.ATMICtx) error {

	return nil
}

//Send Establish to their node
func SendEstablish(ac *atmi.ATMICtx) error {

	return nil
}

func SetAnswerBusy(ac *atmi.ATMICtx) error {

	MAnswer = t.CMD_SIGNAL_BUSY
	return nil
}

//Step the state machine - execute the transitions & switch the states
//The exeuction/command sources can be different ones - internal routines
//or XATMI servic call sources
//NOTE: The time-out generator must fix the state at which it is started
//If state is switched then timeout command must be ignored as it entered in
//race condion.
//@param cmd 	Command to run
func StepStateMachine(ac *atmi.ATMICtx, cmd rune) {
	MachineLock.Lock()

	//Run the state machine here

	MachineLock.Unlock()
}

//Ring the bell
//@param
func GoRing(node int) {

	var revent int64

	bellSvc := fmt.Sprintf("BELL%02d", node)

	ret := SUCCEED

	ac, errA := atmi.NewATMICtx()

	if nil != errA {
		fmt.Fprintf(os.Stderr, "Failed to allocate new context: %s",
			errA.Message())
		MSysError = true
		os.Exit(atmi.FAIL)
	}

	//Return to the caller
	defer func() {

		ac.TpLogError("Ring terminates with  %d", ret)
		MRing = false
	}()

	//Allocate configuration buffer
	buf, errB := ac.NewUBF(16 * 1024)
	if nil != errB {
		ac.TpLogError("Failed to allocate buffer: [%s]", errB.Error())
		MSysError = true
		return
	}

	if errB := buf.BChg(u.A_CMD, 0, t.CMD_RING_BELL); errB != nil {
		ac.TpLogError("Failed to set A_CMD to [%c]: [%s]",
			t.CMD_RING_BELL, errB.Error())
		MSysError = true
		return
	}

	//Allocate data buffer (UBF)
	cdP, errA := ac.TpConnect(bellSvc, buf.GetBuf(),
		atmi.TPNOTRAN|atmi.TPSENDONLY) //<<< Set to RCVONLY to get segfault!

	//Possible causes segementation faul!!!
	defer ac.TpDiscon(cdP)

	//Establish connection
	for MRing {

		buf.TpLogPrintUBF(atmi.LOG_DEBUG, "Sending ring clock...")

		//Send audio data to playback... data
		if errA := ac.TpSend(cdP, buf.GetBuf(), 0, &revent); nil != errA {

			ac.TpLogError("Failed to send sound data: %s",
				errA.Message())

			ret = FAIL
			return

		}

		time.Sleep(100 * time.Millisecond)
	}
}

//Redirec the voice from MIC to PHONE
//@param
func GoPlayback(node int, whatCmd string) {

	var revent int64

	playBackSvc := fmt.Sprintf("PLAYBACK%02d", node)

	ret := SUCCEED

	ac, errA := atmi.NewATMICtx()

	if nil != errA {
		fmt.Fprintf(os.Stderr, "Failed to allocate new context: %s",
			errA.Message())
		MSysError = true
		os.Exit(atmi.FAIL)
	}

	//Return to the caller
	defer func() {

		ac.TpLogError("Voice terminates with  %d", ret)
		MBusy = false
		MWait = false
	}()

	//Allocate configuration buffer
	buf, errB := ac.NewUBF(16 * 1024)
	if nil != errB {
		ac.TpLogError("Failed to allocate buffer: [%s]", errB.Error())
		MSysError = true
		return
	}

	if errB := buf.BChg(u.A_CMD, 0, whatCmd); errB != nil {
		ac.TpLogError("Failed to set A_CMD to [%s]: [%s]",
			whatCmd, errB.Error())
		MSysError = true
		return
	}

	//Allocate data buffer (UBF)
	cdP, errA := ac.TpConnect(playBackSvc, buf.GetBuf(),
		atmi.TPNOTRAN|atmi.TPSENDONLY) //<<< Set to RCVONLY to get segfault!

	//Possible causes segementation faul!!!
	defer ac.TpDiscon(cdP)

	//Establish connection
	for MBusy || MWait {

		buf.TpLogPrintUBF(atmi.LOG_DEBUG, "Sending playback clock...")

		//Send audio data to playback... data
		if errA := ac.TpSend(cdP, buf.GetBuf(), 0, &revent); nil != errA {

			ac.TpLogError("Failed to send sound data: %s",
				errA.Message())

			ret = FAIL
			return

		}

		time.Sleep(100 * time.Millisecond)
	}
}

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
	for MVoice {

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
		MVoice = false
		MBusy = false
		break

	case 1:
		ac.TpLogInfo("Starting call...")
		MVoice = true
		MBusy = true
		//go GoVoice(19, 19)
		go GoPlayback(19, "B")
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

	MVoice = false

	//TODO: Generate basic HUP signal...
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
