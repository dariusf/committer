package server

import (
	"fmt"

	pb "github.com/vadiminshakov/committer/proto"
)

func RefineParty(pid int) TLA {
	return str(fmt.Sprintf("r%d", pid+1))
}

func RefineProposeMsg(pid int, _ *pb.ProposeRequest) TLA {
	// all the fields of the propose are irrelevant
	return record(
		str("type"), str("Prepare"),
		str("msource"), str("coordinator"),
		str("mdest"), RefineParty(pid),
	)
}

func RefineProposeReply(pid int) TLA {
	return record(
		str("type"), str("Prepared"),
		str("mdest"), str("coordinator"),
		str("msource"), RefineParty(pid),
	)
}

func RefineAbortedMsg(pid int) TLA {
	return record(
		str("type"), str("Aborted"),
		str("mdest"), str("coordinator"),
		str("msource"), RefineParty(pid),
	)
}

func RefineCommitMsg(pid int) TLA {
	return record(
		str("type"), str("Commit"),
		str("msource"), str("coordinator"),
		str("mdest"), RefineParty(pid),
	)
}

func RefineCommittedMsg(pid int) TLA {
	return record(
		str("type"), str("Committed"),
		str("mdest"), str("coordinator"),
		str("msource"), RefineParty(pid),
	)
}

func (s *Server) Starting() State {
	return State{
		who:      str("none"),
		actions:  seq(),
		inflight: seq(),
		inbox: record(
			str("coordinator"), seq(),
			str("r1"), seq(),
			str("r2"), seq()),
		outbox: record(
			str("coordinator"), seq(),
			str("r1"), seq(),
			str("r2"), seq()),
		tmCommitted: seq(),
		tmPrepared:  seq(),
		tmAborted:   seq(),
		tmDecision:  str("none"),
		rmState: record(
			str("r1"), str("working"),
			str("r2"), str("working")),
	}
}
