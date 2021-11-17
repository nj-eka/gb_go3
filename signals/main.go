package main

import (
	"context"
	"fmt"
	"golang.org/x/sys/unix"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	appTimeout     = 30 * time.Second
	sigGenInterval = 100 * time.Millisecond
)

func gotSIGHUP() {
	log.Print("SIGHUP handler")
}
func gotSIGINT() {
	log.Print("SIGINT handler")
}
func gotSIGTERM() {
	log.Print("SIGTERM handler")
}
func gotSIGQUIT() {
	log.Print("SIGQUIT handler")
}

type sigHandler func()

var (
	sigHandlers = map[syscall.Signal]sigHandler{
		syscall.SIGHUP:  gotSIGHUP,
		syscall.SIGINT:  gotSIGINT,
		syscall.SIGTERM: gotSIGTERM,
		syscall.SIGQUIT: gotSIGQUIT,
	}
)

func runSignalHandlers(parentCtx context.Context, sigs <-chan os.Signal) {
	for {
		select {
		case <-parentCtx.Done():
			log.Printf("stop signal processing: %v\n", parentCtx.Err())
			return
		case sig, more := <-sigs:
			if more {
				log.Printf("<- Got %T: '%+v' (#%#v)\n", sig, sig, sig)
				switch sig := sig.(type) {
				case syscall.Signal:
					if handler, ok := sigHandlers[sig]; ok {
						log.Println("run handler...")
						handler()
					} else {
						log.Println("no handler is registered for this signal")
					}
				default:
					log.Printf("Unknown signal signatute...") // just in case)
				}
			} else {
				log.Println("stop signal processing: signal channel closed")
				return
			}
		}
	}
}

func GetSystemSignals() (result []syscall.Signal) {
	// See https://github.com/golang/go/issues/28027#issuecomment-427377759
	for signum := syscall.Signal(0); signum < syscall.Signal(255); signum++ {
		if signame := unix.SignalName(signum); signame != "" {
			fmt.Printf("%s (#%d) - %v\n", signame, signum, signum)
			result = append(result, signum)
		}
	}
	return
}

func main() {
	pid := os.Getpid()
	log.Printf("start watching signals (pid: %d)", pid)
	ctx, cancel := context.WithTimeout(context.Background(), appTimeout)
	defer cancel()
	//as an example:
	//sigHandlers[syscall.SIGINT] = sigHandler(cancel)

	sysSigs := GetSystemSignals()
	signalChan := make(chan os.Signal, len(sysSigs))
	defer close(signalChan)
	signal.Notify(signalChan) // syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go runSignalHandlers(ctx, signalChan)

	log.Println("test all signals (!9 !19)") // +23 SIGSTOP/SIGCONT ...
	for _, sig := range sysSigs {
		if sig == 9 || sig == 19 {
			continue
		} // not to terminate this process
		log.Printf("-> Sending '%v' (%#v)...\n", sig, sig)
		if err := syscall.Kill(pid, sig); err != nil {
			log.Printf("stop sending signal: %v", err)
			break
		}
		time.Sleep(sigGenInterval)
	}
	<-ctx.Done()
}

// Linux xxx 5.8.0-63-generic #71-Ubuntu SMP Tue Jul 13 15:59:12 UTC 2021 x86_64 x86_64 x86_64 GNU/Linux

//kill -l:
//1) SIGHUP	 2) SIGINT	 3) SIGQUIT	 4) SIGILL	 5) SIGTRAP
//6) SIGABRT	 7) SIGBUS	 8) SIGFPE	 9) SIGKILL	10) SIGUSR1
//11) SIGSEGV	12) SIGUSR2	13) SIGPIPE	14) SIGALRM	15) SIGTERM
//16) SIGSTKFLT	17) SIGCHLD	18) SIGCONT	19) SIGSTOP	20) SIGTSTP
//21) SIGTTIN	22) SIGTTOU	23) SIGURG	24) SIGXCPU	25) SIGXFSZ
//26) SIGVTALRM	27) SIGPROF	28) SIGWINCH	29) SIGIO	30) SIGPWR
//31) SIGSYS	34) SIGRTMIN	35) SIGRTMIN+1	36) SIGRTMIN+2	37) SIGRTMIN+3
//38) SIGRTMIN+4	39) SIGRTMIN+5	40) SIGRTMIN+6	41) SIGRTMIN+7	42) SIGRTMIN+8
//43) SIGRTMIN+9	44) SIGRTMIN+10	45) SIGRTMIN+11	46) SIGRTMIN+12	47) SIGRTMIN+13
//48) SIGRTMIN+14	49) SIGRTMIN+15	50) SIGRTMAX-14	51) SIGRTMAX-13	52) SIGRTMAX-12
//53) SIGRTMAX-11	54) SIGRTMAX-10	55) SIGRTMAX-9	56) SIGRTMAX-8	57) SIGRTMAX-7
//58) SIGRTMAX-6	59) SIGRTMAX-5	60) SIGRTMAX-4	61) SIGRTMAX-3	62) SIGRTMAX-2
//63) SIGRTMAX-1	64) SIGRTMAX

// unix.SignalName:
//SIGHUP (#1) - hangup
//SIGINT (#2) - interrupt
//SIGQUIT (#3) - quit
//SIGILL (#4) - illegal instruction
//SIGTRAP (#5) - trace/breakpoint trap
//SIGABRT (#6) - aborted
//SIGBUS (#7) - bus error
//SIGFPE (#8) - floating point exception
//SIGKILL (#9) - killed
//SIGUSR1 (#10) - user defined signal 1
//SIGSEGV (#11) - segmentation fault
//SIGUSR2 (#12) - user defined signal 2
//SIGPIPE (#13) - broken pipe
//SIGALRM (#14) - alarm clock
//SIGTERM (#15) - terminated
//SIGSTKFLT (#16) - stack fault
//SIGCHLD (#17) - child exited
//SIGCONT (#18) - continued
//SIGSTOP (#19) - stopped (signal)
//SIGTSTP (#20) - stopped
//SIGTTIN (#21) - stopped (tty input)
//SIGTTOU (#22) - stopped (tty output)
//SIGURG (#23) - urgent I/O condition
//SIGXCPU (#24) - CPU time limit exceeded
//SIGXFSZ (#25) - file size limit exceeded
//SIGVTALRM (#26) - virtual timer expired
//SIGPROF (#27) - profiling timer expired
//SIGWINCH (#28) - window changed
//SIGIO (#29) - I/O possible
//SIGPWR (#30) - power failure
//SIGSYS (#31) - bad system call

// /usr/local/go/src/syscall/zerrors_linux_amd64.go:
//// Signals
//const (
//	SIGABRT   = Signal(0x6)
//	SIGALRM   = Signal(0xe)
//	SIGBUS    = Signal(0x7)
//	SIGCHLD   = Signal(0x11)
//	SIGCLD    = Signal(0x11)
//	SIGCONT   = Signal(0x12)
//	SIGFPE    = Signal(0x8)
//	SIGHUP    = Signal(0x1)
//	SIGILL    = Signal(0x4)
//	SIGINT    = Signal(0x2)
//	SIGIO     = Signal(0x1d)
//	SIGIOT    = Signal(0x6)
//	SIGKILL   = Signal(0x9)
//	SIGPIPE   = Signal(0xd)
//	SIGPOLL   = Signal(0x1d)
//	SIGPROF   = Signal(0x1b)
//	SIGPWR    = Signal(0x1e)
//	SIGQUIT   = Signal(0x3)
//	SIGSEGV   = Signal(0xb)
//	SIGSTKFLT = Signal(0x10)
//	SIGSTOP   = Signal(0x13)
//	SIGSYS    = Signal(0x1f)
//	SIGTERM   = Signal(0xf)
//	SIGTRAP   = Signal(0x5)
//	SIGTSTP   = Signal(0x14)
//	SIGTTIN   = Signal(0x15)
//	SIGTTOU   = Signal(0x16)
//	SIGUNUSED = Signal(0x1f)
//	SIGURG    = Signal(0x17)
//	SIGUSR1   = Signal(0xa)
//	SIGUSR2   = Signal(0xc)
//	SIGVTALRM = Signal(0x1a)
//	SIGWINCH  = Signal(0x1c)
//	SIGXCPU   = Signal(0x18)
//	SIGXFSZ   = Signal(0x19)
//)

//// Signal table
//var signals = [...]string{
//	1:  "hangup",
//	2:  "interrupt",
//	3:  "quit",
//	4:  "illegal instruction",
//	5:  "trace/breakpoint trap",
//	6:  "aborted",
//	7:  "bus error",
//	8:  "floating point exception",
//	9:  "killed",
//	10: "user defined signal 1",
//	11: "segmentation fault",
//	12: "user defined signal 2",
//	13: "broken pipe",
//	14: "alarm clock",
//	15: "terminated",
//	16: "stack fault",
//	17: "child exited",
//	18: "continued",
//	19: "stopped (signal)",
//	20: "stopped",
//	21: "stopped (tty input)",
//	22: "stopped (tty output)",
//	23: "urgent I/O condition",
//	24: "CPU time limit exceeded",
//	25: "file size limit exceeded",
//	26: "virtual timer expired",
//	27: "profiling timer expired",
//	28: "window changed",
//	29: "I/O possible",
//	30: "power failure",
//	31: "bad system call",
//}
