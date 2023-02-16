package server

import (
	"context"
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/vadiminshakov/committer/db"
	pb "github.com/vadiminshakov/committer/proto"
)

func (s *Server) ProposeHandler(ctx context.Context, req *pb.ProposeRequest, hook func(req *pb.ProposeRequest) bool) (*pb.Response, error) {
	// if err := s.MonitorP.StepA(rvp.PReceivePrepare1, "c"); err != nil {
	// 	log.Printf("%v\n", err)
	// }
	var response *pb.Response
	if hook(req) {
		log.Infof("received: %s=%s\n", req.Key, string(req.Value))
		s.NodeCache.Set(req.Index, req.Key, req.Value)
		response = &pb.Response{Type: pb.Type_ACK, Index: req.Index}
		// if err := s.MonitorP.StepA(rvp.PSendPrepared2, "c"); err != nil {
		// 	log.Printf("%v\n", err)
		// }
	} else {
		response = &pb.Response{Type: pb.Type_NACK, Index: req.Index}
		// if err := s.MonitorP.StepA(rvp.PSendAbort3, "c"); err != nil {
		// 	log.Printf("%v\n", err)
		// }
	}
	if s.Height > req.Index {
		response = &pb.Response{Type: pb.Type_NACK, Index: s.Height}
		// if err := s.MonitorP.StepA(rvp.PSendAbort3, "c"); err != nil {
		// 	log.Printf("%v\n", err)
		// }
	}
	return response, nil
}

func (s *Server) PrecommitHandler(ctx context.Context, req *pb.PrecommitRequest) (*pb.Response, error) {
	return &pb.Response{Type: pb.Type_ACK}, nil
}

func (s *Server) CommitHandler(ctx context.Context, req *pb.CommitRequest, hook func(req *pb.CommitRequest) bool, db db.Database) (*pb.Response, error) {
	var response *pb.Response
	// if err := s.MonitorP.StepA(rvp.PReceiveCommit4, "c"); err != nil {
	// 	log.Printf("%v\n", err)
	// }
	if hook(req) {
		log.Printf("Committing on height: %d\n", req.Index)
		key, value, ok := s.NodeCache.Get(req.Index)
		if !ok {
			s.NodeCache.Delete(req.Index)
			return &pb.Response{Type: pb.Type_NACK}, errors.New(fmt.Sprintf("no value in node cache on the index %d", req.Index))
		}

		if err := db.Put(key, value); err != nil {
			return nil, err
		}
		// if err := s.MonitorP.StepA(rvp.PSendCommitAck5, "c"); err != nil {
		// 	log.Printf("%v\n", err)
		// }
		response = &pb.Response{Type: pb.Type_ACK}
	} else {
		s.NodeCache.Delete(req.Index)
		// if err := s.MonitorP.StepA(rvp.PSendAbortAck7, "c"); err != nil {
		// 	log.Printf("%v\n", err)
		// }
		response = &pb.Response{Type: pb.Type_NACK}
	}
	// s.MonitorP.Reset()
	return response, nil
}
