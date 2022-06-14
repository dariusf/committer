package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/openzipkin/zipkin-go"
	zipkingrpc "github.com/openzipkin/zipkin-go/middleware/grpc"
	log "github.com/sirupsen/logrus"
	"github.com/vadiminshakov/committer/cache"
	"github.com/vadiminshakov/committer/config"
	"github.com/vadiminshakov/committer/db"
	"github.com/vadiminshakov/committer/peer"
	pb "github.com/vadiminshakov/committer/proto"
	"github.com/vadiminshakov/committer/rvc"
	"github.com/vadiminshakov/committer/rvp"
	"github.com/vadiminshakov/committer/trace"
	"google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	status "google.golang.org/grpc/status"
)

type Option func(server *Server) error

const (
	TWO_PHASE   = "two-phase"
	THREE_PHASE = "three-phase"
)

// Server holds server instance, node config and connections to followers (if it's a coordinator node)
type Server struct {
	pb.UnimplementedCommitServer
	Addr                 string
	Followers            []*peer.CommitClient
	Config               *config.Config
	GRPCServer           *grpc.Server
	DB                   db.Database
	DBPath               string
	ProposeHook          func(req *pb.ProposeRequest) bool
	CommitHook           func(req *pb.CommitRequest) bool
	NodeCache            *cache.Cache
	Height               uint64
	cancelCommitOnHeight map[uint64]bool
	mu                   sync.RWMutex
	Tracer               *zipkin.Tracer
	mParts               map[string]bool
	MonitorC             *rvc.Monitor
	MonitorP             *rvp.Monitor
}

var f *os.File = nil

func writeToJson(data map[string]interface{}) {
	str, err := json.Marshal(data)
	if err != nil {
		panic(err)
		// log.Println(err)
		// return
	}
	if f == nil {
		f1, err := os.OpenFile("log.ndjson", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		f = f1
		if err != nil {
			panic(err)
			// log.Println(err)
			// return
		}
	}
	// defer f.Close()
	if _, err = f.WriteString(string(str) + "\n"); err != nil {
		panic(err)
	}
}

func (s *Server) serverToJson(action string) {
	writeToJson(map[string]interface{}{
		"Role":        s.Config.Role,
		"Nodeaddr":    s.Config.Nodeaddr,
		"Coordinator": s.Config.Coordinator,
		"Followers":   s.Config.Followers,
		"Whitelist":   s.Config.Whitelist,
		"CommitType":  s.Config.CommitType,
		"Timeout":     s.Config.Timeout,
		"DBPath":      s.Config.DBPath,
		"WithTrace":   s.Config.WithTrace,
		"_name":       action,
		"_type":       "internal",
	})
}

func (s *Server) Propose(ctx context.Context, req *pb.ProposeRequest) (*pb.Response, error) {
	var span zipkin.Span
	if s.Tracer != nil {
		span, ctx = s.Tracer.StartSpanFromContext(ctx, "ProposeHandle")
		defer span.Finish()
	}
	s.SetProgressForCommitPhase(req.Index, false)
	return s.ProposeHandler(ctx, req, s.ProposeHook)
}

func (s *Server) Precommit(ctx context.Context, req *pb.PrecommitRequest) (*pb.Response, error) {
	var span zipkin.Span
	if s.Tracer != nil {
		span, _ = s.Tracer.StartSpanFromContext(ctx, "PrecommitHandle")
		defer span.Finish()
	}
	if s.Config.CommitType == THREE_PHASE {
		ctx, _ = context.WithTimeout(context.Background(), time.Duration(s.Config.Timeout)*time.Millisecond)
		go func(ctx context.Context) {
		ForLoop:
			for {
				select {
				case <-ctx.Done():
					md := metadata.Pairs("mode", "autocommit")
					ctx := metadata.NewOutgoingContext(context.Background(), md)
					if !s.HasProgressForCommitPhase(req.Index) {
						s.Commit(ctx, &pb.CommitRequest{Index: s.Height})
						log.Warn("commit without coordinator after timeout")
					}
					break ForLoop
				}
			}
		}(ctx)
	}
	return s.PrecommitHandler(ctx, req)
}

func (s *Server) Commit(ctx context.Context, req *pb.CommitRequest) (resp *pb.Response, err error) {
	var span zipkin.Span
	if s.Tracer != nil {
		span, ctx = s.Tracer.StartSpanFromContext(ctx, "CommitHandle")
		defer span.Finish()
	}

	if s.Config.CommitType == THREE_PHASE {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Internal, "no metadata")
		}

		meta := md["mode"]

		if len(meta) == 0 {
			s.SetProgressForCommitPhase(s.Height, true) // set flag for cancelling 'commit without coordinator' action, coz coordinator responded actually
			resp, err = s.CommitHandler(ctx, req, s.CommitHook, s.DB)
			if err != nil {
				return nil, err
			}
			if resp.Type == pb.Type_ACK {
				atomic.AddUint64(&s.Height, 1)
			}
			return
		}

		if req.IsRollback {
			s.rollback()
		}
	} else {

		resp, err = s.CommitHandler(ctx, req, s.CommitHook, s.DB)
		if err != nil {
			return
		}

		if resp.Type == pb.Type_ACK {
			atomic.AddUint64(&s.Height, 1)
		}
		return
	}
	return
}

func (s *Server) Get(ctx context.Context, req *pb.Msg) (*pb.Value, error) {
	var span zipkin.Span
	if s.Tracer != nil {
		span, ctx = s.Tracer.StartSpanFromContext(ctx, "GetHandle")
		defer span.Finish()
	}

	value, err := s.DB.Get(req.Key)
	if err != nil {
		return nil, err
	}
	return &pb.Value{Value: value}, nil
}

func (s *Server) Put(ctx context.Context, req *pb.Entry) (*pb.Response, error) {
	var (
		response *pb.Response
		err      error
		span     zipkin.Span
	)

	if s.Tracer != nil {
		span, ctx = s.Tracer.StartSpanFromContext(ctx, "PutHandle")
		defer span.Finish()
	}

	var ctype pb.CommitType
	if s.Config.CommitType == THREE_PHASE {
		ctype = pb.CommitType_THREE_PHASE_COMMIT
	} else {
		ctype = pb.CommitType_TWO_PHASE_COMMIT
	}

	g := rvc.Global{HasAborted: false, Committed: map[string]bool{}, Aborted: map[string]bool{}}

	// propose
	log.Infof("propose key %s", req.Key)
	for pid, follower := range s.Followers {
		if s.Config.CommitType == THREE_PHASE {
			ctx, _ = context.WithTimeout(ctx, time.Duration(s.Config.Timeout)*time.Millisecond)
		}
		if s.Tracer != nil {
			span, ctx = s.Tracer.StartSpanFromContext(ctx, "Propose")
		}
		var (
			response *pb.Response
			err      error
		)
		for response == nil || response != nil && response.Type == pb.Type_NACK {
			writeToJson(map[string]interface{}{
				"_type":      "send",
				"_name":      "Propose",
				"Key":        req.Key,
				"Value":      req.Value,
				"CommitType": ctype,
				"Index":      s.Height,
			})
			response, err = follower.Propose(ctx, &pb.ProposeRequest{Key: req.Key,
				Value:      req.Value,
				CommitType: ctype,
				Index:      s.Height})
			writeToJson(map[string]interface{}{
				"_type": "receive",
				"_name": "Propose",
				"Type":  response.Type,
				"Index": response.Index,
			})
			if s.Tracer != nil && span != nil {
				span.Finish()
			}
			s.serverToJson("CSendPrepare8")
			if err := s.MonitorC.Step(g, rvc.CSendPrepare8, strconv.Itoa(pid)); err != nil {
				log.Printf("%v\n", err)
			}
			if response != nil && response.Index > s.Height {
				log.Warnf("сoordinator has stale height [%d], update to [%d] and try to send again", s.Height, response.Index)
				s.Height = response.Index
			}
		}
		if err != nil {
			log.Errorf(err.Error())
			s.serverToJson("CReceiveAbort10")
			if err := s.MonitorC.Step(g, rvc.CReceiveAbort10, strconv.Itoa(pid)); err != nil {
				log.Printf("%v\n", err)
			}
			g.HasAborted = true
			return &pb.Response{Type: pb.Type_NACK}, nil
		}
		s.NodeCache.Set(s.Height, req.Key, req.Value)
		if response.Type != pb.Type_ACK {
			return nil, status.Error(codes.Internal, "follower not acknowledged msg")
		}
		s.serverToJson("CReceivePrepared9")
		if err := s.MonitorC.Step(g, rvc.CReceivePrepared9, strconv.Itoa(pid)); err != nil {
			log.Printf("%v\n", err)
		}
	}

	// precommit phase only for three-phase mode
	log.Infof("precommit key %s", req.Key)
	for _, follower := range s.Followers {
		if s.Config.CommitType == THREE_PHASE {
			ctx, _ = context.WithTimeout(ctx, time.Duration(s.Config.Timeout)*time.Millisecond)
		}
		if s.Tracer != nil {
			span, ctx = s.Tracer.StartSpanFromContext(ctx, "Precommit")
		}
		response, err = follower.Precommit(ctx, &pb.PrecommitRequest{Index: s.Height})
		if s.Tracer != nil && span != nil {
			span.Finish()
		}
		if err != nil {
			log.Errorf(err.Error())
			return &pb.Response{Type: pb.Type_NACK}, nil
		}
		if response.Type != pb.Type_ACK {
			return nil, status.Error(codes.Internal, "follower not acknowledged msg")
		}
	}

	// the coordinator got all the answers, so it's time to persist msg and send commit command to followers
	key, value, ok := s.NodeCache.Get(s.Height)
	if !ok {
		return nil, status.Error(codes.Internal, "can't to find msg in the coordinator's cache")
	}
	if err = s.DB.Put(key, value); err != nil {
		for pid := range s.Followers {
			g.Aborted[strconv.Itoa(pid)] = true
		}
		for pid := range s.Followers {
			s.serverToJson("CSendAbort13")
			if err := s.MonitorC.Step(g, rvc.CSendAbort13, strconv.Itoa(pid)); err != nil {
				log.Printf("%v\n", err)
			}
			s.serverToJson("CReceiveAbortAck14")
			if err := s.MonitorC.Step(g, rvc.CReceiveAbortAck14, strconv.Itoa(pid)); err != nil {
				log.Printf("%v\n", err)
			}
		}
		return &pb.Response{Type: pb.Type_NACK}, status.Error(codes.Internal, "failed to save msg on coordinator")
	}

	// commit
	log.Infof("commit %s", req.Key)
	for pid, follower := range s.Followers {
		if s.Tracer != nil {
			span, ctx = s.Tracer.StartSpanFromContext(ctx, "Commit")
		}
		writeToJson(map[string]interface{}{
			"_type": "send",
			"_name": "Commit",
			"Index": s.Height,
		})
		response, err = follower.Commit(ctx, &pb.CommitRequest{Index: s.Height})
		writeToJson(map[string]interface{}{
			"_type": "receive",
			"_name": "Commit",
			"Type":  response.Type,
			"Index": response.Index,
		})
		if s.Tracer != nil && span != nil {
			span.Finish()
		}
		s.serverToJson("CSendCommit11")
		if err := s.MonitorC.Step(g, rvc.CSendCommit11, strconv.Itoa(pid)); err != nil {
			log.Printf("%v\n", err)
		}
		if err != nil {
			log.Errorf(err.Error())
			return &pb.Response{Type: pb.Type_NACK}, nil
		}
		if response.Type != pb.Type_ACK {
			return nil, status.Error(codes.Internal, "follower not acknowledged msg")
		}
		s.serverToJson("CReceiveCommitAck12")
		if err := s.MonitorC.Step(g, rvc.CReceiveCommitAck12, strconv.Itoa(pid)); err != nil {
			log.Printf("%v\n", err)
		}
		g.Committed[strconv.Itoa(pid)] = true
	}
	log.Infof("committed key %s", req.Key)
	// increase height for next round
	atomic.AddUint64(&s.Height, 1)

	s.MonitorC.Reset()

	return &pb.Response{Type: pb.Type_ACK}, nil
}

func (s *Server) NodeInfo(ctx context.Context, req *empty.Empty) (*pb.Info, error) {
	var span zipkin.Span
	if s.Tracer != nil {
		span, ctx = s.Tracer.StartSpanFromContext(ctx, "NodeInfoHandle")
		defer span.Finish()
	}

	return &pb.Info{Height: s.Height}, nil
}

// NewCommitServer fabric func for Server
func NewCommitServer(conf *config.Config, opts ...Option) (*Server, error) {
	log.SetFormatter(&log.TextFormatter{
		ForceColors:     true, // Seems like automatic color detection doesn't work on windows terminals
		FullTimestamp:   true,
		TimestampFormat: time.RFC822,
	})

	mParts := map[string]bool{}
	server := &Server{Addr: conf.Nodeaddr, DBPath: conf.DBPath,
		mParts: mParts,
	}
	var err error
	for _, option := range opts {
		err = option(server)
		if err != nil {
			return nil, err
		}
	}

	if conf.WithTrace {
		// get Zipkin tracer
		server.Tracer, err = trace.Tracer(fmt.Sprintf("%s:%s", conf.Role, conf.Nodeaddr), conf.Nodeaddr)
		if err != nil {
			return nil, err
		}
	}

	for pid, node := range conf.Followers {
		cli, err := peer.New(node, server.Tracer)
		if err != nil {
			return nil, err
		}
		mParts[strconv.Itoa(pid)] = true
		server.Followers = append(server.Followers, cli)
	}

	server.Config = conf
	if conf.Role == "coordinator" {
		server.Config.Coordinator = server.Addr
		server.MonitorC = rvc.NewMonitor(map[string]map[string]bool{"P": mParts})
	} else {
		server.MonitorP = rvp.NewMonitor(map[string]map[string]bool{"C": {"c": true}})
	}

	server.DB, err = db.New(conf.DBPath)
	if err != nil {
		return nil, err
	}
	server.NodeCache = cache.New()
	server.cancelCommitOnHeight = map[uint64]bool{}

	if server.Config.CommitType == TWO_PHASE {
		log.Info("two-phase-commit mode enabled")
	} else {
		log.Info("three-phase-commit mode enabled")
	}
	err = checkServerFields(server)
	return server, err
}

// WithBadgerDB adds BadgerDB manager to the Server instance
func WithBadgerDB(path string) func(*Server) error {
	return func(server *Server) error {
		var err error
		server.DB, err = db.New(path)
		return err
	}
}

func checkServerFields(server *Server) error {
	if server.DB == nil {
		return errors.New("database is not selected")
	}
	return nil
}

// Run starts non-blocking GRPC server
func (s *Server) Run(opts ...grpc.UnaryServerInterceptor) {
	var err error

	if s.Config.WithTrace {
		s.GRPCServer = grpc.NewServer(grpc.ChainUnaryInterceptor(opts...), grpc.StatsHandler(zipkingrpc.NewServerHandler(s.Tracer)))
	} else {
		s.GRPCServer = grpc.NewServer(grpc.ChainUnaryInterceptor(opts...))
	}
	pb.RegisterCommitServer(s.GRPCServer, s)

	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Infof("listening on tcp://%s", s.Addr)

	go s.GRPCServer.Serve(l)
}

// Stop stops server
func (s *Server) Stop() {
	log.Info("stopping server")
	s.GRPCServer.GracefulStop()
	if err := s.DB.Close(); err != nil {
		log.Infof("failed to close db, err: %s\n", err)
	}
	log.Info("server stopped")
}

func (s *Server) rollback() {
	s.NodeCache.Delete(s.Height)
}

func (s *Server) SetProgressForCommitPhase(height uint64, docancel bool) {
	s.mu.Lock()
	s.cancelCommitOnHeight[height] = docancel
	s.mu.Unlock()
}

func (s *Server) HasProgressForCommitPhase(height uint64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cancelCommitOnHeight[height]
}
