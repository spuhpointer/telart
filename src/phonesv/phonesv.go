package main

import (
	"fmt"
	t "include"
	"math/rand"
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

	ConstFindPhoneTime = 30 /* Search for phone 30 sec */
)

//Get UTC milliseconds since epoch
//@return epoch milliseconds
func GetEpochMillis() int64 {
	now := time.Now()
	nanos := now.UnixNano()
	millis := nanos / 1000000

	return millis
}

//About incoming & outgoing messages:
type StopWatch struct {
	start int64 //Timestamp messag sent
}

//Reset the stopwatch
func (s *StopWatch) Reset() {
	s.start = GetEpochMillis()
}

//Get delta milliseconds
//@return time spent in milliseconds
func (s *StopWatch) GetDeltaMillis() int64 {
	return GetEpochMillis() - s.start
}

//Get delta seconds of the stopwatch
//@return return seconds spent
func (s *StopWatch) GetDetlaSec() int64 {
	return (GetEpochMillis() - s.start) / 1000
}

type TransitionFunc func(ac *atmi.ATMICtx) atmi.ATMIError

type TransitionFuncTranslate func(ac *atmi.ATMICtx, errA atmi.ATMIError) string

type Transition struct {
	cmd        byte           /* Command, see t.CMD_ */
	f          TransitionFunc /* transision func 1 */
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
			/* if having some late tout... */
			Transition{cmd: t.CMD_TIMEOUT, f: nil, next_state: SIdle},
			Transition{cmd: t.CMD_HUP_OUR, f: nil, next_state: SIdle},
			Transition{cmd: t.CMD_PICK_OUR, f: GoFindFreePhone, next_state: SActivFind},
			/* They send us ring the bell - if idle, accept... */
			Transition{cmd: t.CMD_RING_BELL, f: SetLockToPartner, next_state: SPasivRing},
		},
	},
	State{
		state: SActivFind, voice: false, ring: false, playBusy: false, playWait: true, tout: ConstFindPhoneTime,
		transitions: []Transition{
			Transition{cmd: t.CMD_TIMEOUT, f: nil, next_state: SAllBusy},
			Transition{cmd: t.CMD_FOUND, f: nil, next_state: SActivRing},
		},
	},
	State{
		/* ring their */
		state: SActivRing, voice: false, ring: true, playBusy: false, playWait: true, tout: ConstFindPhoneTime,
		transitions: []Transition{
			Transition{cmd: t.CMD_TIMEOUT, f: nil, next_state: SAllBusy},
			/* they send us establish... */
			Transition{cmd: t.CMD_PICK_THEIR, f: nil, next_state: SActivConv},
			Transition{cmd: t.CMD_HUP_OUR, f: SendHUP, next_state: SIdle},
			Transition{cmd: t.CMD_RING_BELL, f: SetAnswerBusy, next_state: SActivRing},
		},
	},
	State{
		state: SActivConv, voice: false, ring: false, playBusy: false, playWait: false, tout: 600,
		transitions: []Transition{
			Transition{cmd: t.CMD_TIMEOUT, f: SendHUP, next_state: SAllBusy},
			Transition{cmd: t.CMD_HUP_OUR, f: SendHUP, next_state: SIdle},
			Transition{cmd: t.CMD_HUP_THEIR, f: nil, next_state: SAllBusy},
			Transition{cmd: t.CMD_RING_BELL, f: SetAnswerBusy, next_state: SActivConv},
		},
	},
	State{
		state: SAllBusy, voice: false, ring: false, playBusy: true, playWait: false, tout: -1,
		transitions: []Transition{
			Transition{cmd: t.CMD_HUP_OUR, f: nil, next_state: SIdle},
			Transition{cmd: t.CMD_RING_BELL, f: SetAnswerBusy, next_state: SAllBusy},
		},
	},

	/* passive states: we receive the call:
	 * The other node is generating ring...
	 */
	State{
		state: SPasivRing, voice: false, ring: false, playBusy: false, playWait: false, tout: 90,
		transitions: []Transition{
			Transition{cmd: t.CMD_TIMEOUT, f: SendTimeOut, next_state: SIdle},
			Transition{cmd: t.CMD_PICK_OUR, f: SendPick, next_state: SPasivConv},
			Transition{cmd: t.CMD_HUP_THEIR, f: nil, next_state: SIdle},
			Transition{cmd: t.CMD_RING_BELL, f: SetAnswerBusy, next_state: SPasivRing},
		},
	},
	State{
		state: SPasivConv, voice: true, ring: false, playBusy: false, playWait: false, tout: 600,
		transitions: []Transition{
			Transition{cmd: t.CMD_TIMEOUT, f: SendHUP, next_state: SAllBusy},
			Transition{cmd: t.CMD_HUP_OUR, f: SendHUP, next_state: SIdle},
			Transition{cmd: t.CMD_HUP_THEIR, f: nil, next_state: SIdle},
		},
	},
}

var MOurNode int       /* our call end */
var MTheirNode int     /* their call end... */
var MTheirNodeLast int /* Last their node (last command from) */

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

var MMinNode = 1 /* search in random from... */
var MMaxNode = 6 /* search in random to... */

var MAnswer byte

var MachineLock = &sync.Mutex{}

var MTout = -1
var MToutStamp int64

var MScheduleNextCmd byte = 0 /* No command at the moment */

//Send the command to locked partner
//@param ac	ATMI Context into which send the command
//@param cmd	Command out
//@param cmd	Command received back...
//@return error or nil
func SendCmd(ac *atmi.ATMICtx, cmd byte, cmdRet *byte) atmi.ATMIError {
	buf, errB := ac.NewUBF(16 * 1024)
	if nil != errB {
		ac.TpLogError("Failed to allocate buffer: [%s]", errB.Error())
		MSysError = true
		return atmi.NewCustomATMIError(atmi.TPESYSTEM,
			fmt.Sprintf("Failed to allocate buffer: [%s]", errB.Error()))
	}

	if errB := buf.BChg(u.A_CMD, 0, cmd); errB != nil {
		ac.TpLogError("Failed to set A_CMD to [%c]: [%s]",
			cmd, errB.Error())
		MSysError = true

		return atmi.NewCustomATMIError(atmi.TPESYSTEM,
			fmt.Sprintf("Failed to set A_CMD to [%c]: [%s]",
				cmd, errB.Error()))
	}

	if errB := buf.BChg(u.A_SRC_NODE, 0, MOurNode); errB != nil {
		ac.TpLogError("Failed to set A_SRC_NODE: [%s]",
			errB.Error())
		MSysError = true

		return atmi.NewCustomATMIError(atmi.TPESYSTEM,
			fmt.Sprintf("Failed to set A_SRC_NODE: [%s]",
				errB.Error()))
	}

	//Call the server
	svc := fmt.Sprintf("PHONE%02d", MTheirNode)

	ac.TpLogInfo("Calling phone: [%s]", svc)

	if _, err := ac.TpCall(svc, buf, 0); nil != err {
		ac.TpLogError("ATMI Error %d:[%s]", err.Code(), err.Message())
		return err
	}

	/* read the command back */
	*cmdRet = 0

	if *cmdRet, errB = buf.BGetByte(u.A_CMD, 0); errB != nil {
		ac.TpLogError("Failed to get A_CMD from phone call: [%s]",
			cmd, errB.Error())
	}

	ac.TpLogInfo("Got command back: %c", rune(*cmdRet))

	return nil

}

func random(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

//Search for free phone
func GoFindFreePhone(ac *atmi.ATMICtx) atmi.ATMIError {

	var w StopWatch

	MTheirNode = 0
	w.Reset()

	for w.GetDetlaSec() < ConstFindPhoneTime {
		//Get random host
		tryNode := random(MMinNode, MMaxNode)

		ac.TpLogInfo("Trying to call to: %d", tryNode)
		//Try to access it
		var cmdRet byte

		errA := SendCmd(ac, t.CMD_RING_BELL, &cmdRet)

		if errA.Code() == atmi.TPMINVAL {
			ac.TpLogInfo("Call ok, command ret: %c", rune(cmdRet))
			if cmdRet == t.CMD_PICK_THEIR {
				ac.TpLogInfo("Their accepted incoming call")
				MTheirNode = tryNode
			}
		} else {
			//If not locked, then sleep(500 ms)
			//Ignore the error on get some sleep
			time.Sleep(time.Duration(500) * time.Millisecond)
		}
	}

	//Generate timeout...
	if MTheirNode > 0 {
		MScheduleNextCmd = t.CMD_TIMEOUT
	} else {
		MScheduleNextCmd = t.CMD_PICK_THEIR
	}

	return nil
}

//Send timeout command to their node
func SendTimeOut(ac *atmi.ATMICtx) atmi.ATMIError {
	var cmdRet byte
	return SendCmd(ac, t.CMD_TIMEOUT, &cmdRet)
}

//Send HUP signal to their
func SendHUP(ac *atmi.ATMICtx) atmi.ATMIError {
	var cmdRet byte
	return SendCmd(ac, t.CMD_HUP_THEIR, &cmdRet)
}

//Send Establish to their node
func SendPick(ac *atmi.ATMICtx) atmi.ATMIError {
	var cmdRet byte
	return SendCmd(ac, t.CMD_PICK_THEIR, &cmdRet)
}

//We are locking to to caller partner
func SetLockToPartner(ac *atmi.ATMICtx) atmi.ATMIError {
	ac.TpLogInfo("Locking to partner: %d", MTheirNodeLast)
	MTheirNode = MTheirNodeLast
	MAnswer = t.CMD_LOCK
	return nil
}

//We are busy, thus respond with busy signal...
func SetAnswerBusy(ac *atmi.ATMICtx) atmi.ATMIError {
	ac.TpLogInfo("Sending to partner: %d busy signal", MTheirNodeLast)
	MAnswer = t.CMD_SIGNAL_BUSY
	return nil
}

//Find the state
//Simple one, we could use binary search, but we do not have such number of states
//and execution is not so often..
//@param state 	state to search for
//@return state found or nil
func FindState(state string) *State {

	for _, elm := range Machine {
		if elm.state == state {

			return &elm
		}
	}
	return nil
}

//Find the transision within state
//Not the best way, as we could do some binary search, but
//we do not have such quantities of states...
//@param state	State to search within
//@param cmd	Transition command to Find
//@return transision found or nil
func FindTransision(state *State, cmd byte) *Transition {

	for _, elm := range state.transitions {
		if elm.cmd == cmd {
			return &elm
		}
	}

	return nil
}

//Go global timeout...
//Lock to some timestamp...
func GoTimeout() {

	stamp := MToutStamp
	tout := MTout

	//Go sleep
	time.Sleep(time.Duration(tout) * time.Second)

	if stamp == MToutStamp && tout == MTout {
		//Generate timeout command

		ac, errA := atmi.NewATMICtx()

		if nil != errA {
			fmt.Fprintf(os.Stderr,
				"Failed to allocate new context for tout: %s",
				errA.Message())
			MSysError = true
			return
		}

		ac.TpLogError("Timeout condition, spent: %d", MTout)

		StepStateMachine(ac, t.CMD_TIMEOUT)
	}
}

//Step the state machine - execute the transitions & switch the states
//The exeuction/command sources can be different ones - internal routines
//or XATMI servic call sources
//NOTE: The time-out generator must fix the state at which it is started
//If state is switched then timeout command must be ignored as it entered in
//race condion.
//@param cmd 	Command to run
func StepStateMachine(ac *atmi.ATMICtx, cmd byte) {
	MachineLock.Lock()

next:
	ac.TpLogInfo("Current state: [%s], got command: %c", MState, rune(cmd))
	curState := FindState(MState)

	if nil == curState {
		ac.TpLog(atmi.LOG_ERROR, "ERROR ! Current state not found: %s", MState)
		/* Should be picked up by periodic scan and terminate the server */
		MSysError = true
		return
	}

	curTran := FindTransision(curState, cmd)

	if nil == curTran {
		ac.TpLog(atmi.LOG_ERROR, "Transition not found! State: %s cmd: %c - ignore...",
			MState, rune(cmd))
		return
	}

	ac.TpLogInfo("Executing transition, next state: [%s]", curTran.next_state)

	/* execute transisions... */
	if nil != curTran.f {
		ac.TpLogInfo("Executing f1")
		err := curTran.f(ac)
		ac.TpLogInfo("Got error from transition: [%s] - ignore.", err.Error())
	}

	/* Switch next state... */
	nextState := FindState(curTran.next_state)
	if nil == nextState {
		ac.TpLog(atmi.LOG_ERROR, "ERROR ! Next state not found: %s", curTran.next_state)
		/* Should be picked up by periodic scan and terminate the server */
		MSysError = true
		return
	}

	/* compare the state data... */

	ac.TpLog(atmi.LOG_INFO, "TRAN: State: %s (cur: %s) voice: %t ring: %t "+
		"busy: %t wait: %t tout: %d",
		nextState.state, MState,
		nextState.voice, nextState.ring, nextState.playBusy,
		nextState.playWait, nextState.tout)
	ac.TpLog(atmi.LOG_INFO, "CUR: State: %s voice: %t ring: %t busy: %t "+
		"wait: %t tout: %d (stamp: %d)",
		MState, MVoice, MRing, MBusy, MWait, MTout, MToutStamp)

	/* Process voice block: */
	if nextState.voice && !MVoice {
		ac.TpLogWarn("Voice start")
		MVoice = true
		go GoVoice(MOurNode, MTheirNode)
	} else if !nextState.voice && MVoice {
		ac.TpLogWarn("Voice terminate")
		MVoice = false
	}

	/* Process ring block: */
	if nextState.ring && !MRing {
		ac.TpLogWarn("Ring start")
		MRing = true
		go GoRing(MTheirNode)
	} else if !nextState.voice && MRing {
		ac.TpLogWarn("Ring terminate")
		MRing = false
	}

	/* Process busy block: */
	if nextState.playBusy && !MBusy {
		ac.TpLogWarn("Play Busy start")
		MBusy = true
		go GoPlayback(MOurNode, t.CMD_SIGNAL_BUSY)
	} else if !nextState.voice && MBusy {
		ac.TpLogWarn("Play Busy terminate")
		MBusy = false
	}

	/* Process wait block: */
	if nextState.playBusy && !MWait {
		ac.TpLogWarn("Play Wait start")
		MWait = true
		go GoPlayback(MOurNode, t.CMD_SIGNAL_BUSY)
	} else if !nextState.voice && MWait {
		ac.TpLogWarn("Play Wait terminate")
		MWait = false
	}

	/* Finally switch the state */

	MState = nextState.state

	MTout = nextState.tout
	MToutStamp = time.Now().UnixNano()

	if nextState.tout > 0 {
		ac.TpLogInfo("Setting timeout to: %d", nextState.tout)

		go GoTimeout()
	}

	if MScheduleNextCmd > 0 {
		cmd = MScheduleNextCmd
		MScheduleNextCmd = 0
		goto next
	}

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
func GoPlayback(node int, whatCmd byte) {

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

		if SUCCEED != ret {
			StepStateMachine(ac, t.CMD_SYSERR)
		}
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

			ac.TpLogError("Failed to receive mic data: %s (%d)",
				errA.Message(), revent)

			if revent != atmi.TPEV_DISCONIMM {
				ret = FAIL
			}

			return

		}

		buf.TpLogPrintUBF(atmi.LOG_DEBUG, "Transfering")

		//Send audio data to playback... data
		if errA := ac.TpSend(cdP, buf.GetBuf(), 0, &revent); nil != errA {

			ac.TpLogError("Failed to send sound data: %s (%d)",
				errA.Message(), revent)

			if revent != atmi.TPEV_DISCONIMM {
				ret = FAIL
			}

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
	cmd, errB := ub.BGetByte(u.A_CMD, 0)

	if nil != errB {
		ac.TpLogError("Failed to get A_CMD: %s", errB.Error())
		ret = FAIL
		return
	}

	source, errB := ub.BGetInt(u.A_SRC_NODE, 0)
	if nil != errB {
		ac.TpLogError("Failed to get A_SRC_NODE: %s", errB.Error())
		ret = FAIL
		return
	}

	ac.TpLogInfo("Got command: from node: %d command: %c", source, rune(cmd))
	step := false

	MAnswer = 0

	//At idle we allow all nodes to enter..
	MTheirNodeLast = source
	if MState == SIdle {
		step = true
		ac.TpLogInfo("We are at idle, allow any command...")
	} else if cmd == t.CMD_RING_BELL {
		//Accept ring bell...
		step = true
		ac.TpLogInfo("Incoming bell ring...")
	}

	if step {
		ac.TpLogInfo("Stepping the state machine...")
		StepStateMachine(ac, cmd)
	}

	/* check the response command... */
	if MAnswer > 0 {
		if errB := ub.BChg(u.A_SRC_NODE, 0, MOurNode); nil != errB {
			ac.TpLogError("Failed to setup A_SRC_NODE: %s", errB.Error())
			ret = FAIL
			return
		} else if errB := ub.BChg(u.A_CMD, 0, MAnswer); nil != errB {
			ac.TpLogError("Failed to setup A_CMD answer: %s", errB.Error())
			ret = FAIL
			return
		}
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

		case "min":
			MMinNode, _ := buf.BGetInt(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MMinNode)
			break
		case "max":
			MMaxNode, _ := buf.BGetInt(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MMaxNode)
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
